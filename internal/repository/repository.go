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

// Repository реализует интерфейсы handler.ProviderRouterRepository и
// handler.ProviderRouterUserRepository. Файлы хранит через интерфейс MediaStorage.
type Repository struct {
	db    *gorm.DB
	media MediaStorage
}

func New(databaseURL string, media MediaStorage) (*Repository, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, err
	}
	return NewWithDB(db, media), nil
}

// NewWithDB — для тестов и случаев, когда соединение уже открыто.
func NewWithDB(db *gorm.DB, media MediaStorage) *Repository {
	return &Repository{db: db, media: media}
}

// MigrateAndSeed выполняет SQL-миграции из каталога dir и наполняет пустую БД.
func (r *Repository) MigrateAndSeed(dir string) error {
	if err := ApplyMigrations(r.db, dir); err != nil {
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
	// Черновик 104 принадлежит оператору, а не текущему пользователю: так при показе
	// запрос «добавление» у студента сразу проходит, без предварительного удаления.
	now := time.Now()
	text := func(s string) *string { return &s }
	num := func(n int) *int { return &n }
	items := []ds.ProviderRouter{
		{Name: "Core Backbone One", Description: text("Центральный маршрутизатор ядра провайдера для магистрального узла и распределения трафика между сегментами сети."), Status: ds.ProviderRouterStatusPublished, ImageKey: "provider_routers/core.svg", VideoKey: "videos/core-loop.mp4", BandwidthMbps: num(100000), CostRub: num(980000), CreatedByID: operator.ID, PublishedAt: &now},
		{Name: "North Ring Hub", Description: text("Промежуточный маршрутизатор кольцевой сети, который агрегирует районные узлы и передаёт трафик на магистраль."), Status: ds.ProviderRouterStatusPublished, ImageKey: "provider_routers/ring.svg", VideoKey: "videos/ring-loop.mp4", BandwidthMbps: num(10000), CostRub: num(310000), CreatedByID: operator.ID, PublishedAt: &now},
		{Name: "Harbor Residence Gateway", Description: text("Конечный маршрутизатор жилого комплекса для распределения доступа к сети между квартирами."), Status: ds.ProviderRouterStatusPublished, ImageKey: "provider_routers/residential.svg", VideoKey: "videos/residential-loop.mp4", BandwidthMbps: num(1000), CostRub: num(72000), CreatedByID: operator.ID, PublishedAt: &now},
		{Name: "Riverside Residence Gateway", Status: ds.ProviderRouterStatusDraft, ImageKey: DefaultImageKey, VideoKey: DefaultVideoKey, CreatedByID: operator.ID},
		{Name: "Legacy South Edge", Description: text("Выведенный из эксплуатации пограничный маршрутизатор."), Status: ds.ProviderRouterStatusDeleted, ImageKey: "provider_routers/ring.svg", VideoKey: "videos/ring-loop.mp4", BandwidthMbps: num(1000), CostRub: num(50000), CreatedByID: operator.ID, PublishedAt: &now},
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
