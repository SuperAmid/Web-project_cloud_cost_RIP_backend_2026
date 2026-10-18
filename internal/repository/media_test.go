package repository

import (
	"mime/multipart"
	"strings"
	"testing"
)

// Проверка расширений и размеров файлов — без MinIO.
func TestMediaKindValidateFile(t *testing.T) {
	cases := []struct {
		kind    MediaKind
		name    string
		size    int64
		wantErr string
		wantCT  string
	}{
		{ProviderRouterImage, "router.PNG", 1024, "", "image/png"},
		{ProviderRouterImage, "router.svg", 10, "", "image/svg+xml"},
		{ProviderRouterImage, "router.gif", 1024, "недопустимое расширение", ""},
		{ProviderRouterImage, "router.png", 0, "пустой", ""},
		{ProviderRouterImage, "router.png", ProviderRouterImage.MaxBytes + 1, "больше 10 МБ", ""},
		{ProviderRouterVideo, "loop.mp4", 2048, "", "video/mp4"},
		{ProviderRouterVideo, "loop.png", 2048, "недопустимое расширение", ""},
		{ProviderRouterVideo, "loop.webm", ProviderRouterVideo.MaxBytes + 1, "больше 50 МБ", ""},
	}
	for _, c := range cases {
		_, contentType, err := c.kind.ValidateFile(&multipart.FileHeader{Filename: c.name, Size: c.size})
		switch {
		case c.wantErr == "" && err != nil:
			t.Errorf("%s: неожиданная ошибка %v", c.name, err)
		case c.wantErr != "" && (err == nil || !strings.Contains(err.Error(), c.wantErr)):
			t.Errorf("%s: ожидалась ошибка %q, получено %v", c.name, c.wantErr, err)
		case c.wantErr == "" && contentType != c.wantCT:
			t.Errorf("%s: Content-Type %q, ожидался %q", c.name, contentType, c.wantCT)
		}
	}
}

func TestGeneratedObjectNameIsLatin(t *testing.T) {
	name := generateObjectName(".png")
	if !strings.HasPrefix(name, "router-") || !strings.HasSuffix(name, ".png") {
		t.Fatalf("неожиданное имя %q", name)
	}
	for _, r := range name {
		if r > 127 {
			t.Fatalf("в имени объекта %q не латинский символ", name)
		}
	}
}

func TestMediaURLFallback(t *testing.T) {
	m := &Media{publicURL: "http://localhost:9000/provider-media"}
	if got := m.URL("", DefaultImageKey); got != "http://localhost:9000/provider-media/"+DefaultImageKey {
		t.Fatalf("пустой ключ должен давать файл по умолчанию, получено %s", got)
	}
	if got := m.URL("provider_routers/core.svg", DefaultImageKey); !strings.HasSuffix(got, "/provider_routers/core.svg") {
		t.Fatalf("неожиданный URL %s", got)
	}
}
