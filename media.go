package main

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// Фото и видео по умолчанию лежат на самом SSR-сервере (static/media) вместе с
// иконками и стилями. Они показываются, если ключ в БД пустой или файл по ссылке
// в MinIO недоступен (например, по «Далее» в таблицу записываются неверные ключи).
const (
	defaultImageURL = "/static/media/router-default.svg"
	defaultVideoURL = "/static/media/router-default.mp4"
)

// mediaResolver превращает ключ из БД в URL: MinIO, если файл доступен, иначе — файл по умолчанию.
type mediaResolver struct {
	baseURL string
	client  *http.Client
	ttl     time.Duration

	mu    sync.Mutex
	cache map[string]mediaCheck
}

type mediaCheck struct {
	ok      bool
	checked time.Time
}

func newMediaResolver(baseURL string) *mediaResolver {
	return &mediaResolver{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 1500 * time.Millisecond},
		ttl:     30 * time.Second,
		cache:   map[string]mediaCheck{},
	}
}

// URL возвращает ссылку на файл в MinIO или fallback, если ключ пуст или файла нет.
func (m *mediaResolver) URL(key, fallback string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return fallback
	}
	url := m.baseURL + "/" + key
	if !m.available(url) {
		return fallback
	}
	return url
}

// available проверяет файл запросом HEAD; результат кешируется, чтобы не дёргать MinIO на каждый рендер.
func (m *mediaResolver) available(url string) bool {
	m.mu.Lock()
	cached, found := m.cache[url]
	m.mu.Unlock()
	if found && time.Since(cached.checked) < m.ttl {
		return cached.ok
	}
	ok := false
	if response, err := m.client.Head(url); err == nil {
		response.Body.Close()
		ok = response.StatusCode == http.StatusOK
	}
	m.mu.Lock()
	m.cache[url] = mediaCheck{ok: ok, checked: time.Now()}
	m.mu.Unlock()
	return ok
}
