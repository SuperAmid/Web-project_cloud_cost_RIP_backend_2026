package handler

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"provider_routers/internal/ds"
	"provider_routers/internal/repository"
	"provider_routers/internal/serializer"
)

func toUserResponse(user ds.ProviderRouterUser) serializer.ProviderRouterUserResponse {
	return serializer.ProviderRouterUserResponse{ID: user.ID, Email: user.Email, DisplayName: user.DisplayName}
}

// RegisterProviderRouterUser — POST /api/provider_router_users/register
//
//	@Summary		Регистрация
//	@Description	Пароль хранится как bcrypt-хеш.
//	@Tags			provider_router_users
//	@Accept			json
//	@Produce		json
//	@Param			body	body		serializer.ProviderRouterUserRegisterRequest	true	"email, имя, пароль 6–72 байта"
//	@Success		201		{object}	serializer.ProviderRouterUserResponse
//	@Failure		400		{object}	serializer.ErrorResponse
//	@Failure		409		{object}	serializer.ErrorResponse
//	@Router			/provider_router_users/register [post]
func (h *Handler) RegisterProviderRouterUser(w http.ResponseWriter, r *http.Request) {
	var request serializer.ProviderRouterUserRegisterRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	email := strings.ToLower(strings.TrimSpace(request.Email))
	if address, err := mail.ParseAddress(email); err != nil || address.Address != email || len(email) > 120 {
		writeError(w, http.StatusBadRequest, "email некорректен")
		return
	}
	displayName := strings.TrimSpace(request.DisplayName)
	if displayName == "" || utf8.RuneCountInString(displayName) > 80 {
		writeError(w, http.StatusBadRequest, "display_name обязателен, не длиннее 80 символов")
		return
	}
	if len(request.Password) < 6 || len(request.Password) > 72 {
		writeError(w, http.StatusBadRequest, "password должен быть от 6 до 72 байт")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		writeRepositoryError(w, err, nil)
		return
	}
	user := ds.ProviderRouterUser{Email: email, DisplayName: displayName, PasswordHash: string(hash)}
	if err := h.users.CreateProviderRouterUser(&user); err != nil {
		writeRepositoryError(w, err, map[error]string{repository.ErrConflict: "пользователь с таким email уже зарегистрирован"})
		return
	}
	writeJSON(w, http.StatusCreated, toUserResponse(user))
}

// LoginProviderRouterUser — POST /api/provider_router_users/login: проверяет
// email и пароль и выдаёт JWT. Токен передаётся в заголовке Authorization: Bearer <токен>.
//
//	@Summary		Вход: выдача JWT
//	@Description	Токен из access_token передаётся в заголовке Authorization: Bearer <токен>.
//	@Tags			provider_router_users
//	@Accept			json
//	@Produce		json
//	@Param			body	body		serializer.ProviderRouterUserLoginRequest	true	"email и пароль"
//	@Success		200		{object}	serializer.ProviderRouterUserLoginResponse
//	@Failure		400		{object}	serializer.ErrorResponse
//	@Failure		401		{object}	serializer.ErrorResponse
//	@Router			/provider_router_users/login [post]
func (h *Handler) LoginProviderRouterUser(w http.ResponseWriter, r *http.Request) {
	var request serializer.ProviderRouterUserLoginRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	user, err := h.users.FindProviderRouterUserByEmail(strings.ToLower(strings.TrimSpace(request.Email)))
	if err == nil && user.PasswordHash != "" {
		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password))
	} else if err == nil {
		err = errors.New("password not set")
	}
	if err != nil {
		writeError(w, http.StatusUnauthorized, "неверный email или пароль")
		return
	}
	token, expires, err := h.tokens.Issue(user.ID, user.Email)
	if err != nil {
		writeRepositoryError(w, err, nil)
		return
	}
	writeJSON(w, http.StatusOK, serializer.ProviderRouterUserLoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(time.Until(expires).Seconds()),
		User:        toUserResponse(user),
	})
}

// LogoutProviderRouterUser — POST /api/provider_router_users/logout: токен
// попадает в blacklist Redis до конца срока действия и больше не принимается.
//
//	@Summary		Выход
//	@Description	Текущий токен попадает в blacklist Redis до конца срока действия.
//	@Tags			provider_router_users
//	@Produce		json
//	@Success		200	{object}	serializer.MessageResponse
//	@Failure		401	{object}	serializer.ErrorResponse
//	@Security		BearerAuth
//	@Router			/provider_router_users/logout [post]
func (h *Handler) LogoutProviderRouterUser(w http.ResponseWriter, r *http.Request) {
	s, _ := currentSession(r)
	ttl := time.Until(s.claims.ExpiresAt.Time)
	if ttl > 0 {
		if err := h.blacklist.WriteJWTToBlacklist(r.Context(), s.token, s.claims.Email, ttl); err != nil {
			writeRepositoryError(w, err, nil)
			return
		}
	}
	writeJSON(w, http.StatusOK, serializer.MessageResponse{Message: "выход выполнен, токен больше не действует"})
}
