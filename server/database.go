package server

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
	DBSSLMode    string
	ServerPort   string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  time.Duration
}

func loadConfig() *Config {
	// Load .env file - godotenv.Load() will not overwrite existing OS env vars
	godotenv.Load()

	maxOpenConns, _ := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", "25"))
	maxIdleConns, _ := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", "10"))
	maxLifetimeStr := getEnv("DB_MAX_LIFETIME", "300s")
	maxLifetime, _ := time.ParseDuration(maxLifetimeStr)

	return &Config{
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "gogymrest"),
		DBPassword:   getEnv("DB_PASSWORD", "gogymrest"),
		DBName:       getEnv("DB_NAME", "gogym"),
		DBSSLMode:    getEnv("DB_SSL_MODE", "disable"),
		ServerPort:   getEnv("SERVER_PORT", "8080"),
		MaxOpenConns: maxOpenConns,
		MaxIdleConns: maxIdleConns,
		MaxLifetime:  maxLifetime,
	}
}

// getEnv gets environment variable with fallback to default value
// OS environment variables take precedence over .env file values
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (app *App) initDB() error {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		app.Config.DBHost, app.Config.DBPort, app.Config.DBUser,
		app.Config.DBPassword, app.Config.DBName, app.Config.DBSSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	// Configure connection pool
	db.SetMaxOpenConns(app.Config.MaxOpenConns)
	db.SetMaxIdleConns(app.Config.MaxIdleConns)
	db.SetConnMaxLifetime(app.Config.MaxLifetime)

	app.DB = db
	return nil
}
