package server

import (
	"database/sql"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/time/rate"
)

type App struct {
	DB       *sql.DB
	Config   *Config
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
}

type Response struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Rate limiter configuration
const (
	rateLimitRequests = 100         // requests per time window
	rateLimitWindow   = time.Minute // time window
	rateLimitBurst    = 10          // burst capacity
)

// Security headers
var securityHeaders = map[string]string{
	"X-Content-Type-Options":    "nosniff",
	"X-Frame-Options":           "DENY",
	"X-XSS-Protection":          "1; mode=block",
	"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
	"Content-Security-Policy":   "default-src 'self'",
	"Referrer-Policy":           "strict-origin-when-cross-origin",
}

func RunServer() {
	config := loadConfig()

	app := &App{
		Config:   config,
		limiters: make(map[string]*rate.Limiter),
	}

	err := app.initDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return
	}
	defer app.DB.Close()

	if err := app.DB.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	r := mux.NewRouter()

	// Apply middleware in order (security first, then rate limiting, then logging)
	r.Use(securityHeadersMiddleware)
	r.Use(app.rateLimitMiddleware)
	r.Use(corsMiddleware)
	r.Use(loggingMiddleware)
	r.Use(timeoutMiddleware)

	app.setupApiRouter(r)

	log.Println("Server starting on :8080 with rate limiting and security protection")
	log.Fatal(http.ListenAndServe(":8080", r))
}
