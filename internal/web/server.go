package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/skalluru/velocix/internal/manager"
)

//go:embed static/*
var staticFiles embed.FS

func NewServer(mgr *manager.Manager, port int, logger *slog.Logger) *http.Server {
	mux := http.NewServeMux()

	staticFS, _ := fs.Sub(staticFiles, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	mux.HandleFunc("GET /", handleIndex())
	mux.HandleFunc("GET /api/runs", handleRuns(mgr))
	mux.HandleFunc("GET /api/stats", handleStats(mgr))
	mux.HandleFunc("GET /api/filters", handleFilters(mgr))
	mux.HandleFunc("GET /api/orgs", handleListOrgs(mgr))
	mux.HandleFunc("POST /api/orgs/active", handleSetActiveOrg(mgr))

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: logMiddleware(logger, mux),
	}
}

func logMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("request", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func handleIndex() http.HandlerFunc {
	data, _ := staticFiles.ReadFile("static/index.html")
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	}
}

func handleRuns(mgr *manager.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st := mgr.GetActiveStore()
		if st == nil {
			writeJSON(w, []any{})
			return
		}
		repo := r.URL.Query().Get("repo")
		status := r.URL.Query().Get("status")
		workflow := r.URL.Query().Get("workflow")
		runs := st.Filter(repo, status, workflow)
		writeJSON(w, runs)
	}
}

func handleStats(mgr *manager.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st := mgr.GetActiveStore()
		if st == nil {
			writeJSON(w, map[string]int{})
			return
		}
		stats := st.GetStats()
		writeJSON(w, stats)
	}
}

func handleFilters(mgr *manager.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st := mgr.GetActiveStore()
		if st == nil {
			writeJSON(w, map[string]any{"repos": []string{}, "workflows": []string{}})
			return
		}
		writeJSON(w, map[string]any{
			"repos":     st.GetRepoNames(),
			"workflows": st.GetWorkflowNames(),
		})
	}
}

func handleListOrgs(mgr *manager.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, mgr.ListOrgs())
	}
}

func handleSetActiveOrg(mgr *manager.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
			return
		}
		if !mgr.SetActiveOrg(body.Name) {
			http.Error(w, `{"error":"org not found"}`, http.StatusNotFound)
			return
		}
		writeJSON(w, map[string]string{"active": mgr.GetActiveOrg()})
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
	}
}
