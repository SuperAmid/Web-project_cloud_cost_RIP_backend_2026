package api

import (
	"net/http"

	swaggerFiles "github.com/swaggo/files/v2"

	"provider_routers/docs/swagger"
)

// Swagger UI: /swagger/ — страница, /swagger/doc.json — спецификация, сгенерированная swag.
const swaggerInitializer = `window.onload = function () {
  window.ui = SwaggerUIBundle({
    url: "/swagger/doc.json",
    dom_id: "#swagger-ui",
    deepLinking: true,
    persistAuthorization: true,
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
    plugins: [SwaggerUIBundle.plugins.DownloadUrl],
    layout: "StandaloneLayout"
  });
};
`

func registerSwagger(mux *http.ServeMux) {
	files := http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerFiles.FS)))
	mux.HandleFunc("GET /swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})
	mux.Handle("GET /swagger/", files)
	mux.HandleFunc("GET /swagger/doc.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(swagger.JSON)
	})
	mux.HandleFunc("GET /swagger/swagger-initializer.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		_, _ = w.Write([]byte(swaggerInitializer))
	})
}
