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
	"github.com/ValievM/family-expense-bot/internal/adapters/telegram"
	"github.com/ValievM/family-expense-bot/internal/config"
	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/service/agent"
	"github.com/ValievM/family-expense-bot/internal/service/agent/tools"
	"github.com/ValievM/family-expense-bot/internal/service/expense"
)

const (
	databaseWait    = 30 * time.Second
	shutdownTimeout = 10 * time.Second
)

// Принимает: ничего.
// Делает: запускает приложение и превращает ошибку запуска в код возврата процесса.
// Возвращает: ничего.
func main() {
	if err := run(); err != nil {
		slog.Error("server stopped with an error", "error", err)
		os.Exit(1)
	}
}

// Принимает: ничего.
// Делает: читает конфигурацию, готовит базу, собирает сервис, API и бота, работает до сигнала остановки и затем гасит всё по порядку.
// Возвращает: nil при штатной остановке либо первую ошибку запуска или работы.
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

	service := expense.NewService(postgres.NewRepository(pool), nil, cfg.Location)
	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewServer(service, logger).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	var workers sync.WaitGroup
	failures := make(chan error, 2)
	workers.Add(1)
	go func() {
		defer workers.Done()
		logger.Info("http api listening", "addr", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			failures <- fmt.Errorf("serve http: %w", err)
		}
	}()

	if cfg.TelegramEnabled {
		bot, err := newBot(cfg, service, logger)
		if err != nil {
			return err
		}
		workers.Add(1)
		go func() {
			defer workers.Done()
			logger.Info("telegram bot started", "llm_model", cfg.LLMModel)
			if err := bot.Run(ctx); err != nil {
				failures <- fmt.Errorf("run telegram bot: %w", err)
			}
		}()
	} else {
		logger.Warn("TELEGRAM_BOT_TOKEN is empty: the bot is disabled and only the REST API runs")
	}

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

// Принимает: ctx — контекст остановки; databaseURL — строка подключения к PostgreSQL.
// Делает: открывает пул, ждёт готовности базы до databaseWait и применяет встроенные миграции.
// Возвращает: готовый пул либо ошибку подключения или миграции.
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
// Делает: раз в секунду проверяет доступность базы, пока не истечёт databaseWait.
// Возвращает: nil, когда база ответила, либо последнюю ошибку проверки.
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
		case <-time.After(time.Second):
		}
	}
}

// Принимает: cfg — конфигурация; service — сервис трат; logger — логгер.
// Делает: собирает цепочку LLM-клиент, агент, Telegram-бот; агенту выдаётся только фабрика трат, привязанных к автору сообщения.
// Возвращает: готового бота либо ошибку, если выбран вход через ChatGPT, а входа ещё не было.
func newBot(cfg config.Config, service *expense.Service, logger *slog.Logger) (*telegram.Bot, error) {
	model, err := newLLM(cfg)
	if err != nil {
		return nil, err
	}
	scope := func(owner domain.UserID) agent.UserExpenses { return service.ForUser(owner) }
	assistant := agent.New(model, scope, tools.Default(), agent.WithLocation(cfg.Location), agent.WithLogger(logger))
	return telegram.NewBot(telegram.NewClient(cfg.TelegramToken), assistant, cfg, telegram.WithLogger(logger)), nil
}

// Принимает: cfg — конфигурация с режимом LLM_AUTH.
// Делает: выбирает способ доступа к модели: ключ API через Chat Completions или вход через ChatGPT через Responses API; при втором проверяет, что вход уже выполнен.
// Возвращает: реализацию agent.LLM либо ошибку (например, chatgpt.ErrNotLoggedIn).
func newLLM(cfg config.Config) (agent.LLM, error) {
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
