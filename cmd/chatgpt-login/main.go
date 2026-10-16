// Command chatgpt-login выполняет вход через аккаунт ChatGPT и сохраняет токены для режима LLM_AUTH=oauth.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/ValievM/family-expense-bot/internal/adapters/chatgpt"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "login failed:", err)
		os.Exit(1)
	}
}

// Принимает: ничего.
// Делает: читает флаги, подхватывает прежний вход (сохраняя client и host id), открывает браузер, пишет токены в файл.
// Возвращает: nil при успешном входе или ошибку.
func run() error {
	file := flag.String("file", "data/chatgpt-auth.json", "where to store the tokens")
	port := flag.Int("port", 1455, "local callback port on 127.0.0.1")
	name := flag.String("name", "FamilyExpenseBot", "application name shown on the consent screen")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	store := chatgpt.NewFileStore(*file)
	options := chatgpt.LoginOptions{AgentName: *name, Port: *port, OpenBrowser: openBrowser, Timeout: 5 * time.Minute}
	if existing, err := store.Load(); err == nil {
		options.Existing = &existing
	} else if !errors.Is(err, chatgpt.ErrNotLoggedIn) {
		return err
	}

	credentials, err := chatgpt.Login(ctx, chatgpt.NewOAuth(), options)
	if err != nil {
		return err
	}
	if err := store.Save(credentials); err != nil {
		return err
	}
	fmt.Printf("Signed in. Tokens saved to %s (access token valid until %s).\n", *file, credentials.ExpiresAt.Format(time.RFC3339))
	return nil
}

// Принимает: authURL — адрес страницы входа.
// Делает: печатает адрес и пробует открыть его в браузере; при неудаче пользователь откроет ссылку сам.
// Возвращает: всегда nil: неудача открытия входу не мешает.
func openBrowser(authURL string) error {
	fmt.Println("Open this link in a browser and allow plan usage:")
	fmt.Println(authURL)

	opener := "xdg-open"
	if runtime.GOOS == "darwin" {
		opener = "open"
	}
	// Ошибку запуска намеренно игнорируем: ссылка уже напечатана выше.
	_ = exec.Command(opener, authURL).Start()
	return nil
}
