package main

import "time"

// Статусы маршрутизатора: draft → published, published → deleted.
const (
	statusDraft     = "draft"
	statusPublished = "published"
	statusDeleted   = "deleted"
)

// ProviderRouterUser — таблица provider_router_users. Владелец черновиков, ставит лайки.
type ProviderRouterUser struct {
	ID          uint   `gorm:"primaryKey"`
	Email       string `gorm:"size:120;unique;not null"`
	DisplayName string `gorm:"size:80;not null"`
}

func (ProviderRouterUser) TableName() string { return "provider_router_users" }

// ProviderRouter — таблица provider_routers (услуга).
// Поля по теме: BandwidthMbps (пропускная способность) и CostRub (стоимость).
// Description, BandwidthMbps, CostRub и PublishedAt необязательные (NULL):
// по «Далее» заполняются только название, ключи медиа, статус, создатель и дата
// создания, а эти четыре поля — только на шаге «Опубликовать».
type ProviderRouter struct {
	ID            uint                 `gorm:"primaryKey"`
	Name          string               `gorm:"size:120;not null"`
	Description   *string              `gorm:"size:500"`
	Status        string               `gorm:"size:12;index;not null"`
	ImageKey      string               `gorm:"size:160;not null;default:''"`
	VideoKey      string               `gorm:"size:160;not null;default:''"`
	BandwidthMbps *int                 ``
	CostRub       *int                 ``
	CreatedAt     time.Time            `gorm:"not null"`
	PublishedAt   *time.Time           // дата формирования (публикации)
	CreatedByID   uint                 `gorm:"not null;index"`
	CreatedBy     ProviderRouterUser   `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`
	Likes         []ProviderRouterLike `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`
}

func (ProviderRouter) TableName() string { return "provider_routers" }

// ProviderRouterLike — таблица м-м provider_router_likes: собственный первичный ключ
// id и уникальная пара (provider_router_id, user_id) — один лайк от пользователя.
type ProviderRouterLike struct {
	ID               uint               `gorm:"primaryKey"`
	ProviderRouterID uint               `gorm:"not null;uniqueIndex:provider_router_like_pair"`
	UserID           uint               `gorm:"not null;uniqueIndex:provider_router_like_pair"`
	ProviderRouter   ProviderRouter     `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`
	User             ProviderRouterUser `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`
}

func (ProviderRouterLike) TableName() string { return "provider_router_likes" }
