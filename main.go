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

type ProviderRouterStatus string

const (
	statusDraft     ProviderRouterStatus = "draft"
	statusPublished ProviderRouterStatus = "published"
	statusDeleted   ProviderRouterStatus = "deleted"
)

// ProviderRouter is the atomic in-memory model required in Lab 1.
type ProviderRouter struct {
	ID            int
	Name          string
	Description   string
	Status        ProviderRouterStatus
	ImageKey      string
	VideoKey      string
	RouterType    string
	BandwidthMbps int
	CostRub       int
	LikedUserIDs  []int
}

type ProviderRouterView struct {
	ProviderRouter
	ImageURL string
	VideoURL string
	Likes    int
}

type providerRoutersApp struct {
	templates       *template.Template
	mediaURL        string
	providerRouters []ProviderRouter
}

var providerRouters = []ProviderRouter{
	{ID: 101, Name: "Core Backbone One", Description: "Центральный маршрутизатор ядра провайдера для магистрального узла и распределения трафика между городскими сегментами сети.", Status: statusPublished, ImageKey: "provider_routers/core.svg", VideoKey: "videos/core-loop.mp4", RouterType: "central", BandwidthMbps: 100000, CostRub: 980000, LikedUserIDs: []int{2, 7, 13, 21}},
	{ID: 102, Name: "North Ring Hub", Description: "Промежуточный маршрутизатор кольцевой сети, который агрегирует районные узлы и передаёт трафик на магистраль.", Status: statusPublished, ImageKey: "provider_routers/ring.svg", VideoKey: "videos/ring-loop.mp4", RouterType: "intermediate", BandwidthMbps: 10000, CostRub: 310000, LikedUserIDs: []int{4, 8, 15}},
	{ID: 103, Name: "Harbor Residence Gateway", Description: "Конечный маршрутизатор жилого комплекса для распределения доступа к сети между квартирами.", Status: statusPublished, ImageKey: "provider_routers/residential.svg", VideoKey: "videos/residential-loop.mp4", RouterType: "residential", BandwidthMbps: 1000, CostRub: 72000, LikedUserIDs: []int{1, 3, 6, 9, 12, 18}},
	{ID: 104, Name: "Riverside Residence Gateway", Description: "Черновик карточки маршрутизатора для следующего жилого дома.", Status: statusDraft, ImageKey: "provider_routers/draft.svg", VideoKey: "videos/draft-loop.mp4", RouterType: "residential", BandwidthMbps: 1000, CostRub: 65000, LikedUserIDs: []int{5}},
	{ID: 105, Name: "Legacy South Edge", Description: "Выведенный из эксплуатации пограничный маршрутизатор.", Status: statusDeleted, ImageKey: "provider_routers/ring.svg", VideoKey: "videos/ring-loop.mp4", RouterType: "intermediate", BandwidthMbps: 1000, CostRub: 50000, LikedUserIDs: []int{11}},
}

func main() {
	app := &providerRoutersApp{
		templates:       template.Must(template.ParseGlob("templates/*.html")),
		mediaURL:        strings.TrimRight(envOr("MINIO_PUBLIC_URL", "http://localhost:9000/provider-media"), "/"),
		providerRouters: providerRouters,
	}
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/provider_routers/feed", http.StatusFound)
	})
	mux.HandleFunc("/provider_routers/feed", app.feedHandler)
	mux.HandleFunc("/provider_routers/draft", app.draftHandler)
	mux.HandleFunc("/provider_routers", app.gridHandler)
	addr := envOr("APP_ADDR", ":8080")
	log.Printf("provider_routers lab1 listens on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func (a *providerRoutersApp) publishedProviderRouters() []ProviderRouter {
	items := make([]ProviderRouter, 0)
	for _, providerRouter := range a.providerRouters {
		if providerRouter.Status == statusPublished {
			items = append(items, providerRouter)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (a *providerRoutersApp) view(providerRouter ProviderRouter) ProviderRouterView {
	return ProviderRouterView{ProviderRouter: providerRouter, ImageURL: a.mediaURL + "/" + providerRouter.ImageKey, VideoURL: a.mediaURL + "/" + providerRouter.VideoKey, Likes: len(providerRouter.LikedUserIDs)}
}

func (a *providerRoutersApp) feedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	items := a.publishedProviderRouters()
	if len(items) == 0 {
		http.NotFound(w, r)
		return
	}
	selected := items[0]
	if rawID := r.URL.Query().Get("id"); rawID != "" {
		id, err := strconv.Atoi(rawID)
		if err != nil {
			http.Error(w, "invalid provider_router id", http.StatusBadRequest)
			return
		}
		found := -1
		for i, item := range items {
			if item.ID == id {
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
	a.render(w, "feed.html", map[string]any{"ProviderRouter": a.view(selected), "Title": "Лента маршрутизаторов"})
}

func (a *providerRoutersApp) draftHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	for _, providerRouter := range a.providerRouters {
		if providerRouter.Status == statusDraft {
			a.render(w, "draft.html", map[string]any{"ProviderRouter": a.view(providerRouter), "Title": "Добавление маршрутизатора"})
			return
		}
	}
	http.NotFound(w, r)
}

func (a *providerRoutersApp) gridHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rawBandwidth := strings.TrimSpace(r.URL.Query().Get("minBandwidthMbps"))
	minimum := 0
	if rawBandwidth != "" {
		var err error
		minimum, err = strconv.Atoi(rawBandwidth)
		if err != nil || minimum < 0 {
			http.Error(w, "minBandwidthMbps must be a non-negative integer", http.StatusBadRequest)
			return
		}
	}
	views := make([]ProviderRouterView, 0)
	for _, providerRouter := range a.publishedProviderRouters() {
		if providerRouter.BandwidthMbps >= minimum {
			views = append(views, a.view(providerRouter))
		}
	}
	a.render(w, "grid.html", map[string]any{"ProviderRouters": views, "MinBandwidth": rawBandwidth, "Title": "Маршрутизаторы"})
}

func (a *providerRoutersApp) render(w http.ResponseWriter, name string, data any) {
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
