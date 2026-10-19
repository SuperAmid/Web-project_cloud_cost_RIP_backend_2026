// Package api связывает URL веб-сервиса с обработчиками.
package api

import (
	"log"
	"net/http"
	"time"

	"provider_routers/internal/handler"
)

// NewRouter — 10 методов API под /api и Swagger UI под /swagger/.
// Гостю доступно только чтение (список и лента), остальное — после входа (JWT).
func NewRouter(h *handler.Handler) http.Handler {
	mux := http.NewServeMux()

	// Домен маршрутизатора (услуги)
	mux.HandleFunc("GET /api/provider_routers", h.GetProviderRouters)                                // список с фильтром
	mux.HandleFunc("GET /api/provider_routers/feed", h.GetProviderRouterFeed)                        // лента
	mux.HandleFunc("GET /api/provider_routers/draft", h.RequireUser(h.GetProviderRouterDraft))       // черновик
	mux.HandleFunc("POST /api/provider_routers", h.RequireUser(h.CreateProviderRouter))              // создание + фото и видео
	mux.HandleFunc("PUT /api/provider_routers/{id}/publish", h.RequireUser(h.PublishProviderRouter)) // публикация
	mux.HandleFunc("DELETE /api/provider_routers/{id}", h.RequireUser(h.DeleteProviderRouter))       // soft delete
	mux.HandleFunc("POST /api/provider_routers/{id}/like", h.RequireUser(h.LikeProviderRouter))      // лайк 0/1

	// Домен пользователя
	mux.HandleFunc("POST /api/provider_router_users/register", h.RegisterProviderRouterUser)
	mux.HandleFunc("POST /api/provider_router_users/login", h.LoginProviderRouterUser)
	mux.HandleFunc("POST /api/provider_router_users/logout", h.RequireUser(h.LogoutProviderRouterUser))

	registerSwagger(mux)
	mux.HandleFunc("/", h.NotFound)
	return logRequests(h.Authenticate(mux))
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.RequestURI(), recorder.status, time.Since(started).Round(time.Millisecond))
	})
}
