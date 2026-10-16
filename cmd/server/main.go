// Command server запускает REST API панели и Telegram-бота в одном процессе; здесь собираются все зависимости.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	// Вшитая база часовых поясов нужна, чтобы TZ работал и в минимальных образах без tzdata.
	_ "time/tzdata"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/ValievM/family-expense-bot/internal/adapters/chatgpt"
	"github.com/ValievM/family-expense-bot/internal/adapters/httpapi"
	"github.com/ValievM/family-expense-bot/internal/adapters/llm"
	"github.com/ValievM/family-expense-bot/internal/adapters/postgres"
	"github.com/ValievM/family-expense-bot/internal/adapters/systemclock"
	"github.com/ValievM/family-expense-bot/internal/adapters/telegram"
	"github.com/ValievM/family-expense-bot/internal/config"
	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/ports"
	"github.com/ValievM/family-expense-bot/internal/service/agent"
	"github.com/ValievM/family-expense-bot/internal/service/agent/tools"
	"github.com/ValievM/family-expense-bot/internal/service/expense"
)

const (
	databaseWait     = 30 * time.Second
	databasePollStep = time.Second
	shutdownTimeout  = 10 * time.Second

	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped with an error", "error", err)
		os.Exit(1)
	}
}

// Принимает: ничего.
// Делает: читает конфиг, готовит базу, собирает сервис, запускает API и бота, ждёт сигнала и гасит всё по порядку.
// Возвращает: nil при штатной остановке или первую ошибку запуска либо работы.
func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := connectDatabase(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	clock := systemclock.Clock{}
	service := expense.NewService(postgres.NewRepository(pool), clock, cfg.Location)
	httpServer := newHTTPServer(cfg.HTTPAddr, httpapi.NewServer(service, pool, logger).Handler())

	var workers sync.WaitGroup
	failures := make(chan error, 2)
	goWorker(&workers, failures, func() error {
		logger.Info("http api listening", "addr", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve http: %w", err)
		}
		return nil
	})

	if cfg.TelegramEnabled {
		bot, err := newBot(cfg, service, clock, logger)
		if err != nil {
			return err
		}
		goWorker(&workers, failures, func() error {
			logger.Info("telegram bot started", "llm_model", cfg.LLMModel)
			return bot.Run(ctx)
		})
	} else {
		logger.Warn("TELEGRAM_BOT_TOKEN is empty: the bot is disabled and only the REST API runs")
	}

	return waitAndShutdown(ctx, stop, failures, &workers, httpServer, logger)
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}

// Принимает: workers — счётчик; failures — канал ошибок; job — долгая задача.
// Делает: запускает задачу в горутине, учитывая её в счётчике, и передаёт ненулевую ошибку в канал.
// Возвращает: ничего.
func goWorker(workers *sync.WaitGroup, failures chan<- error, job func() error) {
	workers.Add(1)
	go func() {
		defer workers.Done()
		if err := job(); err != nil {
			failures <- err
		}
	}()
}

// Принимает: ctx и stop — контекст сигнала и его отмена; failures — ошибки задач; workers — ожидание; httpServer, logger.
// Делает: ждёт сигнала или первой ошибки задачи, затем останавливает HTTP-сервер и дожидается задач.
// Возвращает: первую ошибку работы вместе с ошибкой остановки либо nil.
func waitAndShutdown(ctx context.Context, stop context.CancelFunc, failures <-chan error,
	workers *sync.WaitGroup, httpServer *http.Server, logger *slog.Logger) error {
	var runErr error
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case runErr = <-failures:
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		runErr = errors.Join(runErr, fmt.Errorf("shutdown http server: %w", err))
	}
	workers.Wait()
	return runErr
}

// Принимает: ctx — контекст остановки; databaseURL — строка подключения.
// Делает: открывает пул, ждёт базу до databaseWait, применяет встроенные миграции.
// Возвращает: пул или ошибку подключения либо миграции.
func connectDatabase(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	if err := waitForDatabase(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	database := stdlib.OpenDBFromPool(pool)
	defer func() { _ = database.Close() }()
	if err := postgres.Migrate(ctx, database); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// Принимает: ctx — контекст остановки; pool — пул соединений.
// Делает: раз в секунду пингует базу, пока не истечёт databaseWait.
// Возвращает: nil, когда база ответила, или последнюю ошибку проверки.
func waitForDatabase(ctx context.Context, pool *pgxpool.Pool) error {
	deadline := time.Now().Add(databaseWait)
	for {
		err := pool.Ping(ctx)
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			return fmt.Errorf("database is not reachable: %w", err)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("database is not reachable: %w", ctx.Err())
		case <-time.After(databasePollStep):
		}
	}
}

// Принимает: cfg; service — сервис трат; clock; logger.
// Делает: собирает LLM-клиент, агент и бота; агенту выдаётся только фабрика трат, привязанных к автору сообщения.
// Возвращает: бота или ошибку, если выбран вход через ChatGPT, а входа ещё не было.
func newBot(cfg config.Config, service *expense.Service, clock ports.Clock, logger *slog.Logger) (*telegram.Bot, error) {
	model, err := newLLM(cfg)
	if err != nil {
		return nil, err
	}
	scope := func(owner domain.UserID) ports.UserExpenses { return service.ForUser(owner) }
	assistant := agent.New(model, scope, tools.Default(), clock, agent.WithLocation(cfg.Location), agent.WithLogger(logger))
	return telegram.NewBot(telegram.NewClient(cfg.TelegramToken), assistant, cfg, telegram.WithLogger(logger)), nil
}

// Принимает: cfg — конфигурация с режимом LLM_AUTH.
// Делает: выбирает ключ API (Chat Completions) или вход через ChatGPT (Responses API); во втором проверяет наличие входа.
// Возвращает: ports.LLM или ошибку, например chatgpt.ErrNotLoggedIn.
func newLLM(cfg config.Config) (ports.LLM, error) {
	if cfg.LLMAuth != config.AuthOAuth {
		return llm.NewClient(cfg.LLMBaseURL, cfg.LLMModel, llm.NewAPIKeyAuthorizer(cfg.LLMAPIKey)), nil
	}

	store := chatgpt.NewFileStore(cfg.LLMAuthFile)
	if _, err := store.Load(); err != nil {
		return nil, err
	}
	source := chatgpt.NewTokenSource(store, chatgpt.NewOAuth())
	return llm.NewResponsesClient(cfg.LLMBaseURL, cfg.LLMModel, source), nil
}
