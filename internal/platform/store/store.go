package store

import "unifiedsubscriptionproxy/internal/platform/domain"

type Store interface {
	Load() (domain.PlatformData, error)
	Save(data domain.PlatformData) error
	Mutate(fn func(*domain.PlatformData) error) (domain.PlatformData, error)
}
