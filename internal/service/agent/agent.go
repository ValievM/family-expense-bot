package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

const (
	defaultMaxSteps = 5
	emptyReply      = "Не удалось сформулировать ответ, попробуйте переформулировать запрос."
)

// ErrStepLimit сообщает, что модель не дала финальный ответ за допустимое число вызовов инструментов.
var ErrStepLimit = errors.New("agent exceeded the tool-call step limit")

// ScopeFactory выдаёт траты, жёстко привязанные к одному Telegram-пользователю; владелец берётся только из кода.
type ScopeFactory func(userID int64) UserExpenses

// Clock отдаёт текущее время, чтобы промпт и тесты были детерминированными.
type Clock interface {
	// Принимает: ничего.
	// Делает: читает текущее время.
	// Возвращает: момент времени.
	Now() time.Time
}

// systemClock — Clock на системных часах.
type systemClock struct{}

// Принимает: ничего.
// Делает: читает системные часы.
// Возвращает: текущее время.
func (systemClock) Now() time.Time {
	return time.Now()
}

// Agent превращает свободный текст пользователя в вызовы инструментов учёта и короткий ответ.
type Agent struct {
	llm      LLM
	scope    ScopeFactory
	tools    *ToolSet
	history  History
	clock    Clock
	location *time.Location
	maxSteps int
	logger   *slog.Logger
}

// Option настраивает Agent при создании.
type Option func(*Agent)

// Принимает: clock — источник времени.
// Делает: подменяет часы агента.
// Возвращает: Option.
func WithClock(clock Clock) Option {
	return func(agent *Agent) { agent.clock = clock }
}

// Принимает: location — часовой пояс календаря пользователей.
// Делает: задаёт часовой пояс для дат в промпте и инструментах и пересобирает стандартные инструменты под него.
// Возвращает: Option.
func WithLocation(location *time.Location) Option {
	return func(agent *Agent) {
		agent.location = location
		agent.tools = DefaultTools(location)
	}
}

// Принимает: history — хранилище памяти диалога.
// Делает: заменяет хранилище истории.
// Возвращает: Option.
func WithHistory(history History) Option {
	return func(agent *Agent) { agent.history = history }
}

// Принимает: steps — максимум раундов с вызовом инструментов за одно сообщение.
// Делает: меняет лимит шагов, если значение положительное.
// Возвращает: Option.
func WithMaxSteps(steps int) Option {
	return func(agent *Agent) {
		if steps > 0 {
			agent.maxSteps = steps
		}
	}
}

// Принимает: logger — логгер агента.
// Делает: заменяет логгер.
// Возвращает: Option.
func WithLogger(logger *slog.Logger) Option {
	return func(agent *Agent) { agent.logger = logger }
}

// Принимает: llm — провайдер модели; scope — фабрика трат по владельцу; options — необязательные настройки.
// Делает: собирает агента со стандартными инструментами, UTC, памятью на 8 сообщений и лимитом в 5 шагов.
// Возвращает: готового агента.
func New(llm LLM, scope ScopeFactory, options ...Option) *Agent {
	agent := &Agent{
		llm:      llm,
		scope:    scope,
		clock:    systemClock{},
		location: time.UTC,
		history:  NewMemoryHistory(0),
		maxSteps: defaultMaxSteps,
		logger:   slog.Default(),
	}
	agent.tools = DefaultTools(agent.location)
	for _, option := range options {
		option(agent)
	}
	return agent
}

// Принимает: ctx — контекст запроса; userID — Telegram ID автора сообщения; text — текст сообщения.
// Делает: гоняет цикл «модель → инструменты → модель», где владельцем данных всегда служит userID, и запоминает реплику в истории.
// Возвращает: итоговый ответ для пользователя либо ошибку LLM, ErrStepLimit или сбой инфраструктуры.
func (agent *Agent) Handle(ctx context.Context, userID int64, text string) (string, error) {
	expenses := agent.scope(userID)
	userMessage := Message{Role: RoleUser, Content: text}

	messages := []Message{{Role: RoleSystem, Content: buildSystemPrompt(agent.clock.Now(), agent.location)}}
	messages = append(messages, agent.history.Recent(userID)...)
	messages = append(messages, userMessage)

	for range agent.maxSteps {
		reply, err := agent.llm.Complete(ctx, Request{Messages: messages, Tools: agent.tools.Specs()})
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
			messages = append(messages, Message{Role: RoleTool, ToolCallID: call.ID, Content: result})
		}
	}
	return "", ErrStepLimit
}

// Принимает: userID — владелец диалога; userMessage — исходное сообщение; answer — ответ модели.
// Делает: подставляет запасной текст на пустой ответ и сохраняет пару «вопрос — ответ» в истории.
// Возвращает: текст ответа пользователю.
func (agent *Agent) finish(userID int64, userMessage Message, answer string) string {
	if answer == "" {
		answer = emptyReply
	}
	agent.history.Remember(userID, userMessage, Message{Role: RoleAssistant, Content: answer})
	return answer
}

// Принимает: ctx — контекст запроса; userID — владелец, нужен только для логов; expenses — траты владельца; call — вызов от модели.
// Делает: находит инструмент и выполняет его; ошибки, которые модель может исправить, превращает в ответ-инструкцию.
// Возвращает: JSON-строку для модели либо ошибку, если сбой не связан с аргументами модели.
func (agent *Agent) runTool(ctx context.Context, userID int64, expenses UserExpenses, call ToolCall) (string, error) {
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
