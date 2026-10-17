// Package ds описывает модели GORM — по одной структуре на таблицу БД.
package ds

import "time"

// Статусы маршрутизатора. Переходы: draft -> published, draft|published -> deleted.
// Вернуть запись в draft нельзя.
const (
	ProviderRouterStatusDraft     = "draft"
	ProviderRouterStatusPublished = "published"
	ProviderRouterStatusDeleted   = "deleted"
)

// Допустимые типы маршрутизатора сети провайдера.
var ProviderRouterTypes = map[string]bool{
	"central":      true,
	"intermediate": true,
	"residential":  true,
}

// ProviderRouterUser — таблица provider_router_users.
type ProviderRouterUser struct {
	ID           uint      `gorm:"primaryKey"`
	Email        string    `gorm:"size:120;unique;not null"`
	DisplayName  string    `gorm:"size:80;not null"`
	PasswordHash string    `gorm:"size:72;not null;default:''"`
	CreatedAt    time.Time `gorm:"not null"`
}

func (ProviderRouterUser) TableName() string { return "provider_router_users" }

// ProviderRouter — таблица provider_routers (услуга предметной области).
// Поля по теме: BandwidthMbps (пропускная способность) и CostRub (стоимость).
type ProviderRouter struct {
	ID            uint                 `gorm:"primaryKey"`
	Name          string               `gorm:"size:120;not null"`
	Description   string               `gorm:"size:500;not null;default:''"`
	Status        string               `gorm:"size:12;index;not null"`
	ImageKey      string               `gorm:"size:160;not null;default:''"`
	VideoKey      string               `gorm:"size:160;not null;default:''"`
	RouterType    string               `gorm:"size:24;not null;default:'residential'"`
	BandwidthMbps int                  `gorm:"not null;default:0"`
	CostRub       int                  `gorm:"not null;default:0"`
	CreatedAt     time.Time            `gorm:"not null"`
	PublishedAt   *time.Time           // дата формирования (публикации)
	CreatedByID   uint                 `gorm:"not null;index"`
	CreatedBy     ProviderRouterUser   `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`
	Likes         []ProviderRouterLike `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`
}

func (ProviderRouter) TableName() string { return "provider_routers" }

// ProviderRouterLike — таблица м-м provider_router_likes (пользователь лайкнул маршрутизатор).
type ProviderRouterLike struct {
	ProviderRouterID uint               `gorm:"primaryKey"`
	UserID           uint               `gorm:"primaryKey"`
	CreatedAt        time.Time          `gorm:"not null"`
	ProviderRouter   ProviderRouter     `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`
	User             ProviderRouterUser `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`
}

func (ProviderRouterLike) TableName() string { return "provider_router_likes" }
