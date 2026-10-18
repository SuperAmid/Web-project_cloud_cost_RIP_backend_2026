package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMediaFallbackToSSRDefaults(t *testing.T) {
	minio := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/provider_routers/core.svg" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer minio.Close()
	media := newMediaResolver(minio.URL)

	if got := media.URL("provider_routers/core.svg", defaultImageURL); got != minio.URL+"/provider_routers/core.svg" {
		t.Fatalf("доступный файл должен браться из MinIO, получено %s", got)
	}
	if got := media.URL(notUploadedImageKey, defaultImageURL); got != defaultImageURL {
		t.Fatalf("неверный ключ должен заменяться фото по умолчанию, получено %s", got)
	}
	if got := media.URL("", defaultVideoURL); got != defaultVideoURL {
		t.Fatalf("пустой ключ должен заменяться видео по умолчанию, получено %s", got)
	}
}

func TestViewHandlesNullFields(t *testing.T) {
	app := &providerRoutersApp{media: newMediaResolver("http://127.0.0.1:1")}
	view := app.toView(ProviderRouter{ID: 7, Name: "Черновик", ImageKey: notUploadedImageKey})
	if view.Description != "" || view.BandwidthMbps != 0 || view.CostRub != 0 {
		t.Fatal("NULL-поля черновика должны выводиться пустыми")
	}
	if view.ImageURL != defaultImageURL || view.VideoURL != defaultVideoURL {
		t.Fatal("для черновика без файлов должны показываться фото и видео по умолчанию")
	}
}
