package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

type RouterStatus string

const (
	StatusDraft     RouterStatus = "draft"
	StatusPublished RouterStatus = "published"
	StatusDeleted   RouterStatus = "deleted"
)

// Router — модель маршрутизатора провайдера.
type Router struct {
	ID                int
	Name              string
	ModelName         string
	Description       string
	Status            RouterStatus
	ImageKey          string
	VideoKey          string
	RouterType        string
	ThroughputMbps    int
	PowerConsumptionW int
	PortCount         int
	Location          string
	MasterRouterName  string
	LikedUserIDs      []int
}

type RouterView struct {
	Router
	ImageURL string
	VideoURL string
	Likes    int
}

type App struct {
	templates *template.Template
	mediaURL  string
	routers   []Router
}

var routerCollection = []Router{
	{
		ID:                101,
		Name:              "Core Backbone One",
		ModelName:         "Juniper PTX10008",
		Description:       "Центральный маршрутизатор ядра провайдера для магистрального узла.",
		Status:            StatusPublished,
		ImageKey:          "routers/core.svg",
		VideoKey:          "videos/core-loop.mp4",
		RouterType:        "central",
		ThroughputMbps:    100000,
		PowerConsumptionW: 1450,
		PortCount:         8,
		Location:          "Центральный ЦОД, стойка A-12",
		MasterRouterName:  "—",
		LikedUserIDs:      []int{2, 7, 13, 21},
	},
	{
		ID:                102,
		Name:              "North Ring Hub",
		ModelName:         "Cisco NCS 540",
		Description:       "Промежуточный маршрутизатор кольцевой сети с агрегацией районных узлов.",
		Status:            StatusPublished,
		ImageKey:          "routers/ring.svg",
		VideoKey:          "videos/ring-loop.mp4",
		RouterType:        "intermediate",
		ThroughputMbps:    10000,
		PowerConsumptionW: 310,
		PortCount:         24,
		Location:          "Северный узел",
		MasterRouterName:  "Core Backbone One",
		LikedUserIDs:      []int{4, 8, 15},
	},
	{ID: 103, Name: "Harbor Residence Gateway", ModelName: "MikroTik CCR2116", Description: "Конечный маршрутизатор жилого комплекса: распределение трафика квартир.", Status: StatusPublished, ImageKey: "routers/residential.svg", VideoKey: "videos/residential-loop.mp4", RouterType: "residential", ThroughputMbps: 1000, PowerConsumptionW: 72, PortCount: 16, Location: "Жилой комплекс «Панорама», корпус 3", MasterRouterName: "North Ring Hub", LikedUserIDs: []int{1, 3, 6, 9, 12, 18}},
	{ID: 104, Name: "Riverside Residence Gateway", ModelName: "MikroTik CCR2004", Description: "Черновик карточки маршрутизатора для следующего жилого дома.", Status: StatusDraft, ImageKey: "routers/draft.svg", VideoKey: "videos/draft-loop.mp4", RouterType: "residential", ThroughputMbps: 1000, PowerConsumptionW: 48, PortCount: 12, Location: "Жилой комплекс «Речной», корпус 1", MasterRouterName: "North Ring Hub", LikedUserIDs: []int{5}},
	{ID: 105, Name: "Legacy South Edge", ModelName: "Cisco ASR 920", Description: "Выведенный из эксплуатации пограничный маршрутизатор.", Status: StatusDeleted, ImageKey: "routers/legacy.svg", VideoKey: "videos/legacy-loop.mp4", RouterType: "intermediate", ThroughputMbps: 1000, PowerConsumptionW: 210, PortCount: 8, Location: "Южный узел", MasterRouterName: "Core Backbone One", LikedUserIDs: []int{11}},
}

func main() {
	mediaURL := strings.TrimRight(envOr("MINIO_PUBLIC_URL", "http://localhost:9000/provider-media"), "/")
	app := &App{templates: template.Must(template.ParseGlob("templates/*.html")), mediaURL: mediaURL, routers: routerCollection}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/provider-routers/feed", http.StatusFound)
	})
	mux.HandleFunc("/provider-routers/feed", app.feedHandler)
	mux.HandleFunc("/provider-routers/draft", app.draftHandler)
	mux.HandleFunc("/provider-routers", app.gridHandler)

	addr := envOr("APP_ADDR", ":8080")
	log.Printf("provider-routers listens on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func (a *App) published() []Router {
	items := make([]Router, 0)
	for _, router := range a.routers {
		if router.Status == StatusPublished {
			items = append(items, router)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (a *App) view(router Router) RouterView {
	return RouterView{Router: router, ImageURL: a.mediaURL + "/" + router.ImageKey,
		VideoURL: a.mediaURL + "/" + router.VideoKey, Likes: len(router.LikedUserIDs)}
}

func (a *App) feedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	items := a.published()
	if len(items) == 0 {
		http.Error(w, "no published routers", http.StatusNotFound)
		return
	}
	selected := items[0]
	if rawID := r.URL.Query().Get("id"); rawID != "" {
		id, err := strconv.Atoi(rawID)
		if err != nil {
			http.Error(w, "invalid router id", http.StatusBadRequest)
			return
		}
		found := -1
		for i, router := range items {
			if router.ID == id {
				found = i
				break
			}
		}
		if found == -1 {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("next") == "true" {
			selected = items[(found+1)%len(items)]
		} else {
			selected = items[found]
		}
	}
	a.render(w, "feed.html", map[string]any{"Router": a.view(selected), "Title": "Лента маршрутизаторов"})
}

func (a *App) draftHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	for _, router := range a.routers {
		if router.Status == StatusDraft {
			a.render(w, "draft.html", map[string]any{"Router": a.view(router), "Title": "Добавление маршрутизатора"})
			return
		}
	}
	http.NotFound(w, r)
}

func (a *App) gridHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rawLimit := strings.TrimSpace(r.URL.Query().Get("minThroughputMbps"))
	minimum := 0
	if rawLimit != "" {
		var err error
		minimum, err = strconv.Atoi(rawLimit)
		if err != nil || minimum < 0 {
			http.Error(w, "minThroughputMbps must be a non-negative integer", http.StatusBadRequest)
			return
		}
	}
	views := make([]RouterView, 0)
	maxThroughput := 0
	for _, router := range a.published() {
		if router.ThroughputMbps >= minimum {
			views = append(views, a.view(router))
		}
		if router.ThroughputMbps > maxThroughput {
			maxThroughput = router.ThroughputMbps
		}
	}
	if maxThroughput == 0 {
		maxThroughput = 100000
	}
	a.render(w, "grid.html", map[string]any{"Routers": views, "MinThroughput": minimum, "MaxThroughput": maxThroughput, "Title": "Маршрутизаторы"})
}

func (a *App) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template %s: %v", name, err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func (r RouterView) ThroughputLabel() string {
	return fmt.Sprintf("%s Мбит/с", formatNumber(r.ThroughputMbps))
}
func formatNumber(value int) string { return strconv.FormatInt(int64(value), 10) }
