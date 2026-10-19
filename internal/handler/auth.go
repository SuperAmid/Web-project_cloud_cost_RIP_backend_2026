package handler

import (
	"context"
	"log"
	"net/http"
	"strings"

	"provider_routers/internal/auth"
)

const bearerPrefix = "Bearer "

type sessionKey struct{}

// session — кто выполняет запрос: данные из проверенного JWT и сам токен (нужен для выхода).
type session struct {
	claims *auth.Claims
	token  string
}

// Authenticate читает заголовок Authorization: Bearer <JWT>. Действующий токен,
// которого нет в blacklist, кладёт пользователя в контекст запроса. Без токена
// запрос идёт как от гостя: гостю доступны только методы чтения.
func (h *Handler) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, bearerPrefix) {
			next.ServeHTTP(w, r)
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix))
		claims, err := h.tokens.Parse(token)
		if err != nil {
			next.ServeHTTP(w, r) // недействительный токен = гость
			return
		}
		revoked, err := h.blacklist.IsJWTBlacklisted(r.Context(), token)
		if err != nil {
			log.Printf("redis: %v", err)
			writeError(w, http.StatusInternalServerError, "не удалось проверить токен")
			return
		}
		if revoked {
			next.ServeHTTP(w, r) // после выхода токен больше не действует
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sessionKey{}, session{claims: claims, token: token})))
	})
}

// RequireUser пропускает только авторизованного пользователя, гостю — 401.
func (h *Handler) RequireUser(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := currentSession(r); !ok {
			w.Header().Set("WWW-Authenticate", `Bearer realm="provider_routers"`)
			writeError(w, http.StatusUnauthorized, "нужна авторизация: войдите и передайте заголовок Authorization: Bearer <токен>")
			return
		}
		next(w, r)
	}
}

func currentSession(r *http.Request) (session, bool) {
	s, ok := r.Context().Value(sessionKey{}).(session)
	return s, ok
}

// currentUserID — id пользователя из JWT; 0 для гостя.
func currentUserID(r *http.Request) uint {
	if s, ok := currentSession(r); ok {
		return s.claims.UserID
	}
	return 0
}
