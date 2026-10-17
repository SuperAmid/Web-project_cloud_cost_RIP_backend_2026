// Package repository — доступ к PostgreSQL (через GORM) и MinIO.
package repository

import (
	"errors"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"provider_routers/internal/ds"
)

// Ошибки бизнес-логики; handler переводит их в HTTP-коды.
var (
	ErrNotFound  = errors.New("not found") // 404
	ErrForbidden = errors.New("forbidden") // 403
	ErrConflict  = errors.New("conflict")  // 409
)

type Repository struct {
	db    *gorm.DB
	media *Media
}

func New(databaseURL string, media *Media) (*Repository, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, err
	}
	return &Repository{db: db, media: media}, nil
}

// MigrateAndSeed создаёт/обновляет таблицы и заполняет пустую БД начальными данными.
func (r *Repository) MigrateAndSeed() error {
	if err := r.db.AutoMigrate(&ds.ProviderRouterUser{}, &ds.ProviderRouter{}, &ds.ProviderRouterLike{}); err != nil {
		return err
	}
	if err := r.db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS one_draft_provider_router_per_creator ON provider_routers (created_by_id) WHERE status = 'draft'").Error; err != nil {
		return err
	}
	var count int64
	if err := r.db.Model(&ds.ProviderRouter{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	student := ds.ProviderRouterUser{Email: fixedProviderRouterUserEmail, DisplayName: "Студент РИП"}
	operator := ds.ProviderRouterUser{Email: "operator@provider-routers.local", DisplayName: "Оператор сети"}
	viewer := ds.ProviderRouterUser{Email: "viewer@provider-routers.local", DisplayName: "Наблюдатель"}
	if err := r.db.Create(&[]*ds.ProviderRouterUser{&student, &operator, &viewer}).Error; err != nil {
		return err
	}
	now := time.Now()
	items := []ds.ProviderRouter{
		{Name: "Core Backbone One", Description: "Центральный маршрутизатор ядра провайдера для магистрального узла и распределения трафика между сегментами сети.", Status: ds.ProviderRouterStatusPublished, ImageKey: "provider_routers/core.svg", VideoKey: "videos/core-loop.mp4", RouterType: "central", BandwidthMbps: 100000, CostRub: 980000, CreatedByID: operator.ID, PublishedAt: &now},
		{Name: "North Ring Hub", Description: "Промежуточный маршрутизатор кольцевой сети, который агрегирует районные узлы и передаёт трафик на магистраль.", Status: ds.ProviderRouterStatusPublished, ImageKey: "provider_routers/ring.svg", VideoKey: "videos/ring-loop.mp4", RouterType: "intermediate", BandwidthMbps: 10000, CostRub: 310000, CreatedByID: operator.ID, PublishedAt: &now},
		{Name: "Harbor Residence Gateway", Description: "Конечный маршрутизатор жилого комплекса для распределения доступа к сети между квартирами.", Status: ds.ProviderRouterStatusPublished, ImageKey: "provider_routers/residential.svg", VideoKey: "videos/residential-loop.mp4", RouterType: "residential", BandwidthMbps: 1000, CostRub: 72000, CreatedByID: operator.ID, PublishedAt: &now},
		{Name: "Riverside Residence Gateway", Description: "Черновик карточки маршрутизатора для следующего жилого дома.", Status: ds.ProviderRouterStatusDraft, ImageKey: DefaultImageKey, VideoKey: DefaultVideoKey, RouterType: "residential", BandwidthMbps: 1000, CostRub: 65000, CreatedByID: student.ID},
		{Name: "Legacy South Edge", Description: "Выведенный из эксплуатации пограничный маршрутизатор.", Status: ds.ProviderRouterStatusDeleted, ImageKey: "provider_routers/ring.svg", VideoKey: "videos/ring-loop.mp4", RouterType: "intermediate", BandwidthMbps: 1000, CostRub: 50000, CreatedByID: operator.ID},
	}
	if err := r.db.Create(&items).Error; err != nil {
		return err
	}
	return r.db.Create(&[]ds.ProviderRouterLike{
		{ProviderRouterID: items[0].ID, UserID: student.ID},
		{ProviderRouterID: items[0].ID, UserID: viewer.ID},
		{ProviderRouterID: items[1].ID, UserID: viewer.ID},
		{ProviderRouterID: items[2].ID, UserID: student.ID},
		{ProviderRouterID: items[2].ID, UserID: operator.ID},
	}).Error
}
