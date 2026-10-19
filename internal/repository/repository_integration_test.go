package repository

// Интеграционные тесты на реальной PostgreSQL. Запуск:
//   docker compose up -d postgres
//   TEST_DATABASE_URL="postgres://provider_routers_user:provider_routers_password@localhost:5433/provider_routers_test?sslmode=disable" go test ./internal/repository/
// Без TEST_DATABASE_URL тесты пропускаются.

import (
	"context"
	"errors"
	"mime/multipart"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"provider_routers/internal/ds"
)

// fakeMedia — MediaStorage без MinIO: проверяет файл и возвращает ключ.
type fakeMedia struct{ removed []string }

func (f *fakeMedia) Upload(_ context.Context, kind MediaKind, header *multipart.FileHeader) (string, error) {
	if _, _, err := kind.ValidateFile(header); err != nil {
		return "", err
	}
	return kind.Prefix + "/" + generateObjectName(".bin"), nil
}
func (f *fakeMedia) Remove(_ context.Context, key string) { f.removed = append(f.removed, key) }
func (f *fakeMedia) URL(key, fallback string) string {
	if key == "" {
		key = fallback
	}
	return "http://media/" + key
}

func testRepository(t *testing.T) *Repository {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL не задан — интеграционный тест пропущен")
	}
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	// Чистая схема на каждый запуск.
	if err := db.Exec(`DROP TABLE IF EXISTS provider_router_likes, provider_routers, provider_router_users, schema_migrations CASCADE`).Error; err != nil {
		t.Fatal(err)
	}
	currentProviderRouterUserMu.Lock()
	currentProviderRouterUser = nil
	currentProviderRouterUserMu.Unlock()
	repo := NewWithDB(db, &fakeMedia{})
	if err := repo.MigrateAndSeed("../../migrations"); err != nil {
		t.Fatal(err)
	}
	return repo
}

func files() (*multipart.FileHeader, *multipart.FileHeader) {
	return &multipart.FileHeader{Filename: "r.png", Size: 10}, &multipart.FileHeader{Filename: "r.mp4", Size: 10}
}

func TestStatusTransitions(t *testing.T) {
	repo := testRepository(t)
	ctx := context.Background()
	user, err := repo.CurrentProviderRouterUser()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetDraftProviderRouter(user.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("у студента после сида не должно быть черновика: %v", err)
	}

	image, video := files()
	draft, err := repo.CreateDraftProviderRouter(ctx, user.ID, "Test Router", image, video)
	if err != nil {
		t.Fatal(err)
	}
	if draft.Status != ds.ProviderRouterStatusDraft || draft.Description != nil || draft.PublishedAt != nil {
		t.Fatalf("черновик создан с неверными полями: %+v", draft)
	}
	if _, err := repo.CreateDraftProviderRouter(ctx, user.ID, "Second", image, video); !errors.Is(err, ErrConflict) {
		t.Fatalf("второй черновик должен давать ErrConflict, получено %v", err)
	}

	fields := ProviderRouterPublish{Description: "описание", BandwidthMbps: 2500, CostRub: 84000}
	published, err := repo.PublishProviderRouter(user.ID, draft.ID, fields)
	if err != nil {
		t.Fatal(err)
	}
	if published.Status != ds.ProviderRouterStatusPublished || published.PublishedAt == nil || *published.CostRub != 84000 {
		t.Fatalf("draft → published не выполнен: %+v", published)
	}
	if _, err := repo.PublishProviderRouter(user.ID, draft.ID, fields); !errors.Is(err, ErrConflict) {
		t.Fatalf("повторная публикация должна давать ErrConflict, получено %v", err)
	}

	if count, err := repo.SetProviderRouterLike(user.ID, draft.ID, true); err != nil || count != 1 {
		t.Fatalf("лайк: count=%d err=%v", count, err)
	}
	if count, _ := repo.SetProviderRouterLike(user.ID, draft.ID, true); count != 1 {
		t.Fatalf("повторный лайк не должен увеличивать счётчик, count=%d", count)
	}
	feed, err := repo.GetFeedProviderRouter(&draft.ID, false)
	if err != nil || len(feed.Likes) != 1 || feed.CreatedBy.ID != user.ID {
		t.Fatalf("лента: %+v, %v", feed, err)
	}

	deleted, err := repo.DeleteProviderRouter(user.ID, draft.ID)
	if err != nil || deleted.Status != ds.ProviderRouterStatusDeleted {
		t.Fatalf("published → deleted: %+v, %v", deleted, err)
	}
	if _, err := repo.GetFeedProviderRouter(&draft.ID, false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("удалённая запись не должна попадать в ленту: %v", err)
	}
	if _, err := repo.PublishProviderRouter(user.ID, draft.ID, fields); !errors.Is(err, ErrNotFound) {
		t.Fatalf("удалённую запись нельзя опубликовать (вернуть из deleted): %v", err)
	}
	if _, err := repo.DeleteProviderRouter(user.ID, draft.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("повторное удаление должно давать ErrNotFound: %v", err)
	}
}

func TestForeignRecordsAndFilter(t *testing.T) {
	repo := testRepository(t)
	user, _ := repo.CurrentProviderRouterUser()
	var foreign ds.ProviderRouter
	if err := repo.db.Where("status = ?", ds.ProviderRouterStatusPublished).First(&foreign).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.DeleteProviderRouter(user.ID, foreign.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("чужую запись удалять нельзя: %v", err)
	}
	min, max := 5000, 500000
	items, err := repo.ListProviderRouters(ProviderRouterFilter{MinBandwidthMbps: &min, MaxCostRub: &max})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "North Ring Hub" {
		t.Fatalf("фильтр в БД вернул %d записей: %+v", len(items), items)
	}
	all, _ := repo.ListProviderRouters(ProviderRouterFilter{})
	last := all[len(all)-1].ID
	next, err := repo.GetFeedProviderRouter(&last, true)
	if err != nil || next.ID != all[0].ID {
		t.Fatalf("после последней карточки next=true должен вернуть первую: %+v %v", next, err)
	}
}
