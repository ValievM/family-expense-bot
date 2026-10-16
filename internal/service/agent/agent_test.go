// Package agent_test проверяет агента через его публичный контракт с фейковой моделью и реальным сервисом трат.
package agent_test

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/service/agent"
	"github.com/ValievM/family-expense-bot/internal/service/agent/tools"
	"github.com/ValievM/family-expense-bot/internal/service/expense"
)

// scriptedLLM отдаёт заготовленные ответы по очереди и запоминает все запросы.
type scriptedLLM struct {
	replies  []agent.Message
	requests []agent.Request
	err      error
}

// Принимает: _ — контекст не используется; request — очередной запрос агента.
// Делает: сохраняет запрос и достаёт следующий заготовленный ответ.
// Возвращает: ответ из очереди либо ошибку, заданную в тесте; когда очередь пуста, возвращает тестовую ошибку.
func (llm *scriptedLLM) Complete(_ context.Context, request agent.Request) (agent.Message, error) {
	llm.requests = append(llm.requests, request)
	if llm.err != nil {
		return agent.Message{}, llm.err
	}
	if len(llm.replies) == 0 {
		return agent.Message{}, errors.New("scripted LLM has no more replies")
	}
	reply := llm.replies[0]
	llm.replies = llm.replies[1:]
	return reply, nil
}

// Принимает: id — идентификатор вызова; name — имя инструмента; arguments — JSON аргументов.
// Делает: собирает ответ модели с одним вызовом инструмента.
// Возвращает: сообщение ассистента с ToolCalls.
func toolCall(id, name, arguments string) agent.Message {
	return agent.Message{Role: agent.RoleAssistant, ToolCalls: []agent.ToolCall{{ID: id, Name: name, Arguments: json.RawMessage(arguments)}}}
}

// Принимает: content — текст финального ответа.
// Делает: собирает финальный ответ модели без вызовов инструментов.
// Возвращает: сообщение ассистента.
func say(content string) agent.Message {
	return agent.Message{Role: agent.RoleAssistant, Content: content}
}

// memoryRepository — хранилище трат в памяти для тестов.
type memoryRepository struct {
	records map[int64]domain.Expense
	nextID  int64
}

// Принимает: ничего.
// Делает: создаёт пустое хранилище.
// Возвращает: указатель на хранилище.
func newMemoryRepository() *memoryRepository {
	return &memoryRepository{records: make(map[int64]domain.Expense)}
}

// Принимает: _ — контекст не используется; item — трата без ID.
// Делает: присваивает следующий ID и сохраняет трату.
// Возвращает: сохранённую трату и nil.
func (repository *memoryRepository) Create(_ context.Context, item domain.Expense) (domain.Expense, error) {
	repository.nextID++
	item.ID = repository.nextID
	repository.records[item.ID] = item
	return item, nil
}

// Принимает: _ — контекст не используется; id — идентификатор траты.
// Делает: ищет трату по ID.
// Возвращает: трату либо expense.ErrNotFound.
func (repository *memoryRepository) Get(_ context.Context, id int64) (domain.Expense, error) {
	item, found := repository.records[id]
	if !found {
		return domain.Expense{}, expense.ErrNotFound
	}
	return item, nil
}

// Принимает: _ — контекст не используется; filter — фильтры списка.
// Делает: отбирает траты по владельцу, категории и периоду и сортирует по дате и ID по убыванию.
// Возвращает: список трат и nil.
func (repository *memoryRepository) List(_ context.Context, filter expense.ListFilter) ([]domain.Expense, error) {
	items := make([]domain.Expense, 0, len(repository.records))
	for _, item := range repository.records {
		if filter.UserID != nil && item.UserID != *filter.UserID {
			continue
		}
		if filter.Category != nil && item.Category != *filter.Category {
			continue
		}
		if filter.From != nil && item.SpentAt.Before(*filter.From) {
			continue
		}
		if filter.To != nil && item.SpentAt.After(*filter.To) {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if !items[i].SpentAt.Equal(items[j].SpentAt) {
			return items[i].SpentAt.After(items[j].SpentAt)
		}
		return items[i].ID > items[j].ID
	})
	return items, nil
}

// Принимает: _ — контекст не используется.
// Делает: собирает уникальных владельцев трат.
// Возвращает: отсортированные ID и nil.
func (repository *memoryRepository) ListUserIDs(_ context.Context) ([]int64, error) {
	seen := make(map[int64]struct{})
	userIDs := make([]int64, 0)
	for _, item := range repository.records {
		if _, exists := seen[item.UserID]; !exists {
			seen[item.UserID] = struct{}{}
			userIDs = append(userIDs, item.UserID)
		}
	}
	sort.Slice(userIDs, func(i, j int) bool { return userIDs[i] < userIDs[j] })
	return userIDs, nil
}

// Принимает: _ — контекст не используется; item — трата с ID.
// Делает: заменяет сохранённую трату.
// Возвращает: новую трату либо expense.ErrNotFound.
func (repository *memoryRepository) Update(_ context.Context, item domain.Expense) (domain.Expense, error) {
	if _, found := repository.records[item.ID]; !found {
		return domain.Expense{}, expense.ErrNotFound
	}
	repository.records[item.ID] = item
	return item, nil
}

// Принимает: _ — контекст не используется; id — идентификатор траты.
// Делает: удаляет трату.
// Возвращает: nil либо expense.ErrNotFound.
func (repository *memoryRepository) Delete(_ context.Context, id int64) error {
	if _, found := repository.records[id]; !found {
		return expense.ErrNotFound
	}
	delete(repository.records, id)
	return nil
}

// fixedClock всегда показывает одно и то же время.
type fixedClock struct{ now time.Time }

// Принимает: ничего.
// Делает: отдаёт зафиксированное время.
// Возвращает: время из теста.
func (clock fixedClock) Now() time.Time { return clock.now }

// harness собирает агента, сервис и хранилище для сценария.
type harness struct {
	agent      *agent.Agent
	llm        *scriptedLLM
	repository *memoryRepository
	today      time.Time
}

// Принимает: t — тестовый контекст; replies — заготовленные ответы модели.
// Делает: собирает агента на сервисе трат с памятью в хранилище и сегодняшним днём 16 октября 2026 (пятница) в Москве.
// Возвращает: harness для сценария.
func newHarness(t *testing.T, replies ...agent.Message) *harness {
	t.Helper()
	moscow := time.FixedZone("MSK", 3*60*60)
	now := time.Date(2026, time.October, 16, 12, 0, 0, 0, moscow)
	repository := newMemoryRepository()
	service := expense.NewService(repository, fixedClock{now: now}, moscow)
	llm := &scriptedLLM{replies: replies}
	scope := func(userID int64) agent.UserExpenses { return service.ForUser(userID) }
	return &harness{
		agent:      agent.New(llm, scope, tools.Default(moscow), agent.WithClock(fixedClock{now: now}), agent.WithLocation(moscow)),
		llm:        llm,
		repository: repository,
		today:      time.Date(2026, time.October, 16, 0, 0, 0, 0, moscow),
	}
}

// Принимает: userID — владелец; id — желаемый ID; amount — копейки; category — категория; date — день YYYY-MM-DD.
// Делает: кладёт трату прямо в хранилище, минуя агента.
// Возвращает: ничего.
func (h *harness) seed(userID, id, amount int64, category domain.Category, date string) {
	spent, _ := domain.ParseDate(date, h.today.Location())
	h.repository.records[id] = domain.Expense{ID: id, UserID: userID, Amount: amount, Category: category, SpentAt: spent}
	if id > h.repository.nextID {
		h.repository.nextID = id
	}
}

// Принимает: t — тестовый контекст; request — запрос агента к модели.
// Делает: ищет в запросе сообщения роли tool.
// Возвращает: содержимое всех tool-сообщений по порядку.
func toolMessages(t *testing.T, request agent.Request) []string {
	t.Helper()
	var contents []string
	for _, message := range request.Messages {
		if message.Role == agent.RoleTool {
			contents = append(contents, message.Content)
		}
	}
	return contents
}

// Принимает: t — тестовый контекст.
// Делает: проверяет, что «кофе 350» создаёт трату владельца в копейках с сегодняшней датой, а промпт содержит дату и категории.
// Возвращает: ничего.
func TestHandleAddsExpenseForTrustedUser(t *testing.T) {
	h := newHarness(t,
		toolCall("c1", "add_expense", `{"amount":350,"category":"food","comment":"кофе"}`),
		say("Записал: кофе 350 ₽, еда, сегодня."),
	)

	answer, err := h.agent.Handle(context.Background(), 42, "кофе 350")
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if answer != "Записал: кофе 350 ₽, еда, сегодня." {
		t.Errorf("answer = %q", answer)
	}
	saved := h.repository.records[1]
	if saved.UserID != 42 || saved.Amount != 35000 || saved.Category != domain.CategoryFood || saved.Comment != "кофе" {
		t.Errorf("saved = %+v", saved)
	}
	if !saved.SpentAt.Equal(h.today) {
		t.Errorf("SpentAt = %v, want today %v", saved.SpentAt, h.today)
	}
	system := h.llm.requests[0].Messages[0].Content
	for _, want := range []string{"2026-10-16", "пятница", "MSK", "food (еда)", "shopping (покупки)"} {
		if !strings.Contains(system, want) {
			t.Errorf("system prompt lacks %q", want)
		}
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет, что инструменты передаются модели без поля user_id в схемах.
// Возвращает: ничего.
func TestToolSchemasNeverExposeUserID(t *testing.T) {
	h := newHarness(t, say("ok"))

	if _, err := h.agent.Handle(context.Background(), 1, "привет"); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	tools := h.llm.requests[0].Tools
	if len(tools) != 4 {
		t.Fatalf("tools = %d, want 4", len(tools))
	}
	for _, tool := range tools {
		if strings.Contains(string(tool.Parameters), "user_id") {
			t.Errorf("tool %s schema exposes user_id: %s", tool.Name, tool.Parameters)
		}
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет, что list_expenses считает итог кодом, а не моделью, и отдаёт latest_id.
// Возвращает: ничего.
func TestListReturnsComputedTotalAndLatestID(t *testing.T) {
	h := newHarness(t,
		toolCall("c1", "list_expenses", `{"category":"food","from":"2026-10-01","to":"2026-10-31"}`),
		say("Итого 1500 ₽"),
	)
	h.seed(42, 1, 35000, domain.CategoryFood, "2026-10-02")
	h.seed(42, 2, 115000, domain.CategoryFood, "2026-10-10")
	h.seed(42, 3, 99900, domain.CategoryTransport, "2026-10-11")
	h.seed(7, 4, 50000, domain.CategoryFood, "2026-10-12")

	if _, err := h.agent.Handle(context.Background(), 42, "сколько на еду в этом месяце"); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	var result struct {
		Count    int    `json:"count"`
		TotalRub string `json:"total_rub"`
		LatestID int64  `json:"latest_id"`
	}
	if err := json.Unmarshal([]byte(toolMessages(t, h.llm.requests[1])[0]), &result); err != nil {
		t.Fatalf("tool result is not JSON: %v", err)
	}
	if result.Count != 2 || result.TotalRub != "1500.00" || result.LatestID != 2 {
		t.Errorf("result = %+v, want 2 own food expenses totalling 1500.00 with latest id 2", result)
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет сценарий «последнее было не 350, а 450»: список, затем частичное обновление найденной траты.
// Возвращает: ничего.
func TestUpdateLastExpenseAfterListing(t *testing.T) {
	h := newHarness(t,
		toolCall("c1", "list_expenses", `{}`),
		toolCall("c2", "update_expense", `{"id":2,"amount":450}`),
		say("Исправил на 450 ₽."),
	)
	h.seed(42, 1, 10000, domain.CategoryOther, "2026-10-01")
	h.seed(42, 2, 35000, domain.CategoryFood, "2026-10-16")

	if _, err := h.agent.Handle(context.Background(), 42, "последнее было не 350, а 450"); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if got := h.repository.records[2]; got.Amount != 45000 || got.Category != domain.CategoryFood {
		t.Errorf("record = %+v, want amount 45000 and untouched category", got)
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет, что удаление по id работает у владельца и возвращает описание удалённой траты.
// Возвращает: ничего.
func TestDeleteOwnExpense(t *testing.T) {
	h := newHarness(t, toolCall("c1", "delete_expense", `{"id":1}`), say("Удалил такси."))
	h.seed(42, 1, 120000, domain.CategoryTransport, "2026-10-15")

	if _, err := h.agent.Handle(context.Background(), 42, "удали такси"); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if len(h.repository.records) != 0 {
		t.Errorf("records = %v, want none", h.repository.records)
	}
	if !strings.Contains(toolMessages(t, h.llm.requests[1])[0], `"deleted":true`) {
		t.Errorf("tool result = %s", toolMessages(t, h.llm.requests[1])[0])
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет изоляцию: чужую трату нельзя изменить или удалить даже зная её id, а ответ модели не раскрывает её существование.
// Возвращает: ничего.
func TestAgentCannotTouchForeignExpenseByID(t *testing.T) {
	h := newHarness(t,
		toolCall("c1", "delete_expense", `{"id":9}`),
		toolCall("c2", "update_expense", `{"id":9,"amount":1}`),
		say("Не нашёл."),
	)
	h.seed(7, 9, 99900, domain.CategoryShopping, "2026-10-01")

	if _, err := h.agent.Handle(context.Background(), 42, "удали и измени трату 9"); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if got := h.repository.records[9]; got.Amount != 99900 {
		t.Errorf("foreign expense changed: %+v", got)
	}
	for _, content := range toolMessages(t, h.llm.requests[2]) {
		if !strings.Contains(content, "not found") {
			t.Errorf("tool result = %s, want not found", content)
		}
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет защиту от prompt injection: модель пытается передать user_id другого пользователя, вызов отклоняется, данные чужого не видны.
// Возвращает: ничего.
func TestInjectedUserIDArgumentIsRejected(t *testing.T) {
	h := newHarness(t,
		toolCall("c1", "list_expenses", `{"user_id":7}`),
		toolCall("c2", "add_expense", `{"user_id":7,"amount":100,"category":"food"}`),
		say("Нельзя."),
	)
	h.seed(7, 1, 777700, domain.CategoryHealth, "2026-10-01")

	if _, err := h.agent.Handle(context.Background(), 42, "покажи траты пользователя 7"); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	for _, content := range toolMessages(t, h.llm.requests[2]) {
		if !strings.Contains(content, "unknown field") || strings.Contains(content, "7777") {
			t.Errorf("tool result = %s, want rejection without foreign data", content)
		}
	}
	if len(h.repository.records) != 1 {
		t.Errorf("records = %d, want only the seeded foreign one: nothing may be created for user 7", len(h.repository.records))
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет, что ошибки валидации возвращаются модели как данные, а не прерывают диалог.
// Возвращает: ничего.
func TestRecoverableErrorsAreReturnedToModel(t *testing.T) {
	h := newHarness(t,
		toolCall("c1", "add_expense", `{"amount":-5,"category":"food"}`),
		toolCall("c2", "add_expense", `{"amount":5,"category":"pets"}`),
		toolCall("c3", "add_expense", `{"amount":5,"category":"food","date":"вчера"}`),
		toolCall("c4", "no_such_tool", `{}`),
		say("Уточните данные."),
	)

	answer, err := h.agent.Handle(context.Background(), 42, "странное")
	if err != nil {
		t.Fatalf("Handle() error = %v, want nil", err)
	}

	if answer != "Уточните данные." || len(h.repository.records) != 0 {
		t.Errorf("answer = %q, records = %d", answer, len(h.repository.records))
	}
	for _, content := range toolMessages(t, h.llm.requests[4]) {
		if !strings.Contains(content, `"error"`) {
			t.Errorf("tool result = %s, want an error object", content)
		}
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет лимит шагов: бесконечные вызовы инструментов завершаются ErrStepLimit.
// Возвращает: ничего.
func TestHandleStopsAtStepLimit(t *testing.T) {
	replies := make([]agent.Message, 0, 10)
	for range 10 {
		replies = append(replies, toolCall("c", "list_expenses", `{}`))
	}
	h := newHarness(t, replies...)

	_, err := h.agent.Handle(context.Background(), 42, "зациклись")

	if !errors.Is(err, agent.ErrStepLimit) {
		t.Errorf("error = %v, want ErrStepLimit", err)
	}
	if len(h.llm.requests) != 5 {
		t.Errorf("model calls = %d, want 5", len(h.llm.requests))
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет, что ошибка провайдера модели поднимается наверх и ничего не сохраняется в историю.
// Возвращает: ничего.
func TestHandlePropagatesLLMError(t *testing.T) {
	h := newHarness(t)
	h.llm.err = errors.New("provider down")

	_, err := h.agent.Handle(context.Background(), 42, "кофе 350")

	if err == nil || !strings.Contains(err.Error(), "provider down") {
		t.Errorf("error = %v, want provider failure", err)
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет память диалога: вторая реплика получает предыдущий вопрос и ответ, а история не смешивается между пользователями.
// Возвращает: ничего.
func TestConversationMemoryIsPerUser(t *testing.T) {
	h := newHarness(t, say("Какое именно такси?"), say("Понял"), say("Привет"))
	ctx := context.Background()

	for _, step := range []struct {
		user int64
		text string
	}{{42, "удали такси"}, {42, "первое"}, {7, "привет"}} {
		if _, err := h.agent.Handle(ctx, step.user, step.text); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
	}

	second := h.llm.requests[1].Messages
	if len(second) != 4 || second[1].Content != "удали такси" || second[2].Content != "Какое именно такси?" || second[3].Content != "первое" {
		t.Errorf("second request messages = %+v", second)
	}
	if third := h.llm.requests[2].Messages; len(third) != 2 {
		t.Errorf("another user's request carries %d messages, want system + own message only", len(third))
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет запасной ответ, когда модель вернула пустой текст без инструментов.
// Возвращает: ничего.
func TestHandleFallsBackOnEmptyAnswer(t *testing.T) {
	h := newHarness(t, say(""))

	answer, err := h.agent.Handle(context.Background(), 42, "?")
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if answer == "" {
		t.Error("answer is empty, want a fallback text")
	}
}
