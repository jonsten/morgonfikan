package httpapi

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/jonsten/morgonfikan/internal/menu"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	config *menu.Config
	zone   *time.Location
	now    func() time.Time
}

func New(config *menu.Config, zone *time.Location) *Server {
	return &Server{config: config, zone: zone, now: time.Now}
}

func (s *Server) Handler() http.Handler {
	static, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(fmt.Sprintf("load embedded frontend: %v", err))
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/api/v1/today", s.today)
	mux.HandleFunc("/api/v1/week", s.week)
	mux.Handle("/", http.FileServer(http.FS(static)))
	return s.securityHeaders(mux)
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet, http.MethodHead) {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) today(w http.ResponseWriter, r *http.Request) {
	if !s.allowAPIRequest(w, r) {
		return
	}
	now := s.now().In(s.zone)
	day, err := s.config.Day(now)
	if err != nil {
		log.Printf("resolve today's menu: %v", err)
		http.Error(w, "menu schedule unavailable", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, r, day)
}

func (s *Server) week(w http.ResponseWriter, r *http.Request) {
	if !s.allowAPIRequest(w, r) {
		return
	}
	start := menu.WeekStart(s.now().In(s.zone).AddDate(0, 0, 1))
	dates := menu.WeekdaysFrom(start)
	days := make([]menu.Day, 0, len(dates))
	for _, date := range dates {
		day, err := s.config.Day(date)
		if err != nil {
			log.Printf("resolve menu for %s: %v", date.Format("2006-01-02"), err)
			http.Error(w, "menu schedule unavailable", http.StatusServiceUnavailable)
			return
		}
		days = append(days, day)
	}
	writeJSON(w, r, struct {
		Days []menu.Day `json:"days"`
	}{Days: days})
}

func (s *Server) allowAPIRequest(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
	w.Header().Set("Access-Control-Max-Age", "600")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return false
	}
	return allowMethod(w, r, http.MethodGet, http.MethodHead)
}

func allowMethod(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	for _, method := range methods {
		if r.Method == method {
			return true
		}
	}
	w.Header().Set("Allow", methodsHeader(methods))
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	return false
}

func methodsHeader(methods []string) string {
	result := ""
	for i, method := range methods {
		if i > 0 {
			result += ", "
		}
		result += method
	}
	return result
}

func writeJSON(w http.ResponseWriter, r *http.Request, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode API response: %v", err)
	}
}
