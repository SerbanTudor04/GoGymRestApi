package server

import (
	"database/sql"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

type App struct {
	DB     *sql.DB
	Config *Config
}

type Response struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func RunServer() {
	config := loadConfig()

	app := &App{
		Config: config,
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

	// Enable CORS for local development
	r.Use(corsMiddleware)
	r.Use(loggingMiddleware)
	app.setupApiRouter(r)
	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
