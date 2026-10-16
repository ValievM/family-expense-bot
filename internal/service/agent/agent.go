// Package agent — ядро ИИ-агента учёта расходов: цикл «модель → инструменты → модель», память диалога и системный промпт.
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

// ErrStepLimit сообщает, что модель не дала финальный ответ за допустимое число вызовов инструментов.
var ErrStepLimit = errors.New("agent exceeded the tool-call step limit")

// ScopeFactory выдаёт траты, жёстко привязанные к одному Telegram-пользователю; владелец берётся только из кода.
type ScopeFactory func(owner domain.UserID) ports.UserExpenses

// Agent превращает свободный текст пользователя в вызовы инструментов учёта и короткий ответ.
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

// Option настраивает Agent при создании.
type Option func(*Agent)

// Принимает: location — часовой пояс календаря пользователей.
// Делает: задаёт часовой пояс, в котором считаются «сегодня» и «вчера» в промпте.
// Возвращает: Option.
func WithLocation(location *time.Location) Option {
	return func(agent *Agent) { agent.location = location }
}

// Принимает: logger — логгер агента.
// Делает: заменяет логгер.
// Возвращает: Option.
func WithLogger(logger *slog.Logger) Option {
	return func(agent *Agent) { agent.logger = logger }
}

// Принимает: llm — провайдер модели; scope — фабрика трат по владельцу; tools — набор инструментов; clock — источник времени для промпта; options — необязательные настройки.
// Делает: собирает агента с UTC, памятью на 8 сообщений и лимитом в 5 шагов.
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

// Принимает: ctx — контекст запроса; userID — Telegram ID автора сообщения; text — текст сообщения.
// Делает: гоняет цикл «модель → инструменты → модель», где владельцем данных всегда служит userID, и запоминает реплику в истории.
// Возвращает: итоговый ответ для пользователя либо ошибку LLM, ErrStepLimit или сбой инфраструктуры.
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

// Принимает: userID — владелец диалога; userMessage — исходное сообщение; answer — ответ модели.
// Делает: подставляет запасной текст на пустой ответ и сохраняет пару «вопрос — ответ» в истории.
// Возвращает: текст ответа пользователю.
func (agent *Agent) finish(userID int64, userMessage ports.Message, answer string) string {
	if answer == "" {
		answer = emptyReply
	}
	agent.history.Remember(userID, userMessage, ports.Message{Role: ports.RoleAssistant, Content: answer})
	return answer
}

// Принимает: ctx — контекст запроса; userID — владелец, нужен только для логов; expenses — траты владельца; call — вызов от модели.
// Делает: находит инструмент и выполняет его; ошибки, которые модель может исправить, превращает в ответ-инструкцию.
// Возвращает: JSON-строку для модели либо ошибку, если сбой не связан с аргументами модели.
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

// Принимает: err — ошибка, которую модель должна увидеть.
// Делает: оборачивает текст ошибки в JSON {"error": ...}.
// Возвращает: строку для сообщения роли tool.
func errorResult(err error) string {
	encoded, marshalErr := json.Marshal(map[string]string{"error": err.Error()})
	if marshalErr != nil {
		return `{"error":"internal"}`
	}
	return string(encoded)
}
