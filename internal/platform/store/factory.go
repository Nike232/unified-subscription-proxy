package store

import (
	"context"
	"fmt"
	"strings"
)

func NewConfiguredStore(ctx context.Context, backend, filePath, dsn string) (Store, error) {
	switch strings.ToLower(strings.TrimSpace(backend)) {
	case "", "file":
		return NewFileStore(filePath), nil
	case "postgres":
		return NewPostgresStore(ctx, dsn)
	default:
		return nil, fmt.Errorf("unsupported store backend: %s", backend)
	}
}
