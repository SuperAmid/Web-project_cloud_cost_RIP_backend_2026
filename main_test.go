package main

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testApp() *providerRoutersApp {
	return &providerRoutersApp{
		templates:       template.Must(template.ParseGlob("templates/*.html")),
		mediaURL:        "http://minio.test/provider-media",
		providerRouters: providerRouters,
	}
}

func get(t *testing.T, handler http.HandlerFunc, url string) (int, string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler(recorder, httptest.NewRequest(http.MethodGet, url, nil))
	return recorder.Code, recorder.Body.String()
}

func TestGridFilterByBandwidth(t *testing.T) {
	code, body := get(t, testApp().gridHandler, "/provider_routers?minBandwidthMbps=10000")
	if code != http.StatusOK {
		t.Fatalf("код %d", code)
	}
	if !strings.Contains(body, "Core Backbone One") || !strings.Contains(body, "North Ring Hub") {
		t.Fatal("ожидались маршрутизаторы с пропускной способностью от 10000 Мбит/с")
	}
	if strings.Contains(body, "Harbor Residence Gateway") {
		t.Fatal("маршрутизатор на 1000 Мбит/с не должен пройти фильтр")
	}
	if !strings.Contains(body, `value="10000"`) {
		t.Fatal("значение фильтра должно сохраняться после запроса")
	}
}

func TestFeedNextAndDeletedHidden(t *testing.T) {
	app := testApp()
	code, body := get(t, app.feedHandler, "/provider_routers/feed?id=102&next=true")
	if code != http.StatusOK || !strings.Contains(body, "Harbor Residence Gateway") {
		t.Fatalf("после 102 должна открываться 103, код %d", code)
	}
	if code, _ := get(t, app.feedHandler, "/provider_routers/feed?id=105"); code != http.StatusNotFound {
		t.Fatalf("удалённая запись не должна открываться, код %d", code)
	}
}

func TestDraftShowsDefaultMedia(t *testing.T) {
	_, body := get(t, testApp().draftHandler, "/provider_routers/draft")
	for _, url := range []string{
		"http://minio.test/provider-media/provider_routers/draft.svg",
		"http://minio.test/provider-media/videos/draft-loop.mp4",
	} {
		if !strings.Contains(body, url) {
			t.Fatalf("на странице добавления нет медиа по умолчанию %s", url)
		}
	}
}
