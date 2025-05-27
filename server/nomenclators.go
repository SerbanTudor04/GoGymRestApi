package server

import (
	"net/http"
	"strconv"
)

// Add these structs to your existing code
type Country struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	IsoCode string `json:"iso_code"`
}

type State struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	IsoCode   string `json:"iso_code"`
	CountryID int    `json:"country_id"`
}

func (app *App) getCountries(w http.ResponseWriter, r *http.Request) {
	query := "SELECT id, name, iso_code FROM countries ORDER BY name"

	rows, err := app.DB.Query(query)
	if err != nil {
		sendErrorResponse(w, "Failed to fetch countries", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var countries []Country
	for rows.Next() {
		var country Country
		err := rows.Scan(&country.ID, &country.Name, &country.IsoCode)
		if err != nil {
			sendErrorResponse(w, "Failed to scan country data", http.StatusInternalServerError)
			return
		}
		countries = append(countries, country)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		sendErrorResponse(w, "Error processing countries data", http.StatusInternalServerError)
		return
	}

	// If no countries found, return empty array instead of null
	if countries == nil {
		countries = []Country{}
	}

	sendSuccessResponse(w, "Countries retrieved successfully", countries)
}

func (app *App) getStates(w http.ResponseWriter, r *http.Request) {
	// Get country_id from query parameters
	countryIDStr := r.URL.Query().Get("country_id")
	if countryIDStr == "" {
		sendErrorResponse(w, "country_id parameter is required", http.StatusBadRequest)
		return
	}

	// Convert country_id to integer
	countryID, err := strconv.Atoi(countryIDStr)
	if err != nil {
		sendErrorResponse(w, "Invalid country_id parameter", http.StatusBadRequest)
		return
	}

	query := "SELECT id, name, iso_code, country_id FROM states WHERE country_id = $1 ORDER BY name"

	rows, err := app.DB.Query(query, countryID)
	if err != nil {
		sendErrorResponse(w, "Failed to fetch states", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var states []State
	for rows.Next() {
		var state State
		err := rows.Scan(&state.ID, &state.Name, &state.IsoCode, &state.CountryID)
		if err != nil {
			sendErrorResponse(w, "Failed to scan state data", http.StatusInternalServerError)
			return
		}
		states = append(states, state)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		sendErrorResponse(w, "Error processing states data", http.StatusInternalServerError)
		return
	}

	// If no states found, return empty array instead of null
	if states == nil {
		states = []State{}
	}

	sendSuccessResponse(w, "States retrieved successfully", states)
}
