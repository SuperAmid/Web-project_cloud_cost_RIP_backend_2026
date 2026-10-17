package handler

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"provider_routers/internal/ds"
	"provider_routers/internal/repository"
	"provider_routers/internal/serializer"
)

func toUserResponse(user ds.ProviderRouterUser) serializer.ProviderRouterUserResponse {
	return serializer.ProviderRouterUserResponse{ID: user.ID, Email: user.Email, DisplayName: user.DisplayName, CreatedAt: user.CreatedAt}
}

// RegisterProviderRouterUser — POST /api/provider_router_users/register
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
	if err := h.repo.CreateProviderRouterUser(&user); err != nil {
		writeRepositoryError(w, err, map[error]string{repository.ErrConflict: "пользователь с таким email уже зарегистрирован"})
		return
	}
	writeJSON(w, http.StatusCreated, toUserResponse(user))
}

// LoginProviderRouterUser — POST /api/provider_router_users/login.
// Заглушка до ЛР4: проверяет пароль, но сессию/JWT не выдаёт.
func (h *Handler) LoginProviderRouterUser(w http.ResponseWriter, r *http.Request) {
	var request serializer.ProviderRouterUserLoginRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	user, err := h.repo.FindProviderRouterUserByEmail(strings.ToLower(strings.TrimSpace(request.Email)))
	if err == nil && user.PasswordHash != "" {
		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password))
	} else if err == nil {
		err = errors.New("password not set")
	}
	if err != nil {
		writeError(w, http.StatusUnauthorized, "неверный email или пароль")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "заглушка ЛР3: учётные данные верны, сессия/JWT появятся в ЛР4",
		"user":    toUserResponse(user),
	})
}

// LogoutProviderRouterUser — POST /api/provider_router_users/logout. Заглушка до ЛР4.
func (h *Handler) LogoutProviderRouterUser(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "заглушка ЛР3: деавторизация появится в ЛР4"})
}
