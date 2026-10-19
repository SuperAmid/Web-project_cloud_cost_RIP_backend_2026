package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"provider_routers/internal/auth"
	"provider_routers/internal/ds"
	"provider_routers/internal/handler"
	"provider_routers/internal/repository"
)

// fakeRepo реализует оба интерфейса handler-а в памяти, без БД и MinIO.
type fakeRepo struct {
	users map[string]ds.ProviderRouterUser
}

// fakeBlacklist — blacklist токенов в памяти вместо Redis.
type fakeBlacklist struct{ tokens map[string]string }

func (b *fakeBlacklist) WriteJWTToBlacklist(_ context.Context, token, email string, _ time.Duration) error {
	b.tokens[token] = email
	return nil
}
func (b *fakeBlacklist) IsJWTBlacklisted(_ context.Context, token string) (bool, error) {
	_, ok := b.tokens[token]
	return ok, nil
}

func newTestRouter() (http.Handler, *auth.Manager, *fakeBlacklist) {
	repo := &fakeRepo{users: map[string]ds.ProviderRouterUser{}}
	tokens := auth.NewManager("test-secret", time.Hour)
	blacklist := &fakeBlacklist{tokens: map[string]string{}}
	return NewRouter(handler.New(repo, repo, tokens, blacklist)), tokens, blacklist
}

func do(router http.Handler, method, url, token, contentType string, body *bytes.Buffer) *httptest.ResponseRecorder {
	if body == nil {
		body = &bytes.Buffer{}
	}
	request := httptest.NewRequest(method, url, body)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
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

// Все 10 маршрутов зарегистрированы и отвечают ожидаемыми кодами (с токеном пользователя 1).
func TestRoutes(t *testing.T) {
	router, tokens, _ := newTestRouter()
	token, _, _ := tokens.Issue(1, "student@provider-routers.local")
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
		{"GET", "/api/unknown", "", "", nil, 404},
		{"PATCH", "/api/provider_routers/106", "", "", nil, 404}, // неизвестный метод → JSON 404 из catch-all
	}
	for _, c := range cases {
		body := c.raw
		if body == nil {
			body = bytes.NewBufferString(c.body)
		}
		recorder := do(router, c.method, c.url, token, c.contentType, body)
		if recorder.Code != c.want {
			t.Errorf("%s %s: код %d, ожидался %d (%s)", c.method, c.url, recorder.Code, c.want, recorder.Body.String())
		}
		if ct := recorder.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Errorf("%s %s: Content-Type %q", c.method, c.url, ct)
		}
	}
}

// Гость: чтение доступно, методы создателя — 401.
func TestGuestAccess(t *testing.T) {
	router, _, _ := newTestRouter()
	for _, url := range []string{"/api/provider_routers", "/api/provider_routers/feed"} {
		if code := do(router, "GET", url, "", "", nil).Code; code != 200 {
			t.Errorf("гость GET %s: %d, ожидался 200", url, code)
		}
	}
	for _, c := range []struct{ method, url string }{
		{"GET", "/api/provider_routers/draft"},
		{"POST", "/api/provider_routers"},
		{"PUT", "/api/provider_routers/106/publish"},
		{"DELETE", "/api/provider_routers/106"},
		{"POST", "/api/provider_routers/101/like"},
		{"POST", "/api/provider_router_users/logout"},
	} {
		if code := do(router, c.method, c.url, "", "", nil).Code; code != 401 {
			t.Errorf("гость %s %s: %d, ожидался 401", c.method, c.url, code)
		}
	}
	if code := do(router, "GET", "/api/provider_routers/draft", "not-a-jwt", "", nil).Code; code != 401 {
		t.Errorf("поддельный токен: %d, ожидался 401", code)
	}
}

// Вход → публикация с токеном → выход → тот же токен больше не принимается.
func TestLoginLogout(t *testing.T) {
	router, _, blacklist := newTestRouter()
	register := bytes.NewBufferString(`{"email":"ivan@b.ru","display_name":"Иван","password":"secret1"}`)
	if code := do(router, "POST", "/api/provider_router_users/register", "", "application/json", register).Code; code != 201 {
		t.Fatalf("регистрация: %d", code)
	}
	login := do(router, "POST", "/api/provider_router_users/login", "", "application/json", bytes.NewBufferString(`{"email":"ivan@b.ru","password":"secret1"}`))
	var response struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(login.Body.Bytes(), &response); err != nil || response.AccessToken == "" || response.TokenType != "Bearer" {
		t.Fatalf("вход: %s", login.Body.String())
	}
	publish := `{"description":"d","bandwidth_mbps":1,"cost_rub":1}`
	if code := do(router, "PUT", "/api/provider_routers/106/publish", response.AccessToken, "application/json", bytes.NewBufferString(publish)).Code; code != 200 {
		t.Fatalf("публикация с токеном: %d", code)
	}
	if code := do(router, "POST", "/api/provider_router_users/logout", response.AccessToken, "", nil).Code; code != 200 {
		t.Fatalf("выход: %d", code)
	}
	if blacklist.tokens[response.AccessToken] != "ivan@b.ru" {
		t.Fatalf("токен не попал в blacklist: %v", blacklist.tokens)
	}
	if code := do(router, "PUT", "/api/provider_routers/106/publish", response.AccessToken, "application/json", bytes.NewBufferString(publish)).Code; code != 401 {
		t.Fatalf("после выхода токен должен отклоняться, код %d", code)
	}
	if code := do(router, "GET", "/api/provider_routers", response.AccessToken, "", nil).Code; code != 200 {
		t.Fatalf("список доступен и после выхода: %d", code)
	}
}

func TestSwagger(t *testing.T) {
	router, _, _ := newTestRouter()
	doc := do(router, "GET", "/swagger/doc.json", "", "", nil)
	if doc.Code != 200 || !bytes.Contains(doc.Body.Bytes(), []byte(`"BearerAuth"`)) {
		t.Fatalf("doc.json: %d", doc.Code)
	}
	if page := do(router, "GET", "/swagger/", "", "", nil); page.Code != 200 || !bytes.Contains(page.Body.Bytes(), []byte("swagger-ui")) {
		t.Fatalf("страница Swagger: %d", page.Code)
	}
}
