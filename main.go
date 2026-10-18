// provider_routers — ЛР2: PostgreSQL + GORM + серверные шаблоны.
package main

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// fixedCreatorEmail — в ЛР2 авторизации нет, создатель черновиков зафиксирован.
const fixedCreatorEmail = "student@provider-routers.local"

// Неверные ключи, которые по заданию пишутся в БД по кнопке «Далее»: таких файлов
// в MinIO нет, поэтому на страницах показываются фото и видео по умолчанию с SSR-сервера.
const (
	notUploadedImageKey = "provider_routers/not-uploaded.jpg"
	notUploadedVideoKey = "videos/not-uploaded.mp4"
)

// providerRouterView — данные одной карточки для шаблонов.
type providerRouterView struct {
	ID            uint
	Name          string
	Description   string
	ImageKey      string
	VideoKey      string
	ImageURL      string
	VideoURL      string
	BandwidthMbps int
	CostRub       int
	Likes         int
}

type providerRoutersApp struct {
	db        *gorm.DB
	templates *template.Template
	media     *mediaResolver
}

func main() {
	db, err := gorm.Open(postgres.Open(envOr("DATABASE_URL", "host=127.0.0.1 user=provider_routers_user password=provider_routers_password dbname=provider_routers port=5433 sslmode=disable")), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	app := &providerRoutersApp{
		db: db,
		templates: template.Must(template.New("").Funcs(template.FuncMap{
			// clock — время для строки состояния телефона (рендерится на сервере, без JavaScript)
			"clock": func() string { return time.Now().Format("15:04") },
		}).ParseGlob("templates/*.html")),
		media: newMediaResolver(envOr("MINIO_PUBLIC_URL", "http://localhost:9000/provider-media")),
	}
	if err := applyMigrations(db, "migrations"); err != nil {
		log.Fatal(err)
	}
	if err := app.seed(); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/provider_routers/feed", http.StatusFound)
	})
	// Шесть HTTP-методов ЛР2: три GET и три POST.
	mux.HandleFunc("GET /provider_routers/feed", app.feedHandler)                         // ORM, одна строка
	mux.HandleFunc("GET /provider_routers/draft", app.draftHandler)                       // ORM
	mux.HandleFunc("GET /provider_routers", app.gridHandler)                              // ORM, фильтр
	mux.HandleFunc("POST /provider_routers/draft/create", app.createDraftHandler)         // ORM Create
	mux.HandleFunc("POST /provider_routers/draft/publish", app.publishDraftHandler)       // ORM Updates
	mux.HandleFunc("POST /provider_routers/{id}/delete", app.deleteProviderRouterHandler) // SQL UPDATE через курсор
	addr := envOr("APP_ADDR", ":8080")
	log.Printf("provider_routers ЛР2 слушает http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// seed наполняет пустую БД начальными данными (те же, что в db/seed.sql).
func (a *providerRoutersApp) seed() error {
	var count int64
	if err := a.db.Model(&ProviderRouter{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	student := ProviderRouterUser{Email: fixedCreatorEmail, DisplayName: "Студент РИП"}
	operator := ProviderRouterUser{Email: "operator@provider-routers.local", DisplayName: "Оператор сети"}
	viewer := ProviderRouterUser{Email: "viewer@provider-routers.local", DisplayName: "Наблюдатель"}
	if err := a.db.Create(&[]*ProviderRouterUser{&student, &operator, &viewer}).Error; err != nil {
		return err
	}
	now := time.Now()
	text := func(s string) *string { return &s }
	num := func(n int) *int { return &n }
	items := []ProviderRouter{
		{Name: "Core Backbone One", Description: text("Центральный маршрутизатор ядра провайдера для магистрального узла и распределения трафика между сегментами сети."), Status: statusPublished, ImageKey: "provider_routers/core.svg", VideoKey: "videos/core-loop.mp4", BandwidthMbps: num(100000), CostRub: num(980000), CreatedByID: operator.ID, PublishedAt: &now},
		{Name: "North Ring Hub", Description: text("Промежуточный маршрутизатор кольцевой сети, который агрегирует районные узлы и передаёт трафик на магистраль."), Status: statusPublished, ImageKey: "provider_routers/ring.svg", VideoKey: "videos/ring-loop.mp4", BandwidthMbps: num(10000), CostRub: num(310000), CreatedByID: operator.ID, PublishedAt: &now},
		{Name: "Harbor Residence Gateway", Description: text("Конечный маршрутизатор жилого комплекса для распределения доступа к сети между квартирами."), Status: statusPublished, ImageKey: "provider_routers/residential.svg", VideoKey: "videos/residential-loop.mp4", BandwidthMbps: num(1000), CostRub: num(72000), CreatedByID: operator.ID, PublishedAt: &now},
		{Name: "Riverside Residence Gateway", Status: statusDraft, ImageKey: notUploadedImageKey, VideoKey: notUploadedVideoKey, CreatedByID: student.ID},
		{Name: "Legacy South Edge", Description: text("Выведенный из эксплуатации пограничный маршрутизатор."), Status: statusDeleted, ImageKey: "provider_routers/ring.svg", VideoKey: "videos/ring-loop.mp4", BandwidthMbps: num(1000), CostRub: num(50000), CreatedByID: operator.ID, PublishedAt: &now},
	}
	if err := a.db.Create(&items).Error; err != nil {
		return err
	}
	return a.db.Create(&[]ProviderRouterLike{
		{ProviderRouterID: items[0].ID, UserID: student.ID},
		{ProviderRouterID: items[0].ID, UserID: viewer.ID},
		{ProviderRouterID: items[1].ID, UserID: viewer.ID},
		{ProviderRouterID: items[2].ID, UserID: student.ID},
		{ProviderRouterID: items[2].ID, UserID: operator.ID},
	}).Error
}

func (a *providerRoutersApp) currentCreator() (ProviderRouterUser, error) {
	var user ProviderRouterUser
	return user, a.db.Where("email = ?", fixedCreatorEmail).First(&user).Error
}

// feedHandler — GET ленты. Из БД всегда приходит одна строка (First → LIMIT 1):
// без id — первая опубликованная, с id — эта, с id и next=true — следующая после неё.
func (a *providerRoutersApp) feedHandler(w http.ResponseWriter, r *http.Request) {
	published := func() *gorm.DB {
		return a.db.Preload("Likes").Where("status = ?", statusPublished).Order("id")
	}
	var providerRouter ProviderRouter
	var err error
	rawID := r.URL.Query().Get("id")
	switch {
	case rawID == "":
		err = published().First(&providerRouter).Error
	default:
		id, parseErr := strconv.ParseUint(rawID, 10, 64)
		if parseErr != nil {
			http.Error(w, "некорректный id маршрутизатора", http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("next") == "true" {
			err = published().Where("id > ?", id).First(&providerRouter).Error
			if errors.Is(err, gorm.ErrRecordNotFound) { // после последней карточки — снова первая
				err = published().First(&providerRouter).Error
			}
		} else {
			err = published().Where("id = ?", id).First(&providerRouter).Error
		}
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		a.notFound(w)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	a.render(w, "feed.html", map[string]any{"ProviderRouter": a.toView(providerRouter), "Title": "Лента маршрутизаторов"})
}

// draftHandler — GET страницы добавления: форма «Далее» или заполненный черновик с «Опубликовать».
func (a *providerRoutersApp) draftHandler(w http.ResponseWriter, r *http.Request) {
	creator, err := a.currentCreator()
	if err != nil {
		serverError(w, err)
		return
	}
	page := map[string]any{
		"Title":           "Добавление маршрутизатора",
		"Error":           r.URL.Query().Get("error"),
		"DefaultImageURL": defaultImageURL,
		"DefaultVideoURL": defaultVideoURL,
	}
	var providerRouter ProviderRouter
	err = a.db.Where("created_by_id = ? AND status = ?", creator.ID, statusDraft).First(&providerRouter).Error
	switch {
	case err == nil:
		page["ProviderRouter"] = a.toView(providerRouter)
	case !errors.Is(err, gorm.ErrRecordNotFound):
		serverError(w, err)
		return
	}
	a.render(w, "draft.html", page)
}

// gridHandler — GET плитки с фильтром по минимальной пропускной способности (фильтр в БД).
func (a *providerRoutersApp) gridHandler(w http.ResponseWriter, r *http.Request) {
	rawBandwidth := strings.TrimSpace(r.URL.Query().Get("minBandwidthMbps"))
	query := a.db.Preload("Likes").Where("status = ?", statusPublished)
	if rawBandwidth != "" {
		minimum, err := strconv.Atoi(rawBandwidth)
		if err != nil || minimum < 0 {
			http.Error(w, "minBandwidthMbps должен быть неотрицательным целым числом", http.StatusBadRequest)
			return
		}
		query = query.Where("bandwidth_mbps >= ?", minimum)
	}
	var providerRouters []ProviderRouter
	if err := query.Order("id").Find(&providerRouters).Error; err != nil {
		serverError(w, err)
		return
	}
	views := make([]providerRouterView, 0, len(providerRouters))
	for _, item := range providerRouters {
		views = append(views, a.toView(item))
	}
	a.render(w, "grid.html", map[string]any{"ProviderRouters": views, "MinBandwidth": rawBandwidth, "Title": "Маршрутизаторы"})
}

// createDraftHandler — POST «Далее»: только название (файлы на сервер не передаются),
// в ключи медиа пишутся неверные значения — по заданию ЛР2.
func (a *providerRoutersApp) createDraftHandler(w http.ResponseWriter, r *http.Request) {
	creator, err := a.currentCreator()
	if err != nil {
		serverError(w, err)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		redirectWithError(w, r, "Введите название маршрутизатора")
		return
	}
	providerRouter := ProviderRouter{
		Name:        name,
		Status:      statusDraft,
		ImageKey:    notUploadedImageKey,
		VideoKey:    notUploadedVideoKey,
		CreatedByID: creator.ID,
	}
	if err := a.db.Create(&providerRouter).Error; err != nil {
		redirectWithError(w, r, "У вас уже есть черновик")
		return
	}
	http.Redirect(w, r, "/provider_routers/draft", http.StatusSeeOther)
}

// publishDraftHandler — POST «Опубликовать»: краткое описание и два поля по теме, статус published.
func (a *providerRoutersApp) publishDraftHandler(w http.ResponseWriter, r *http.Request) {
	creator, err := a.currentCreator()
	if err != nil {
		serverError(w, err)
		return
	}
	var providerRouter ProviderRouter
	if err := a.db.Where("created_by_id = ? AND status = ?", creator.ID, statusDraft).First(&providerRouter).Error; err != nil {
		redirectWithError(w, r, "Черновик не найден")
		return
	}
	description := strings.TrimSpace(r.FormValue("description"))
	bandwidth, bandwidthErr := strconv.Atoi(r.FormValue("bandwidthMbps"))
	cost, costErr := strconv.Atoi(r.FormValue("costRub"))
	if description == "" || bandwidthErr != nil || costErr != nil || bandwidth <= 0 || cost <= 0 {
		redirectWithError(w, r, "Заполните описание, пропускную способность и стоимость")
		return
	}
	changes := map[string]any{
		"description":    description,
		"bandwidth_mbps": bandwidth,
		"cost_rub":       cost,
		"status":         statusPublished,
		"published_at":   time.Now(),
	}
	if err := a.db.Model(&providerRouter).Updates(changes).Error; err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/provider_routers/feed?id="+strconv.FormatUint(uint64(providerRouter.ID), 10), http.StatusSeeOther)
}

// deleteProviderRouterHandler — POST «Удалить». Без ORM: SQL UPDATE через database/sql,
// результат (id изменённой строки) читается курсором rows.Next() / rows.Scan().
func (a *providerRoutersApp) deleteProviderRouterHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	sqlDB, err := a.db.DB()
	if err != nil {
		serverError(w, err)
		return
	}
	rows, err := sqlDB.QueryContext(r.Context(),
		"UPDATE provider_routers SET status = $1 WHERE id = $2 AND status = $3 RETURNING id",
		statusDeleted, id, statusPublished)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var deletedID uint64
		if err := rows.Scan(&deletedID); err != nil {
			serverError(w, err)
			return
		}
		log.Printf("маршрутизатор %d удалён (status = deleted)", deletedID)
	}
	if err := rows.Err(); err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/provider_routers", http.StatusSeeOther)
}

func (a *providerRoutersApp) toView(item ProviderRouter) providerRouterView {
	view := providerRouterView{
		ID:       item.ID,
		Name:     item.Name,
		ImageKey: item.ImageKey,
		VideoKey: item.VideoKey,
		ImageURL: a.media.URL(item.ImageKey, defaultImageURL),
		VideoURL: a.media.URL(item.VideoKey, defaultVideoURL),
		Likes:    len(item.Likes),
	}
	if item.Description != nil {
		view.Description = *item.Description
	}
	if item.BandwidthMbps != nil {
		view.BandwidthMbps = *item.BandwidthMbps
	}
	if item.CostRub != nil {
		view.CostRub = *item.CostRub
	}
	return view
}

func (v providerRouterView) BandwidthLabel() string {
	return fmt.Sprintf("%d Мбит/с", v.BandwidthMbps)
}
func (v providerRouterView) CostLabel() string { return fmt.Sprintf("%d ₽", v.CostRub) }
func (v providerRouterView) ShortDescription() string {
	letters := []rune(v.Description)
	if len(letters) <= 92 {
		return v.Description
	}
	return string(letters[:92]) + "…"
}

// notFound — страница 404 в том же оформлении (рамка телефона и меню),
// например для удалённой или несуществующей карточки ленты.
func (a *providerRoutersApp) notFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	if err := a.templates.ExecuteTemplate(w, "notfound.html", map[string]any{"Title": "Маршрутизатор не найден"}); err != nil {
		log.Printf("template notfound.html: %v", err)
	}
}

func (a *providerRoutersApp) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.templates.ExecuteTemplate(w, name, data); err != nil {
		serverError(w, err)
	}
}

func redirectWithError(w http.ResponseWriter, r *http.Request, message string) {
	http.Redirect(w, r, "/provider_routers/draft?error="+template.URLQueryEscaper(message), http.StatusSeeOther)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("ошибка сервера: %v", err)
	http.Error(w, "внутренняя ошибка сервера", http.StatusInternalServerError)
}
