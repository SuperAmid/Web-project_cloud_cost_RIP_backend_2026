package api

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"provider_routers/internal/ds"
	"provider_routers/internal/handler"
	"provider_routers/internal/repository"
)

// fakeRepo реализует оба интерфейса handler-а в памяти, без БД и MinIO.
type fakeRepo struct {
	users map[string]ds.ProviderRouterUser
}

func (f *fakeRepo) CurrentProviderRouterUser() (ds.ProviderRouterUser, error) {
	return ds.ProviderRouterUser{ID: 1, DisplayName: "Студент"}, nil
}
func (f *fakeRepo) published() ds.ProviderRouter {
	now := time.Now()
	text, n := "описание", 1000
	return ds.ProviderRouter{ID: 101, Name: "Core", Description: &text, BandwidthMbps: &n, CostRub: &n, Status: ds.ProviderRouterStatusPublished, PublishedAt: &now, CreatedByID: 2,
		Likes: []ds.ProviderRouterLike{{ProviderRouterID: 101, UserID: 1}}, CreatedBy: ds.ProviderRouterUser{ID: 2, DisplayName: "Оператор"}}
}
func (f *fakeRepo) ListProviderRouters(repository.ProviderRouterFilter) ([]ds.ProviderRouter, error) {
	return []ds.ProviderRouter{f.published()}, nil
}
func (f *fakeRepo) GetFeedProviderRouter(id *uint, next bool) (ds.ProviderRouter, error) {
	if id != nil && *id == 999 {
		return ds.ProviderRouter{}, repository.ErrNotFound
	}
	return f.published(), nil
}
func (f *fakeRepo) GetDraftProviderRouter(uint) (ds.ProviderRouter, error) {
	return ds.ProviderRouter{ID: 106, Name: "Draft", Status: ds.ProviderRouterStatusDraft, CreatedByID: 1}, nil
}
func (f *fakeRepo) CreateDraftProviderRouter(_ context.Context, userID uint, name string, _, _ *multipart.FileHeader) (ds.ProviderRouter, error) {
	return ds.ProviderRouter{ID: 106, Name: name, Status: ds.ProviderRouterStatusDraft, CreatedByID: userID}, nil
}
func (f *fakeRepo) PublishProviderRouter(userID, id uint, _ repository.ProviderRouterPublish) (ds.ProviderRouter, error) {
	if id == 101 {
		return ds.ProviderRouter{}, repository.ErrForbidden
	}
	return ds.ProviderRouter{ID: id, Status: ds.ProviderRouterStatusPublished}, nil
}
func (f *fakeRepo) DeleteProviderRouter(userID, id uint) (ds.ProviderRouter, error) {
	return ds.ProviderRouter{ID: id, Status: ds.ProviderRouterStatusDeleted}, nil
}
func (f *fakeRepo) SetProviderRouterLike(userID, id uint, like bool) (int64, error) { return 1, nil }
func (f *fakeRepo) MediaURL(key, fallback string) string                            { return "http://media/" + key }
func (f *fakeRepo) CreateProviderRouterUser(user *ds.ProviderRouterUser) error {
	if _, ok := f.users[user.Email]; ok {
		return repository.ErrConflict
	}
	user.ID = 4
	f.users[user.Email] = *user
	return nil
}
func (f *fakeRepo) FindProviderRouterUserByEmail(email string) (ds.ProviderRouterUser, error) {
	if u, ok := f.users[email]; ok {
		return u, nil
	}
	return ds.ProviderRouterUser{}, repository.ErrNotFound
}

func multipartBody(t *testing.T) (*bytes.Buffer, string) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("name", "Test")
	for field, name := range map[string]string{"image": "r.png", "video": "r.mp4"} {
		part, err := w.CreateFormFile(field, name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = part.Write([]byte("data"))
	}
	_ = w.Close()
	return &body, w.FormDataContentType()
}

// Все 10 маршрутов зарегистрированы и отвечают ожидаемыми кодами.
func TestRoutes(t *testing.T) {
	repo := &fakeRepo{users: map[string]ds.ProviderRouterUser{}}
	router := NewRouter(handler.New(repo, repo))
	upload, contentType := multipartBody(t)
	cases := []struct {
		method, url, body, contentType string
		raw                            *bytes.Buffer
		want                           int
	}{
		{"GET", "/api/provider_routers?min_bandwidth_mbps=1000&max_cost_rub=500000", "", "", nil, 200},
		{"GET", "/api/provider_routers?min_bandwidth_mbps=-1", "", "", nil, 400},
		{"GET", "/api/provider_routers/feed", "", "", nil, 200},
		{"GET", "/api/provider_routers/feed?id=101&next=true", "", "", nil, 200},
		{"GET", "/api/provider_routers/feed?id=999", "", "", nil, 404},
		{"GET", "/api/provider_routers/feed?next=true", "", "", nil, 400},
		{"GET", "/api/provider_routers/draft", "", "", nil, 200},
		{"POST", "/api/provider_routers", "", contentType, upload, 201},
		{"POST", "/api/provider_routers", `{"name":"x"}`, "application/json", nil, 400},
		{"PUT", "/api/provider_routers/106/publish", `{"description":"d","bandwidth_mbps":1,"cost_rub":1}`, "application/json", nil, 200},
		{"PUT", "/api/provider_routers/101/publish", `{"description":"d","bandwidth_mbps":1,"cost_rub":1}`, "application/json", nil, 403},
		{"PUT", "/api/provider_routers/106/publish", `{"description":"d","bandwidth_mbps":1,"cost_rub":1,"status":"draft"}`, "application/json", nil, 400},
		{"DELETE", "/api/provider_routers/106", "", "", nil, 200},
		{"POST", "/api/provider_routers/101/like", `{"like":1}`, "application/json", nil, 200},
		{"POST", "/api/provider_routers/101/like", `{"like":2}`, "application/json", nil, 400},
		{"POST", "/api/provider_router_users/register", `{"email":"a@b.ru","display_name":"A","password":"secret1"}`, "application/json", nil, 201},
		{"POST", "/api/provider_router_users/register", `{"email":"a@b.ru","display_name":"A","password":"secret1"}`, "application/json", nil, 409},
		{"POST", "/api/provider_router_users/login", `{"email":"a@b.ru","password":"secret1"}`, "application/json", nil, 200},
		{"POST", "/api/provider_router_users/login", `{"email":"a@b.ru","password":"wrong"}`, "application/json", nil, 401},
		{"POST", "/api/provider_router_users/logout", "", "", nil, 200},
		{"GET", "/api/unknown", "", "", nil, 404},
		{"PATCH", "/api/provider_routers/106", "", "", nil, 404}, // неизвестный метод → JSON 404 из catch-all
	}
	for _, c := range cases {
		var body *bytes.Buffer
		if c.raw != nil {
			body = c.raw
		} else {
			body = bytes.NewBufferString(c.body)
		}
		request := httptest.NewRequest(c.method, c.url, body)
		if c.contentType != "" {
			request.Header.Set("Content-Type", c.contentType)
		}
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != c.want {
			t.Errorf("%s %s: код %d, ожидался %d (%s)", c.method, c.url, recorder.Code, c.want, recorder.Body.String())
		}
		if ct := recorder.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Errorf("%s %s: Content-Type %q", c.method, c.url, ct)
		}
	}
}

var _ http.Handler = NewRouter(handler.New(&fakeRepo{}, &fakeRepo{}))
