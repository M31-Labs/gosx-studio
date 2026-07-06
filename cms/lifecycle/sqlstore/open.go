package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

func Open(path string, options ...Option) (*Store, func() error, error) {
	return OpenContext(context.Background(), path, options...)
}

func OpenContext(ctx context.Context, path string, options ...Option) (*Store, func() error, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil, fmt.Errorf("sqlstore requires a sqlite path")
	}
	if path != ":memory:" {
		dir := filepath.Dir(path)
		if dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, nil, err
			}
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, nil, err
	}
	db.SetMaxOpenConns(1)
	store, err := New(db, options...)
	if err != nil {
		_ = db.Close()
		return nil, nil, err
	}
	if err := store.Migrate(ctx); err != nil {
		_ = db.Close()
		return nil, nil, err
	}
	return store, db.Close, nil
}
