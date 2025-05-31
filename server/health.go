// Add this to server/health.go
package server

import (
	"net/http"
	"time"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Database  string    `json:"database"`
	Version   string    `json:"version"`
}

func (app *App) healthCheck(w http.ResponseWriter, r *http.Request) {
	health := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0", // You can make this dynamic
	}

	// Check database connection
	if err := app.DB.Ping(); err != nil {
		health.Status = "unhealthy"
		health.Database = "disconnected"
		sendErrorResponse(w, "Database connection failed", http.StatusServiceUnavailable)
		return
	}
	health.Database = "connected"

	sendSuccessResponse(w, "Service is healthy", health)
}

// Add this route to your router.go setupApiRouter function:
//
