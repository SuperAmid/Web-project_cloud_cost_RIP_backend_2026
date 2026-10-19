package handler

import (
	"errors"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"provider_routers/internal/ds"
	"provider_routers/internal/repository"
	"provider_routers/internal/serializer"
)

func deref[T any](value *T) T {
	var zero T
	if value == nil {
		return zero
	}
	return *value
}

func likedBy(item ds.ProviderRouter, userID uint) bool {
	for _, like := range item.Likes {
		if like.UserID == userID {
			return true
		}
	}
	return false
}

func (h *Handler) toDraft(item ds.ProviderRouter) serializer.ProviderRouterDraft {
	return serializer.ProviderRouterDraft{
		ID:            item.ID,
		Name:          item.Name,
		Description:   item.Description,
		Status:        item.Status,
		BandwidthMbps: item.BandwidthMbps,
		CostRub:       item.CostRub,
		ImageKey:      item.ImageKey,
		VideoKey:      item.VideoKey,
		ImageURL:      h.repo.MediaURL(item.ImageKey, repository.DefaultImageKey),
		VideoURL:      h.repo.MediaURL(item.VideoKey, repository.DefaultVideoKey),
		CreatedAt:     item.CreatedAt,
		PublishedAt:   item.PublishedAt,
	}
}

// GetProviderRouters — GET /api/provider_routers?min_bandwidth_mbps=&max_cost_rub=
//
//	@Summary		Список маршрутизаторов
//	@Description	Только опубликованные, фильтры выполняются в БД. is_own = 1, если создатель — текущий пользователь (для гостя 0). Доступно без авторизации.
//	@Tags			provider_routers
//	@Produce		json
//	@Param			min_bandwidth_mbps	query		int	false	"Минимальная пропускная способность, Мбит/с"
//	@Param			max_cost_rub		query		int	false	"Максимальная стоимость, ₽"
//	@Success		200					{array}		serializer.ProviderRouterListItem
//	@Failure		400					{object}	serializer.ErrorResponse
//	@Security		BearerAuth
//	@Router			/provider_routers [get]
func (h *Handler) GetProviderRouters(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	var filter repository.ProviderRouterFilter
	var err error
	if filter.MinBandwidthMbps, err = optionalNonNegativeInt(r, "min_bandwidth_mbps"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if filter.MaxCostRub, err = optionalNonNegativeInt(r, "max_cost_rub"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	rows, err := h.repo.ListProviderRouters(filter)
	if err != nil {
		writeRepositoryError(w, err, nil)
		return
	}
	items := make([]serializer.ProviderRouterListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, serializer.ProviderRouterListItem{
			ID:            row.ID,
			Name:          row.Name,
			Description:   deref(row.Description),
			BandwidthMbps: deref(row.BandwidthMbps),
			CostRub:       deref(row.CostRub),
			ImageURL:      h.repo.MediaURL(row.ImageKey, repository.DefaultImageKey),
			VideoURL:      h.repo.MediaURL(row.VideoKey, repository.DefaultVideoKey),
			LikesCount:    int64(len(row.Likes)),
			IsOwn:         boolToInt(row.CreatedByID == userID),
			PublishedAt:   row.PublishedAt,
		})
	}
	writeJSON(w, http.StatusOK, items)
}

// GetProviderRouterFeed — GET /api/provider_routers/feed[?id=102[&next=true]]
//
//	@Summary		Лента: одна карточка
//	@Description	Без id — первая опубликованная, с id — эта, с id и next=true — следующая (после последней — первая). is_liked = 1, если текущий пользователь лайкнул (для гостя 0). Доступно без авторизации.
//	@Tags			provider_routers
//	@Produce		json
//	@Param			id		query		int		false	"Id маршрутизатора"
//	@Param			next	query		bool	false	"Следующая после id"
//	@Success		200		{object}	serializer.ProviderRouterFeedItem
//	@Failure		400		{object}	serializer.ErrorResponse
//	@Failure		404		{object}	serializer.ErrorResponse
//	@Security		BearerAuth
//	@Router			/provider_routers/feed [get]
func (h *Handler) GetProviderRouterFeed(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	var id *uint
	if raw := strings.TrimSpace(r.URL.Query().Get("id")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil || parsed == 0 {
			writeError(w, http.StatusBadRequest, "id должен быть положительным целым числом")
			return
		}
		value := uint(parsed)
		id = &value
	}
	next := false
	if raw := r.URL.Query().Get("next"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "next должен быть true или false")
			return
		}
		next = parsed
	}
	if next && id == nil {
		writeError(w, http.StatusBadRequest, "next=true требует параметр id")
		return
	}
	row, err := h.repo.GetFeedProviderRouter(id, next)
	if err != nil {
		writeRepositoryError(w, err, map[error]string{repository.ErrNotFound: "опубликованный маршрутизатор не найден"})
		return
	}
	writeJSON(w, http.StatusOK, serializer.ProviderRouterFeedItem{
		ID:            row.ID,
		Name:          row.Name,
		Description:   deref(row.Description),
		BandwidthMbps: deref(row.BandwidthMbps),
		CostRub:       deref(row.CostRub),
		ImageURL:      h.repo.MediaURL(row.ImageKey, repository.DefaultImageKey),
		VideoURL:      h.repo.MediaURL(row.VideoKey, repository.DefaultVideoKey),
		LikesCount:    int64(len(row.Likes)),
		IsLiked:       boolToInt(likedBy(row, userID)),
		CreatorName:   row.CreatedBy.DisplayName,
		PublishedAt:   row.PublishedAt,
	})
}

// GetProviderRouterDraft — GET /api/provider_routers/draft (id не указывается)
//
//	@Summary		Черновик текущего пользователя
//	@Description	Не более одного черновика у пользователя, id не указывается.
//	@Tags			provider_routers
//	@Produce		json
//	@Success		200	{object}	serializer.ProviderRouterDraft
//	@Failure		401	{object}	serializer.ErrorResponse
//	@Failure		404	{object}	serializer.ErrorResponse
//	@Security		BearerAuth
//	@Router			/provider_routers/draft [get]
func (h *Handler) GetProviderRouterDraft(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	draft, err := h.repo.GetDraftProviderRouter(userID)
	if err != nil {
		writeRepositoryError(w, err, map[error]string{repository.ErrNotFound: "у пользователя нет черновика"})
		return
	}
	writeJSON(w, http.StatusOK, h.toDraft(draft))
}

// CreateProviderRouter — POST /api/provider_routers (multipart/form-data: name, image, video)
//
//	@Summary		Создание черновика с фото и видео
//	@Description	Файлы сохраняются в MinIO под латинскими именами, создатель — пользователь из токена.
//	@Tags			provider_routers
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			name	formData	string	true	"Название"
//	@Param			image	formData	file	true	"Фото: jpg, jpeg, png, webp, svg до 10 МБ"
//	@Param			video	formData	file	true	"Видео: mp4, webm, mov до 50 МБ"
//	@Success		201		{object}	serializer.ProviderRouterDraft
//	@Failure		400		{object}	serializer.ErrorResponse
//	@Failure		401		{object}	serializer.ErrorResponse
//	@Failure		409		{object}	serializer.ErrorResponse
//	@Security		BearerAuth
//	@Router			/provider_routers [post]
func (h *Handler) CreateProviderRouter(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	r.Body = http.MaxBytesReader(w, r.Body, 70<<20)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "ожидается multipart/form-data с полями name, image, video (не больше 70 МБ)")
		return
	}
	defer r.MultipartForm.RemoveAll()
	for field := range r.MultipartForm.Value {
		if field != "name" {
			writeError(w, http.StatusBadRequest, "поле "+field+" передавать нельзя: при создании указываются только name, image, video")
			return
		}
	}
	for field := range r.MultipartForm.File {
		if field != "image" && field != "video" {
			writeError(w, http.StatusBadRequest, "файл "+field+" не ожидается: допустимы image и video")
			return
		}
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" || utf8.RuneCountInString(name) > 120 {
		writeError(w, http.StatusBadRequest, "name обязателен, не длиннее 120 символов")
		return
	}
	image, err := singleFile(r.MultipartForm, "image")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	video, err := singleFile(r.MultipartForm, "video")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, _, err := repository.ProviderRouterImage.ValidateFile(image); err != nil {
		writeError(w, http.StatusBadRequest, "image: "+err.Error())
		return
	}
	if _, _, err := repository.ProviderRouterVideo.ValidateFile(video); err != nil {
		writeError(w, http.StatusBadRequest, "video: "+err.Error())
		return
	}
	created, err := h.repo.CreateDraftProviderRouter(r.Context(), userID, name, image, video)
	if err != nil {
		writeRepositoryError(w, err, map[error]string{repository.ErrConflict: "у пользователя уже есть черновик — опубликуйте или удалите его"})
		return
	}
	w.Header().Set("Location", "/api/provider_routers/draft")
	writeJSON(w, http.StatusCreated, h.toDraft(created))
}

func singleFile(form *multipart.Form, field string) (*multipart.FileHeader, error) {
	files := form.File[field]
	if len(files) != 1 {
		return nil, errors.New("нужно передать ровно один файл в поле " + field)
	}
	return files[0], nil
}

// PublishProviderRouter — PUT /api/provider_routers/{id}/publish
//
//	@Summary		Публикация своего черновика
//	@Description	draft → published, published_at ставит сервер. В той же транзакции заполняются описание и два поля по теме.
//	@Tags			provider_routers
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int										true	"Id черновика"
//	@Param			body	body		serializer.ProviderRouterPublishRequest	true	"Описание и поля по теме"
//	@Success		200		{object}	serializer.ProviderRouterDraft
//	@Failure		400		{object}	serializer.ErrorResponse
//	@Failure		401		{object}	serializer.ErrorResponse
//	@Failure		403		{object}	serializer.ErrorResponse
//	@Failure		404		{object}	serializer.ErrorResponse
//	@Failure		409		{object}	serializer.ErrorResponse
//	@Security		BearerAuth
//	@Router			/provider_routers/{id}/publish [put]
func (h *Handler) PublishProviderRouter(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var request serializer.ProviderRouterPublishRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	fields, err := validatePublish(request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	published, err := h.repo.PublishProviderRouter(userID, id, fields)
	if err != nil {
		writeRepositoryError(w, err, map[error]string{
			repository.ErrNotFound:  "маршрутизатор не найден",
			repository.ErrForbidden: "опубликовать можно только свой черновик",
			repository.ErrConflict:  "опубликовать можно только запись в статусе draft",
		})
		return
	}
	writeJSON(w, http.StatusOK, h.toDraft(published))
}

func validatePublish(request serializer.ProviderRouterPublishRequest) (repository.ProviderRouterPublish, error) {
	var fields repository.ProviderRouterPublish
	if request.Description == nil || strings.TrimSpace(*request.Description) == "" {
		return fields, errors.New("description обязателен")
	}
	fields.Description = strings.TrimSpace(*request.Description)
	if utf8.RuneCountInString(fields.Description) > 500 {
		return fields, errors.New("description не длиннее 500 символов")
	}
	if request.BandwidthMbps == nil || *request.BandwidthMbps <= 0 {
		return fields, errors.New("bandwidth_mbps обязателен и должен быть больше 0")
	}
	if request.CostRub == nil || *request.CostRub <= 0 {
		return fields, errors.New("cost_rub обязателен и должен быть больше 0")
	}
	fields.BandwidthMbps, fields.CostRub = *request.BandwidthMbps, *request.CostRub
	return fields, nil
}

// DeleteProviderRouter — DELETE /api/provider_routers/{id} (только soft delete своей записи)
//
//	@Summary		Логическое удаление своей записи
//	@Description	status = deleted, запись больше не отдаётся клиенту.
//	@Tags			provider_routers
//	@Produce		json
//	@Param			id	path		int	true	"Id маршрутизатора"
//	@Success		200	{object}	serializer.ProviderRouterDraft
//	@Failure		401	{object}	serializer.ErrorResponse
//	@Failure		403	{object}	serializer.ErrorResponse
//	@Failure		404	{object}	serializer.ErrorResponse
//	@Security		BearerAuth
//	@Router			/provider_routers/{id} [delete]
func (h *Handler) DeleteProviderRouter(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	deleted, err := h.repo.DeleteProviderRouter(userID, id)
	if err != nil {
		writeRepositoryError(w, err, map[error]string{
			repository.ErrNotFound:  "маршрутизатор не найден",
			repository.ErrForbidden: "удалить можно только свой маршрутизатор",
		})
		return
	}
	writeJSON(w, http.StatusOK, h.toDraft(deleted))
}

// LikeProviderRouter — POST /api/provider_routers/{id}/like, тело {"like": 1} или {"like": 0}
//
//	@Summary		Лайк от текущего пользователя
//	@Description	like = 1 ставит лайк, 0 снимает.
//	@Tags			provider_routers
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int										true	"Id маршрутизатора"
//	@Param			body	body		serializer.ProviderRouterLikeRequest	true	"0 или 1"
//	@Success		200		{object}	serializer.ProviderRouterLikeResponse
//	@Failure		400		{object}	serializer.ErrorResponse
//	@Failure		401		{object}	serializer.ErrorResponse
//	@Failure		404		{object}	serializer.ErrorResponse
//	@Security		BearerAuth
//	@Router			/provider_routers/{id}/like [post]
func (h *Handler) LikeProviderRouter(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var request serializer.ProviderRouterLikeRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if request.Like == nil || (*request.Like != 0 && *request.Like != 1) {
		writeError(w, http.StatusBadRequest, "like должен быть 0 (снять лайк) или 1 (поставить)")
		return
	}
	likesCount, err := h.repo.SetProviderRouterLike(userID, id, *request.Like == 1)
	if err != nil {
		writeRepositoryError(w, err, map[error]string{repository.ErrNotFound: "опубликованный маршрутизатор не найден"})
		return
	}
	writeJSON(w, http.StatusOK, serializer.ProviderRouterLikeResponse{ProviderRouterID: id, IsLiked: *request.Like, LikesCount: likesCount})
}
