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

const fixedCreatorEmail = "student@provider-routers.local"

const (
	statusDraft     = "draft"
	statusPublished = "published"
	statusDeleted   = "deleted"
	defaultImageKey = "provider_routers/draft.svg"
	defaultVideoKey = "videos/draft-loop.mp4"
)

// ProviderRouterUser owns drafts and can like provider routers.
type ProviderRouterUser struct {
	ID          uint   `gorm:"primaryKey"`
	Email       string `gorm:"size:120;uniqueIndex;not null"`
	DisplayName string `gorm:"size:80;not null"`
	CreatedAt   time.Time
}

func (ProviderRouterUser) TableName() string { return "provider_router_users" }

// ProviderRouter stores the two parameters required by the subject: bandwidth and cost.
type ProviderRouter struct {
	ID            uint   `gorm:"primaryKey"`
	Name          string `gorm:"size:120;not null"`
	Description   string `gorm:"size:500;not null"`
	Status        string `gorm:"size:12;index;not null"`
	ImageKey      string `gorm:"size:160;not null"`
	VideoKey      string `gorm:"size:160;not null"`
	RouterType    string `gorm:"size:24;not null"`
	BandwidthMbps int    `gorm:"not null"`
	CostRub       int    `gorm:"not null"`
	CreatedAt     time.Time
	PublishedAt   *time.Time
	CreatedByID   uint                 `gorm:"not null;index"`
	CreatedBy     ProviderRouterUser   `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`
	Likes         []ProviderRouterLike `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`
}

func (ProviderRouter) TableName() string { return "provider_routers" }

type ProviderRouterLike struct {
	ProviderRouterID uint `gorm:"primaryKey"`
	UserID           uint `gorm:"primaryKey"`
	CreatedAt        time.Time
	ProviderRouter   ProviderRouter     `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`
	User             ProviderRouterUser `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`
}

func (ProviderRouterLike) TableName() string { return "provider_router_likes" }

type ProviderRouterView struct {
	ProviderRouter
	ImageURL string
	VideoURL string
	Likes    int
}

type draftPage struct {
	ProviderRouter *ProviderRouterView
	Error          string
}

type gridPage struct {
	ProviderRouters []ProviderRouterView
	MinBandwidth    string
}

type providerRoutersApp struct {
	db        *gorm.DB
	templates *template.Template
	mediaURL  string
}

func main() {
	db, err := gorm.Open(postgres.Open(envOr("DATABASE_URL", "host=127.0.0.1 user=provider_routers_user password=provider_routers_password dbname=provider_routers port=5433 sslmode=disable")), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	app := &providerRoutersApp{
		db:        db,
		templates: template.Must(template.ParseGlob("templates/*.html")),
		mediaURL:  strings.TrimRight(envOr("MINIO_PUBLIC_URL", "http://localhost:9000/provider-media"), "/"),
	}
	if err := app.migrateAndSeed(); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/provider_routers/feed", http.StatusFound)
	})
	mux.HandleFunc("/provider_routers/feed", app.feedHandler)
	mux.HandleFunc("/provider_routers/draft", app.draftHandler)
	mux.HandleFunc("/provider_routers/draft/create", app.createDraftHandler)
	mux.HandleFunc("/provider_routers/draft/publish", app.publishDraftHandler)
	mux.HandleFunc("/provider_routers/", app.providerRouterActionHandler)
	mux.HandleFunc("/provider_routers", app.gridHandler)
	addr := envOr("APP_ADDR", ":8080")
	log.Printf("provider_routers lab2 listens on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func (a *providerRoutersApp) migrateAndSeed() error {
	if err := a.db.AutoMigrate(&ProviderRouterUser{}, &ProviderRouter{}, &ProviderRouterLike{}); err != nil {
		return err
	}
	if err := a.db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS one_draft_provider_router_per_creator ON provider_routers (created_by_id) WHERE status = 'draft'").Error; err != nil {
		return err
	}
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
	items := []ProviderRouter{
		{Name: "Core Backbone One", Description: "Центральный маршрутизатор ядра провайдера для магистрального узла и распределения трафика между сегментами сети.", Status: statusPublished, ImageKey: "provider_routers/core.svg", VideoKey: "videos/core-loop.mp4", RouterType: "central", BandwidthMbps: 100000, CostRub: 980000, CreatedByID: operator.ID, PublishedAt: &now},
		{Name: "North Ring Hub", Description: "Промежуточный маршрутизатор кольцевой сети, который агрегирует районные узлы и передаёт трафик на магистраль.", Status: statusPublished, ImageKey: "provider_routers/ring.svg", VideoKey: "videos/ring-loop.mp4", RouterType: "intermediate", BandwidthMbps: 10000, CostRub: 310000, CreatedByID: operator.ID, PublishedAt: &now},
		{Name: "Harbor Residence Gateway", Description: "Конечный маршрутизатор жилого комплекса для распределения доступа к сети между квартирами.", Status: statusPublished, ImageKey: "provider_routers/residential.svg", VideoKey: "videos/residential-loop.mp4", RouterType: "residential", BandwidthMbps: 1000, CostRub: 72000, CreatedByID: operator.ID, PublishedAt: &now},
		{Name: "Riverside Residence Gateway", Description: "Черновик карточки маршрутизатора для следующего жилого дома.", Status: statusDraft, ImageKey: defaultImageKey, VideoKey: defaultVideoKey, RouterType: "residential", BandwidthMbps: 1000, CostRub: 65000, CreatedByID: student.ID},
		{Name: "Legacy South Edge", Description: "Выведенный из эксплуатации пограничный маршрутизатор.", Status: statusDeleted, ImageKey: "provider_routers/ring.svg", VideoKey: "videos/ring-loop.mp4", RouterType: "intermediate", BandwidthMbps: 1000, CostRub: 50000, CreatedByID: operator.ID},
	}
	if err := a.db.Create(&items).Error; err != nil {
		return err
	}
	return a.db.Create(&[]ProviderRouterLike{{ProviderRouterID: items[0].ID, UserID: student.ID}, {ProviderRouterID: items[0].ID, UserID: viewer.ID}, {ProviderRouterID: items[1].ID, UserID: viewer.ID}, {ProviderRouterID: items[2].ID, UserID: student.ID}, {ProviderRouterID: items[2].ID, UserID: operator.ID}}).Error
}

func (a *providerRoutersApp) currentCreator() (ProviderRouterUser, error) {
	var user ProviderRouterUser
	return user, a.db.Where("email = ?", fixedCreatorEmail).First(&user).Error
}

func (a *providerRoutersApp) publishedProviderRouters(minimum int) ([]ProviderRouterView, error) {
	var providerRouters []ProviderRouter
	query := a.db.Preload("Likes").Where("status = ?", statusPublished)
	if minimum > 0 {
		query = query.Where("bandwidth_mbps >= ?", minimum)
	}
	if err := query.Order("id").Find(&providerRouters).Error; err != nil {
		return nil, err
	}
	return a.toViews(providerRouters), nil
}

func (a *providerRoutersApp) feedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	items, err := a.publishedProviderRouters(0)
	if err != nil {
		serverError(w, err)
		return
	}
	if len(items) == 0 {
		http.NotFound(w, r)
		return
	}
	selected := items[0]
	if rawID := r.URL.Query().Get("id"); rawID != "" {
		id, err := strconv.ParseUint(rawID, 10, 64)
		if err != nil {
			http.Error(w, "invalid provider_router id", http.StatusBadRequest)
			return
		}
		found := -1
		for i, item := range items {
			if item.ID == uint(id) {
				found = i
				break
			}
		}
		if found == -1 {
			http.NotFound(w, r)
			return
		}
		selected = items[found]
		if r.URL.Query().Get("next") == "true" {
			selected = items[(found+1)%len(items)]
		}
	}
	a.render(w, "feed.html", map[string]any{"ProviderRouter": selected, "Title": "Лента маршрутизаторов"})
}

func (a *providerRoutersApp) draftHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	creator, err := a.currentCreator()
	if err != nil {
		serverError(w, err)
		return
	}
	var providerRouter ProviderRouter
	err = a.db.Preload("Likes").Where("created_by_id = ? AND status = ?", creator.ID, statusDraft).First(&providerRouter).Error
	page := draftPage{Error: r.URL.Query().Get("error")}
	if err == nil {
		view := a.toView(providerRouter)
		page.ProviderRouter = &view
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		serverError(w, err)
		return
	}
	a.render(w, "draft.html", page)
}

func (a *providerRoutersApp) createDraftHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	creator, err := a.currentCreator()
	if err != nil {
		serverError(w, err)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	bandwidth, cost, valid := subjectNumbersFromForm(r)
	if name == "" || !valid {
		http.Redirect(w, r, "/provider_routers/draft?error=Заполните+название,+пропускную+способность+и+стоимость", http.StatusSeeOther)
		return
	}
	providerRouter := ProviderRouter{Name: name, Description: "Новый маршрутизатор провайдера.", Status: statusDraft, ImageKey: defaultImageKey, VideoKey: defaultVideoKey, RouterType: "residential", BandwidthMbps: bandwidth, CostRub: cost, CreatedByID: creator.ID}
	if err := a.db.Create(&providerRouter).Error; err != nil {
		http.Redirect(w, r, "/provider_routers/draft?error=У+вас+уже+есть+черновик", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/provider_routers/draft", http.StatusSeeOther)
}

func (a *providerRoutersApp) publishDraftHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	creator, err := a.currentCreator()
	if err != nil {
		serverError(w, err)
		return
	}
	var providerRouter ProviderRouter
	if err := a.db.Where("created_by_id = ? AND status = ?", creator.ID, statusDraft).First(&providerRouter).Error; err != nil {
		http.Redirect(w, r, "/provider_routers/draft?error=Черновик+не+найден", http.StatusSeeOther)
		return
	}
	bandwidth, cost, valid := subjectNumbersFromForm(r)
	if !valid {
		http.Redirect(w, r, "/provider_routers/draft?error=Введите+корректные+числа", http.StatusSeeOther)
		return
	}
	now := time.Now()
	changes := map[string]any{"bandwidth_mbps": bandwidth, "cost_rub": cost, "status": statusPublished, "published_at": now}
	if err := a.db.Model(&providerRouter).Updates(changes).Error; err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/provider_routers/feed?id="+strconv.FormatUint(uint64(providerRouter.ID), 10), http.StatusSeeOther)
}

// providerRouterActionHandler intentionally uses SQL UPDATE for logical deletion in Lab 2.
func (a *providerRoutersApp) providerRouterActionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/delete") {
		methodNotAllowed(w)
		return
	}
	rawID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/provider_routers/"), "/delete")
	id, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	result := a.db.Exec("UPDATE provider_routers SET status = ? WHERE id = ? AND status = ?", statusDeleted, uint(id), statusPublished)
	if result.Error != nil {
		serverError(w, result.Error)
		return
	}
	http.Redirect(w, r, "/provider_routers", http.StatusSeeOther)
}

func (a *providerRoutersApp) gridHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	rawBandwidth := strings.TrimSpace(r.URL.Query().Get("minBandwidthMbps"))
	minimum := 0
	if rawBandwidth != "" {
		value, err := strconv.Atoi(rawBandwidth)
		if err != nil || value < 0 {
			http.Error(w, "minBandwidthMbps must be a non-negative integer", http.StatusBadRequest)
			return
		}
		minimum = value
	}
	items, err := a.publishedProviderRouters(minimum)
	if err != nil {
		serverError(w, err)
		return
	}
	a.render(w, "grid.html", gridPage{ProviderRouters: items, MinBandwidth: rawBandwidth})
}

func (a *providerRoutersApp) toViews(items []ProviderRouter) []ProviderRouterView {
	views := make([]ProviderRouterView, 0, len(items))
	for _, item := range items {
		views = append(views, a.toView(item))
	}
	return views
}

func (a *providerRoutersApp) toView(providerRouter ProviderRouter) ProviderRouterView {
	if strings.TrimSpace(providerRouter.ImageKey) == "" {
		providerRouter.ImageKey = defaultImageKey
	}
	if strings.TrimSpace(providerRouter.VideoKey) == "" {
		providerRouter.VideoKey = defaultVideoKey
	}
	return ProviderRouterView{ProviderRouter: providerRouter, ImageURL: a.mediaURL + "/" + providerRouter.ImageKey, VideoURL: a.mediaURL + "/" + providerRouter.VideoKey, Likes: len(providerRouter.Likes)}
}

func (r ProviderRouterView) BandwidthLabel() string {
	return fmt.Sprintf("%d Мбит/с", r.BandwidthMbps)
}
func (r ProviderRouterView) CostLabel() string { return fmt.Sprintf("%d ₽", r.CostRub) }
func (r ProviderRouterView) ShortDescription() string {
	letters := []rune(r.Description)
	if len(letters) <= 92 {
		return r.Description
	}
	return string(letters[:92]) + "…"
}

func subjectNumbersFromForm(r *http.Request) (int, int, bool) {
	bandwidth, bandwidthErr := strconv.Atoi(r.FormValue("bandwidthMbps"))
	cost, costErr := strconv.Atoi(r.FormValue("costRub"))
	return bandwidth, cost, bandwidthErr == nil && costErr == nil && bandwidth >= 0 && cost >= 0
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func (a *providerRoutersApp) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.templates.ExecuteTemplate(w, name, data); err != nil {
		serverError(w, err)
	}
}
func methodNotAllowed(w http.ResponseWriter) {
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}
func serverError(w http.ResponseWriter, err error) {
	log.Printf("server error: %v", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
