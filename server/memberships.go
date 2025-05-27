package server

import (
	"database/sql"
	"net/http"
)

// Add this struct to your existing code
type Membership struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
	DaysNo   int    `json:"days_no"`
}

func (app *App) getMemberships(w http.ResponseWriter, r *http.Request) {
	// Check if we should filter by active status
	activeOnly := r.URL.Query().Get("active_only")

	var query string
	var rows *sql.Rows
	var err error

	if activeOnly == "" {
		activeOnly = "true"
	}

	if activeOnly == "true" {
		// Only return active memberships
		query = "SELECT id, name, is_active, days_no FROM memberships WHERE is_active = true ORDER BY level"
		rows, err = app.DB.Query(query)
	} else {
		// Return all memberships
		query = "SELECT id, name, is_active, days_no FROM memberships ORDER BY level"
		rows, err = app.DB.Query(query)
	}

	if err != nil {
		sendErrorResponse(w, "Failed to fetch memberships", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var memberships []Membership
	for rows.Next() {
		var membership Membership
		err := rows.Scan(&membership.ID, &membership.Name, &membership.IsActive, &membership.DaysNo)
		if err != nil {
			sendErrorResponse(w, "Failed to scan membership data", http.StatusInternalServerError)
			return
		}
		memberships = append(memberships, membership)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		sendErrorResponse(w, "Error processing memberships data", http.StatusInternalServerError)
		return
	}

	// If no memberships found, return empty array instead of null
	if memberships == nil {
		memberships = []Membership{}
	}

	sendSuccessResponse(w, "Memberships retrieved successfully", memberships)
}
