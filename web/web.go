package web

import (
	"embed"
	"net/http"

	"github.com/go-chi/chi/v5"
)

//go:embed index.html app.js
var WebFS embed.FS

// RegisterWebRoutes registra as rotas estáticas da interface gráfica web no roteador Chi
func RegisterWebRoutes(r chi.Router) {
	// Rota raiz "/" serve o Dashboard contábil em index.html
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		indexBytes, err := WebFS.ReadFile("index.html")
		if err != nil {
			http.Error(w, "Interface web não encontrada", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexBytes)
	})

	// Rota "/web/app.js"
	r.Get("/web/app.js", func(w http.ResponseWriter, r *http.Request) {
		jsBytes, err := WebFS.ReadFile("app.js")
		if err != nil {
			http.Error(w, "Arquivo app.js não encontrado", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		_, _ = w.Write(jsBytes)
	})
}

