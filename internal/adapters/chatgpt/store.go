package chatgpt

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// FileStore хранит учётные данные в одном JSON-файле с правами только для владельца.
type FileStore struct {
	path string
}

// Принимает: path — путь к файлу учётных данных.
// Делает: создаёт хранилище; сам файл появится при первом сохранении.
// Возвращает: FileStore.
func NewFileStore(path string) FileStore {
	return FileStore{path: path}
}

// Принимает: ничего.
// Делает: читает и разбирает файл.
// Возвращает: учётные данные, ErrNotLoggedIn если файла нет, либо ошибку чтения.
func (store FileStore) Load() (Credentials, error) {
	raw, err := os.ReadFile(store.path)
	if errors.Is(err, fs.ErrNotExist) {
		return Credentials{}, ErrNotLoggedIn
	}
	if err != nil {
		return Credentials{}, fmt.Errorf("read ChatGPT credentials: %w", err)
	}
	var credentials Credentials
	if err := json.Unmarshal(raw, &credentials); err != nil {
		return Credentials{}, fmt.Errorf("parse ChatGPT credentials %s: %w", store.path, err)
	}
	return credentials, nil
}

// Принимает: credentials — данные для сохранения.
// Делает: пишет файл атомарно (временный файл и переименование) с правами 0600, чтобы обрыв не оставил полузаписанные токены.
// Возвращает: nil либо ошибку записи.
func (store FileStore) Save(credentials Credentials) error {
	directory := filepath.Dir(store.path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create credentials directory: %w", err)
	}
	encoded, err := json.MarshalIndent(credentials, "", "  ")
	if err != nil {
		return fmt.Errorf("encode ChatGPT credentials: %w", err)
	}

	temp, err := os.CreateTemp(directory, ".auth-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary credentials file: %w", err)
	}
	defer func() { _ = os.Remove(temp.Name()) }()
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("restrict credentials file: %w", err)
	}
	if _, err := temp.Write(encoded); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write credentials: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close credentials file: %w", err)
	}
	if err := os.Rename(temp.Name(), store.path); err != nil {
		return fmt.Errorf("replace credentials file: %w", err)
	}
	return nil
}
