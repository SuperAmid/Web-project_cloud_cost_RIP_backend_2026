// Package serializer — структуры запросов (десериализация JSON -> Go) и ответов
// (сериализация Go -> JSON). Системных полей (id, status, created_by_id,
// created_at, published_at) в структурах запросов нет: клиент не может их менять.
package serializer

import "time"

// ---------- запросы ----------

// ProviderRouterPublishRequest — тело PUT /api/provider_routers/{id}/publish.
// Указатели нужны, чтобы отличить «поле не передано» от нуля.
type ProviderRouterPublishRequest struct {
	Description   *string `json:"description"`
	BandwidthMbps *int    `json:"bandwidth_mbps"`
	CostRub       *int    `json:"cost_rub"`
}

// ProviderRouterLikeRequest — тело POST /api/provider_routers/{id}/like: 1 ставит лайк, 0 снимает.
type ProviderRouterLikeRequest struct {
	Like *int `json:"like"`
}

// ProviderRouterUserRegisterRequest — тело POST /api/provider_router_users/register.
type ProviderRouterUserRegisterRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

// ProviderRouterUserLoginRequest — тело POST /api/provider_router_users/login (заглушка до ЛР4).
type ProviderRouterUserLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ---------- ответы ----------

// ProviderRouterListItem — элемент плитки. IsOwn = 1, если создатель — текущий пользователь.
type ProviderRouterListItem struct {
	ID            uint       `json:"id"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	BandwidthMbps int        `json:"bandwidth_mbps"`
	CostRub       int        `json:"cost_rub"`
	ImageURL      string     `json:"image_url"`
	VideoURL      string     `json:"video_url"`
	LikesCount    int64      `json:"likes_count"`
	IsOwn         int        `json:"is_own"`
	PublishedAt   *time.Time `json:"published_at"`
}

// ProviderRouterFeedItem — одна карточка ленты. IsLiked = 1, если текущий пользователь лайкнул.
type ProviderRouterFeedItem struct {
	ID            uint       `json:"id"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	BandwidthMbps int        `json:"bandwidth_mbps"`
	CostRub       int        `json:"cost_rub"`
	ImageURL      string     `json:"image_url"`
	VideoURL      string     `json:"video_url"`
	LikesCount    int64      `json:"likes_count"`
	IsLiked       int        `json:"is_liked"`
	CreatorName   string     `json:"creator_name"`
	PublishedAt   *time.Time `json:"published_at"`
}

// ProviderRouterDraft — черновик или результат создания/публикации/удаления.
// У черновика description, bandwidth_mbps, cost_rub и published_at — null.
type ProviderRouterDraft struct {
	ID            uint       `json:"id"`
	Name          string     `json:"name"`
	Description   *string    `json:"description"`
	Status        string     `json:"status"`
	BandwidthMbps *int       `json:"bandwidth_mbps"`
	CostRub       *int       `json:"cost_rub"`
	ImageKey      string     `json:"image_key"`
	VideoKey      string     `json:"video_key"`
	ImageURL      string     `json:"image_url"`
	VideoURL      string     `json:"video_url"`
	CreatedAt     time.Time  `json:"created_at"`
	PublishedAt   *time.Time `json:"published_at"`
}

// ProviderRouterLikeResponse — состояние лайка после POST like.
type ProviderRouterLikeResponse struct {
	ProviderRouterID uint  `json:"provider_router_id"`
	IsLiked          int   `json:"is_liked"`
	LikesCount       int64 `json:"likes_count"`
}

// ProviderRouterUserResponse — пользователь без хеша пароля.
type ProviderRouterUserResponse struct {
	ID          uint   `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

// ErrorResponse — единый формат ошибки.
type ErrorResponse struct {
	Error string `json:"error"`
}
