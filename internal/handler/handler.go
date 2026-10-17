// Package handler — HTTP-обработчики веб-сервиса. Все ответы в JSON, шаблонов нет.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"provider_routers/internal/repository"
	"provider_routers/internal/serializer"
)

type Handler struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Handler { return &Handler{repo: repo} }

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, serializer.ErrorResponse{Error: message})
}

// writeRepositoryError переводит ошибку бизнес-логики в HTTP-код.
func writeRepositoryError(w http.ResponseWriter, err error, messages map[error]string) {
	for target, status := range map[error]int{
		repository.ErrNotFound:  http.StatusNotFound,
		repository.ErrForbidden: http.StatusForbidden,
		repository.ErrConflict:  http.StatusConflict,
	} {
		if errors.Is(err, target) {
			message := messages[target]
			if message == "" {
				message = http.StatusText(status)
			}
			writeError(w, status, message)
			return
		}
	}
	log.Printf("internal error: %v", err)
	writeError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
}

// decodeJSON строго читает тело: неизвестные поля (status, id, created_by_id...) — ошибка 400.
func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("пустое тело запроса, ожидается JSON")
		}
		if field, ok := strings.CutPrefix(err.Error(), "json: unknown field "); ok {
			return fmt.Errorf("поле %s передавать нельзя", field)
		}
		return fmt.Errorf("некорректный JSON: %v", err)
	}
	if decoder.More() {
		return errors.New("в теле запроса должен быть один JSON-объект")
	}
	return nil
}

// pathID читает {id} из пути.
func pathID(r *http.Request) (uint, error) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 32)
	if err != nil || id == 0 {
		return 0, errors.New("id должен быть положительным целым числом")
	}
	return uint(id), nil
}

// optionalNonNegativeInt читает необязательный query-параметр.
func optionalNonNegativeInt(r *http.Request, name string) (*int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return nil, fmt.Errorf("%s должен быть неотрицательным целым числом", name)
	}
	return &value, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// NotFound — JSON-ответ для неизвестных адресов.
func (h *Handler) NotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "метод "+r.Method+" "+r.URL.Path+" не найден")
}
