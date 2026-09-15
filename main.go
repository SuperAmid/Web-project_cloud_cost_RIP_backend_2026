package main

import (
	"errors"
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

const fixedCreatorEmail = "student@provider-network.local"
const (
	statusDraft     = "draft"
	statusPublished = "published"
	statusDeleted   = "deleted"
	defaultImageURL = "/media/routers/draft.svg"
	defaultVideoURL = "/media/videos/draft-loop.mp4"
)

type ProviderUser struct {
	ID          uint   `gorm:"primaryKey"`
	Email       string `gorm:"size:120;uniqueIndex;not null"`
	DisplayName string `gorm:"size:80;not null"`
	CreatedAt   time.Time
}
type Router struct {
	ID                uint   `gorm:"primaryKey"`
	Name              string `gorm:"size:120;not null"`
	Description       string `gorm:"size:500;not null"`
	Status            string `gorm:"size:12;index;not null"`
	ImageURL          string `gorm:"size:500;not null"`
	VideoURL          string `gorm:"size:500;not null"`
	RouterType        string `gorm:"size:24;not null"`
	ThroughputMbps    int    `gorm:"not null"`
	PowerConsumptionW int    `gorm:"not null"`
	PortCount         int    `gorm:"not null"`
	Location          string `gorm:"size:160;not null"`
	MasterRouterName  string `gorm:"size:120;not null"`
	CreatedAt         time.Time
	PublishedAt       *time.Time
	CreatedByID       uint         `gorm:"not null;index"`
	CreatedBy         ProviderUser `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`
	Likes             []RouterLike `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`
}
type RouterLike struct {
	RouterID  uint `gorm:"primaryKey"`
	UserID    uint `gorm:"primaryKey"`
	CreatedAt time.Time
	Router    Router       `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`
	User      ProviderUser `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`
}
type RouterView struct {
	Router
	Likes int
}
type draftPage struct {
	Router *RouterView
	Error  string
}
type gridPage struct {
	Routers       []RouterView
	MinThroughput string
}
type App struct {
	db        *gorm.DB
	templates *template.Template
}

func main() {
	db, err := gorm.Open(postgres.Open(envOr("DATABASE_URL", "host=localhost user=router_user password=router_password dbname=provider_network port=5432 sslmode=disable")), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	if err := migrateAndSeed(db); err != nil {
		log.Fatal(err)
	}
	app := &App{db: db, templates: template.Must(template.ParseGlob("templates/*.html"))}
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("/media/", http.StripPrefix("/media/", http.FileServer(http.Dir("assets/provider-media"))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/routers/feed", http.StatusFound) })
	mux.HandleFunc("/routers/feed", app.feedHandler)
	mux.HandleFunc("/routers/draft", app.draftHandler)
	mux.HandleFunc("/routers/draft/create", app.createDraftHandler)
	mux.HandleFunc("/routers/draft/publish", app.publishDraftHandler)
	mux.HandleFunc("/routers/", app.routerActionHandler)
	mux.HandleFunc("/routers", app.gridHandler)
	log.Printf("provider-router lab2 listens on http://localhost%s", envOr("APP_ADDR", ":8080"))
	log.Fatal(http.ListenAndServe(envOr("APP_ADDR", ":8080"), mux))
}

func migrateAndSeed(db *gorm.DB) error {
	if err := db.AutoMigrate(&ProviderUser{}, &Router{}, &RouterLike{}); err != nil {
		return err
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS one_draft_router_per_creator ON routers (created_by_id) WHERE status = 'draft'").Error; err != nil {
		return err
	}
	var count int64
	if err := db.Model(&Router{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	student, operator, viewer := ProviderUser{Email: fixedCreatorEmail, DisplayName: "Студент РИП"}, ProviderUser{Email: "operator@provider-network.local", DisplayName: "Оператор сети"}, ProviderUser{Email: "viewer@provider-network.local", DisplayName: "Наблюдатель"}
	if err := db.Create(&[]*ProviderUser{&student, &operator, &viewer}).Error; err != nil {
		return err
	}
	now := time.Now()
	media := func(key string) string {
		return strings.TrimRight(envOr("MINIO_PUBLIC_URL", "http://localhost:9000/provider-media"), "/") + "/" + key
	}
	items := []Router{
		{Name: "KZ Backbone One", Description: "Центральный маршрутизатор ядра провайдера для магистрального узла.", Status: statusPublished, ImageURL: media("routers/core.svg"), VideoURL: media("videos/core-loop.mp4"), RouterType: "central", ThroughputMbps: 100000, PowerConsumptionW: 1450, PortCount: 8, Location: "ЦОД Астана, стойка A-12", MasterRouterName: "—", CreatedByID: operator.ID, PublishedAt: &now},
		{Name: "North Ring Hub", Description: "Промежуточный маршрутизатор северного кольца с агрегацией районных узлов.", Status: statusPublished, ImageURL: media("routers/ring.svg"), VideoURL: media("videos/ring-loop.mp4"), RouterType: "intermediate", ThroughputMbps: 10000, PowerConsumptionW: 310, PortCount: 24, Location: "Астана, Северный узел", MasterRouterName: "KZ Backbone One", CreatedByID: operator.ID, PublishedAt: &now},
		{Name: "Turan Residence Gateway", Description: "Конечный маршрутизатор жилого комплекса: распределение трафика квартир.", Status: statusPublished, ImageURL: media("routers/residential.svg"), VideoURL: media("videos/residential-loop.mp4"), RouterType: "residential", ThroughputMbps: 1000, PowerConsumptionW: 72, PortCount: 16, Location: "ЖК Туран, корпус 3", MasterRouterName: "North Ring Hub", CreatedByID: operator.ID, PublishedAt: &now},
		{Name: "Saryarka Residence Gateway", Description: "Черновик карточки маршрутизатора для следующего жилого дома.", Status: statusDraft, ImageURL: media("routers/draft.svg"), VideoURL: media("videos/draft-loop.mp4"), RouterType: "residential", ThroughputMbps: 1000, PowerConsumptionW: 48, PortCount: 12, Location: "ЖК Сарыарка, корпус 1", MasterRouterName: "North Ring Hub", CreatedByID: operator.ID},
		{Name: "Legacy South Edge", Description: "Выведенный из эксплуатации пограничный маршрутизатор.", Status: statusDeleted, ImageURL: media("routers/ring.svg"), VideoURL: media("videos/ring-loop.mp4"), RouterType: "intermediate", ThroughputMbps: 1000, PowerConsumptionW: 210, PortCount: 8, Location: "Астана, Южный узел", MasterRouterName: "KZ Backbone One", CreatedByID: operator.ID}}
	if err := db.Create(&items).Error; err != nil {
		return err
	}
	return db.Create(&[]RouterLike{{RouterID: items[0].ID, UserID: student.ID}, {RouterID: items[0].ID, UserID: viewer.ID}, {RouterID: items[1].ID, UserID: viewer.ID}, {RouterID: items[2].ID, UserID: student.ID}, {RouterID: items[2].ID, UserID: operator.ID}}).Error
}
func (a *App) currentCreator() (ProviderUser, error) {
	var user ProviderUser
	return user, a.db.Where("email = ?", fixedCreatorEmail).First(&user).Error
}
func (a *App) publishedRouters(minimum int) ([]RouterView, error) {
	var routers []Router
	query := a.db.Preload("Likes").Where("status = ?", statusPublished)
	if minimum > 0 {
		query = query.Where("throughput_mbps >= ?", minimum)
	}
	if err := query.Order("id").Find(&routers).Error; err != nil {
		return nil, err
	}
	return toViews(routers), nil
}
func (a *App) feedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	items, err := a.publishedRouters(0)
	if err != nil {
		serverError(w, err)
		return
	}
	if len(items) == 0 {
		http.NotFound(w, r)
		return
	}
	selected := items[0]
	if raw := r.URL.Query().Get("id"); raw != "" {
		id, e := strconv.ParseUint(raw, 10, 64)
		if e != nil {
			http.Error(w, "invalid router id", 400)
			return
		}
		found := -1
		for i, item := range items {
			if item.ID == uint(id) {
				found = i
				break
			}
		}
		if found < 0 {
			http.NotFound(w, r)
			return
		}
		selected = items[found]
		if r.URL.Query().Get("next") == "true" {
			selected = items[(found+1)%len(items)]
		}
	}
	a.render(w, "feed.html", map[string]any{"Router": selected, "Title": "Лента маршрутизаторов"})
}
func (a *App) draftHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	creator, err := a.currentCreator()
	if err != nil {
		serverError(w, err)
		return
	}
	var router Router
	err = a.db.Preload("Likes").Where("created_by_id = ? AND status = ?", creator.ID, statusDraft).First(&router).Error
	page := draftPage{Error: r.URL.Query().Get("error")}
	if err == nil {
		view := toView(router)
		page.Router = &view
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		serverError(w, err)
		return
	}
	a.render(w, "draft.html", page)
}
func (a *App) createDraftHandler(w http.ResponseWriter, r *http.Request) {
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
	if name == "" {
		http.Redirect(w, r, "/routers/draft?error=Введите+название", 303)
		return
	}
	router := Router{Name: name, Description: "", Status: statusDraft, ImageURL: defaultImageURL, VideoURL: defaultVideoURL, RouterType: "residential", Location: "", MasterRouterName: "", CreatedByID: creator.ID}
	if err := a.db.Create(&router).Error; err != nil {
		http.Redirect(w, r, "/routers/draft?error=У+вас+уже+есть+черновик", 303)
		return
	}
	http.Redirect(w, r, "/routers/draft", 303)
}
func (a *App) publishDraftHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	creator, err := a.currentCreator()
	if err != nil {
		serverError(w, err)
		return
	}
	var router Router
	if err := a.db.Where("created_by_id = ? AND status = ?", creator.ID, statusDraft).First(&router).Error; err != nil {
		http.Redirect(w, r, "/routers/draft?error=Черновик+не+найден", 303)
		return
	}
	throughput, power, ports, valid := numbersFromForm(r)
	if !valid || strings.TrimSpace(r.FormValue("description")) == "" {
		http.Redirect(w, r, "/routers/draft?error=Заполните+описание+и+числовые+поля", 303)
		return
	}
	now := time.Now()
	changes := map[string]any{"description": strings.TrimSpace(r.FormValue("description")), "router_type": r.FormValue("routerType"), "throughput_mbps": throughput, "power_consumption_w": power, "port_count": ports, "location": strings.TrimSpace(r.FormValue("location")), "master_router_name": strings.TrimSpace(r.FormValue("masterRouterName")), "status": statusPublished, "published_at": now}
	if err := a.db.Model(&router).Updates(changes).Error; err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/routers/feed?id="+strconv.FormatUint(uint64(router.ID), 10), 303)
}
func (a *App) routerActionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/delete") {
		methodNotAllowed(w)
		return
	}
	raw := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/routers/"), "/delete")
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	result := a.db.Exec("UPDATE routers SET status = ? WHERE id = ? AND status = ?", statusDeleted, uint(id), statusPublished)
	if result.Error != nil {
		serverError(w, result.Error)
		return
	}
	http.Redirect(w, r, "/routers", 303)
}
func (a *App) gridHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	raw := strings.TrimSpace(r.URL.Query().Get("minThroughputMbps"))
	minimum := 0
	if raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			http.Error(w, "minThroughputMbps must be a non-negative integer", 400)
			return
		}
		minimum = value
	}
	items, err := a.publishedRouters(minimum)
	if err != nil {
		serverError(w, err)
		return
	}
	a.render(w, "grid.html", gridPage{Routers: items, MinThroughput: raw})
}
func toViews(items []Router) []RouterView {
	views := make([]RouterView, 0, len(items))
	for _, router := range items {
		views = append(views, toView(router))
	}
	return views
}
func toView(router Router) RouterView {
	if strings.TrimSpace(router.ImageURL) == "" {
		router.ImageURL = defaultImageURL
	}
	if strings.TrimSpace(router.VideoURL) == "" {
		router.VideoURL = defaultVideoURL
	}
	return RouterView{Router: router, Likes: len(router.Likes)}
}
func numbersFromForm(r *http.Request) (int, int, int, bool) {
	a, e1 := strconv.Atoi(r.FormValue("throughputMbps"))
	b, e2 := strconv.Atoi(r.FormValue("powerConsumptionW"))
	c, e3 := strconv.Atoi(r.FormValue("portCount"))
	return a, b, c, e1 == nil && e2 == nil && e3 == nil && a >= 0 && b >= 0 && c >= 0
}
func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func (a *App) render(w http.ResponseWriter, name string, data any) {
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
	http.Error(w, "internal server error", 500)
}
