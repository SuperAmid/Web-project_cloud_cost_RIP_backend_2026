package handler

import (
	"context"
	"mime/multipart"
	"time"

	"provider_routers/internal/ds"
	"provider_routers/internal/redis"
	"provider_routers/internal/repository"
)

// ProviderRouterRepository — интерфейс домена маршрутизаторов (/api/provider_routers).
// Обработчики зависят от интерфейса, реализация — *repository.Repository.
type ProviderRouterRepository interface {
	ListProviderRouters(filter repository.ProviderRouterFilter) ([]ds.ProviderRouter, error)
	GetFeedProviderRouter(id *uint, next bool) (ds.ProviderRouter, error)
	GetDraftProviderRouter(userID uint) (ds.ProviderRouter, error)
	CreateDraftProviderRouter(ctx context.Context, userID uint, name string, image, video *multipart.FileHeader) (ds.ProviderRouter, error)
	PublishProviderRouter(userID, id uint, fields repository.ProviderRouterPublish) (ds.ProviderRouter, error)
	DeleteProviderRouter(userID, id uint) (ds.ProviderRouter, error)
	SetProviderRouterLike(userID, id uint, like bool) (int64, error)
	MediaURL(key, fallback string) string
}

// TokenBlacklist — хранилище отозванных JWT (Redis). После выхода токен
// попадает сюда и до конца срока действия не принимается.
type TokenBlacklist interface {
	WriteJWTToBlacklist(ctx context.Context, token, email string, ttl time.Duration) error
	IsJWTBlacklisted(ctx context.Context, token string) (bool, error)
}

// ProviderRouterUserRepository — интерфейс домена пользователей (/api/provider_router_users).
type ProviderRouterUserRepository interface {
	CreateProviderRouterUser(user *ds.ProviderRouterUser) error
	FindProviderRouterUserByEmail(email string) (ds.ProviderRouterUser, error)
}

var (
	_ ProviderRouterRepository     = (*repository.Repository)(nil)
	_ ProviderRouterUserRepository = (*repository.Repository)(nil)
	_ TokenBlacklist               = (*redis.Client)(nil)
)
