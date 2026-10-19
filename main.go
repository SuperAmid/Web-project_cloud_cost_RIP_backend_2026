// provider_routers — ЛР4: веб-сервис (REST API) маршрутизаторов сети провайдера
// с авторизацией по JWT, blacklist токенов в Redis и описанием методов в Swagger.
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"provider_routers/internal/api"
	"provider_routers/internal/auth"
	"provider_routers/internal/config"
	"provider_routers/internal/handler"
	"provider_routers/internal/redis"
	"provider_routers/internal/repository"
)

// @title						provider_routers API
// @version					1.0
// @description				Маршрутизаторы сети провайдера: список, лента, черновик, публикация, лайки.
// @description				Без авторизации доступно только чтение. Чтобы вызвать методы создателя,
// @description				выполните вход и нажмите Authorize: Bearer <access_token>.
// @host						localhost:8080
// @BasePath					/api
// @schemes					http
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Введите: Bearer <access_token из ответа входа>
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
	blacklist, err := redis.New(ctx, cfg.RedisAddr, cfg.RedisPassword)
	cancel()
	if err != nil {
		log.Fatalf("%v — запустите docker compose up -d", err)
	}

	repo, err := repository.New(cfg.DatabaseURL, media)
	if err != nil {
		log.Fatal(err)
	}
	if err := repo.MigrateAndSeed("migrations"); err != nil {
		log.Fatal(err)
	}

	tokens := auth.NewManager(cfg.JWTSecret, cfg.JWTTTL)
	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.NewRouter(handler.New(repo, repo, tokens, blacklist)),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("provider_routers API (ЛР4) слушает http://localhost%s/api, Swagger: http://localhost%s/swagger/", cfg.Addr, cfg.Addr)
	log.Fatal(server.ListenAndServe())
}
