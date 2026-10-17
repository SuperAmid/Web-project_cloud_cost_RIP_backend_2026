// provider_routers — ЛР3: веб-сервис (REST API) для маршрутизаторов сети провайдера.
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"provider_routers/internal/api"
	"provider_routers/internal/config"
	"provider_routers/internal/handler"
	"provider_routers/internal/repository"
)

func main() {
	cfg := config.FromEnv()

	media, err := repository.NewMedia(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioBucket, cfg.MinioPublicURL)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := media.EnsureBucket(ctx); err != nil {
		log.Printf("MinIO недоступен (%v): загрузка файлов не будет работать, пока он не запущен", err)
	}
	cancel()

	repo, err := repository.New(cfg.DatabaseURL, media)
	if err != nil {
		log.Fatal(err)
	}
	if err := repo.MigrateAndSeed(); err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.NewRouter(handler.New(repo)),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("provider_routers API (ЛР3) слушает http://localhost%s/api", cfg.Addr)
	log.Fatal(server.ListenAndServe())
}
