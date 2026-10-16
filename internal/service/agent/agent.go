// Package agent — ядро ИИ-агента: цикл «модель → инструменты → модель», память диалога, промпт.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/ports"
)

const (
	maxSteps   = 5
	emptyReply = "Не удалось сформулировать ответ, попробуйте переформулировать запрос."
)

var ErrStepLimit = errors.New("agent exceeded the tool-call step limit")

// ScopeFactory выдаёт траты одного пользователя; владелец берётся только из кода, не от модели.
type ScopeFactory func(owner domain.UserID) ports.UserExpenses

type Agent struct {
	llm      ports.LLM
	scope    ScopeFactory
	tools    *ToolSet
	history  ports.History
	clock    ports.Clock
	location *time.Location
	maxSteps int // лимит раундов с инструментами за одно сообщение
	logger   *slog.Logger
}

type Option func(*Agent)

func WithLocation(location *time.Location) Option {
	return func(agent *Agent) { agent.location = location }
}

func WithLogger(logger *slog.Logger) Option {
	return func(agent *Agent) { agent.logger = logger }
}

// Принимает: llm, scope, tools, clock; options — необязательные настройки.
// Делает: собирает агента: UTC, память на 8 сообщений, лимит 5 шагов.
// Возвращает: готового агента.
func New(llm ports.LLM, scope ScopeFactory, tools *ToolSet, clock ports.Clock, options ...Option) *Agent {
	agent := &Agent{
		llm:      llm,
		scope:    scope,
		tools:    tools,
		clock:    clock,
		location: time.UTC,
		history:  NewMemoryHistory(0),
		maxSteps: maxSteps,
		logger:   slog.Default(),
	}
	for _, option := range options {
		option(agent)
	}
	return agent
}

// Принимает: ctx; userID — Telegram ID автора; text — сообщение.
// Делает: гоняет цикл «модель → инструменты → модель» с владельцем userID, пишет историю.
// Возвращает: ответ пользователю либо ошибку LLM, ErrStepLimit или инфраструктуры.
func (agent *Agent) Handle(ctx context.Context, userID int64, text string) (string, error) {
	owner, err := domain.NewUserID(userID)
	if err != nil {
		return "", fmt.Errorf("identify message author: %w", err)
	}
	expenses := agent.scope(owner)
	userMessage := ports.Message{Role: ports.RoleUser, Content: text}

	messages := []ports.Message{{Role: ports.RoleSystem, Content: buildSystemPrompt(agent.clock.Now(), agent.location)}}
	messages = append(messages, agent.history.Recent(userID)...)
	messages = append(messages, userMessage)

	for range agent.maxSteps {
		reply, err := agent.llm.Complete(ctx, ports.Request{Messages: messages, Tools: agent.tools.Specs()})
		if err != nil {
			return "", fmt.Errorf("complete conversation: %w", err)
		}
		if len(reply.ToolCalls) == 0 {
			return agent.finish(userID, userMessage, reply.Content), nil
		}

		messages = append(messages, reply)
		for _, call := range reply.ToolCalls {
			result, err := agent.runTool(ctx, userID, expenses, call)
			if err != nil {
				return "", err
			}
			messages = append(messages, ports.Message{Role: ports.RoleTool, ToolCallID: call.ID, Content: result})
		}
	}
	return "", ErrStepLimit
}

// Принимает: userID; userMessage; answer — ответ модели.
// Делает: подставляет запасной текст на пустой ответ и сохраняет пару в историю.
// Возвращает: текст ответа пользователю.
func (agent *Agent) finish(userID int64, userMessage ports.Message, answer string) string {
	if answer == "" {
		answer = emptyReply
	}
	agent.history.Remember(userID, userMessage, ports.Message{Role: ports.RoleAssistant, Content: answer})
	return answer
}

// Принимает: ctx; userID (для логов); expenses — траты владельца; call — вызов модели.
// Делает: выполняет инструмент; исправимые моделью ошибки отдаёт ей как ответ.
// Возвращает: JSON для модели либо ошибку, не связанную с аргументами модели.
func (agent *Agent) runTool(ctx context.Context, userID int64, expenses ports.UserExpenses, call ports.ToolCall) (string, error) {
	tool, found := agent.tools.Get(call.Name)
	if !found {
		return errorResult(fmt.Errorf("unknown tool %q", call.Name)), nil
	}

	agent.logger.InfoContext(ctx, "agent tool call", "user_id", userID, "tool", call.Name)
	output, err := tool.Execute(ctx, expenses, call.Arguments)
	if err != nil {
		if isRecoverable(err) {
			return errorResult(err), nil
		}
		return "", fmt.Errorf("execute tool %s: %w", call.Name, err)
	}

	encoded, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("encode %s result: %w", call.Name, err)
	}
	return string(encoded), nil
}

func errorResult(err error) string {
	encoded, marshalErr := json.Marshal(map[string]string{"error": err.Error()})
	if marshalErr != nil {
		return `{"error":"internal"}`
	}
	return string(encoded)
}
