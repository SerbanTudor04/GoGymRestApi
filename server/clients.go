package server

import (
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Client struct matching your database schema
type Client struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	CIF             string `json:"cif"`
	DOB             string `json:"dob"`
	TradeRegisterNo string `json:"trade_register_no"`
	CountryID       int    `json:"country_id"`
	CountryName     string `json:"country_name"`
	StateID         int    `json:"state_id"`
	StateName       string `json:"state_name"`
	City            string `json:"city"`
	StreetName      string `json:"street_name"`
	StreetNo        string `json:"street_no"`
	Building        string `json:"building"`
	Floor           string `json:"floor"`
	Apartment       string `json:"apartment"`
	CreatedOn       string `json:"created_on"`
	UpdatedOn       string `json:"updated_on"`
	CreatedBy       int    `json:"created_by"`
	UpdatedBy       int    `json:"updated_by"`
}

func (app *App) getClients(w http.ResponseWriter, r *http.Request) {
	// JWT Authentication (same as gym approach)
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || len(authHeader) < 8 {
		sendErrorResponse(w, "Authorization header missing or invalid", http.StatusUnauthorized)
		return
	}

	tokenString := authHeader[7:] // Remove "Bearer "

	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(JWT_SECRET), nil
	})

	if err != nil {
		sendErrorResponse(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Query to get all clients
	clientQuery := `SELECT c.id, c.name, c.cif, 
                          TO_CHAR(c.dob, 'YYYY-MM-DD') as dob,
                          c.trade_register_no, c.country_id, c.state_id, 
                          c.city, c.street_name, c.street_no, c.building, 
                          c.floor, c.apartment,
                          TO_CHAR(c.created_on, 'YYYY-MM-DD') as created_on,
                          TO_CHAR(c.updated_on, 'YYYY-MM-DD') as updated_on,
                          c.created_by, c.updated_by
                   FROM clients c
                   inner join user_clients uc on uc.client_id = c.id
                   where uc.user_id = $1
                   ORDER BY c.created_on DESC`

	rows, err := app.DB.Query(clientQuery, claims.UserID)
	if err != nil {
		sendErrorResponse(w, "Failed to fetch clients: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var clients []Client
	for rows.Next() {
		var client Client
		err := rows.Scan(
			&client.ID, &client.Name, &client.CIF, &client.DOB,
			&client.TradeRegisterNo, &client.CountryID, &client.StateID,
			&client.City, &client.StreetName, &client.StreetNo, &client.Building,
			&client.Floor, &client.Apartment, &client.CreatedOn, &client.UpdatedOn,
			&client.CreatedBy, &client.UpdatedBy)

		if err != nil {
			sendErrorResponse(w, "Failed to scan client: "+err.Error(), http.StatusInternalServerError)
			return
		}

		clients = append(clients, client)
	}

	// Check for any row iteration errors
	if err = rows.Err(); err != nil {
		sendErrorResponse(w, "Error during row iteration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	sendSuccessResponse(w, "Clients fetched successfully", clients)
}

type AddUserToClientRequest struct {
	UserID   int `json:"user_id"`
	ClientID int `json:"client_id"`
}

type UserClient struct {
	ID        int    `json:"id"`
	UserID    int    `json:"user_id"`
	ClientID  int    `json:"client_id"`
	CreatedOn string `json:"created_on"`
}

func (app *App) addUserToClient(w http.ResponseWriter, r *http.Request) {
	var req AddUserToClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.UserID <= 0 {
		sendErrorResponse(w, "Valid user_id is required", http.StatusBadRequest)
		return
	}
	if req.ClientID <= 0 {
		sendErrorResponse(w, "Valid client_id is required", http.StatusBadRequest)
		return
	}

	// Start a transaction
	tx, err := app.DB.Begin()
	if err != nil {
		sendErrorResponse(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}

	// Ensure we rollback if something goes wrong
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Call the PostgreSQL function
	var result string
	query := "SELECT add_user_to_client($1, $2)"
	err = tx.QueryRow(query, req.ClientID, req.UserID).Scan(&result)
	if err != nil {
		sendErrorResponse(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if operation was successful
	if result != "OK" {
		tx.Rollback()
		sendErrorResponse(w, result, http.StatusBadRequest)
		return
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		sendErrorResponse(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// Get the created user-client relationship details
	var userClient UserClient
	userClientQuery := `SELECT id, user_id, client_id, TO_CHAR(created_on, 'YYYY-MM-DD HH24:MI:SS') as created_on 
                        FROM user_clients 
                        WHERE user_id = $1 AND client_id = $2 
                        ORDER BY id DESC 
                        LIMIT 1`

	err = app.DB.QueryRow(userClientQuery, req.UserID, req.ClientID).Scan(
		&userClient.ID, &userClient.UserID, &userClient.ClientID, &userClient.CreatedOn)

	if err != nil {
		// User was added to client but couldn't fetch details
		sendSuccessResponse(w, "User added to client successfully", map[string]interface{}{
			"status":    "OK",
			"user_id":   req.UserID,
			"client_id": req.ClientID,
		})
		return
	}

	sendSuccessResponse(w, "User added to client successfully", userClient)
}

// Alternative implementation using path parameters instead of JSON body
func (app *App) addUserToClientByPath(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	userIDStr := vars["user_id"]
	clientIDStr := vars["client_id"]

	userID, err := strconv.Atoi(userIDStr)
	if err != nil || userID <= 0 {
		sendErrorResponse(w, "Invalid user_id parameter", http.StatusBadRequest)
		return
	}

	clientID, err := strconv.Atoi(clientIDStr)
	if err != nil || clientID <= 0 {
		sendErrorResponse(w, "Invalid client_id parameter", http.StatusBadRequest)
		return
	}

	// Start a transaction
	tx, err := app.DB.Begin()
	if err != nil {
		sendErrorResponse(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}

	// Ensure we rollback if something goes wrong
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Call the PostgreSQL function
	var result string
	query := "SELECT add_user_to_client($1, $2)"
	err = tx.QueryRow(query, clientID, userID).Scan(&result)
	if err != nil {
		sendErrorResponse(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if operation was successful
	if result != "OK" {
		tx.Rollback()
		sendErrorResponse(w, result, http.StatusBadRequest)
		return
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		sendErrorResponse(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	sendSuccessResponse(w, "User added to client successfully", map[string]interface{}{
		"status":    "OK",
		"user_id":   userID,
		"client_id": clientID,
	})
}

type CreateClientRequest struct {
	Name            string `json:"name"`
	CIF             string `json:"cif"`
	DOB             string `json:"dob"` // Format: "2006-01-02"
	TradeRegisterNo string `json:"trade_register_no"`
	CountryID       int    `json:"country_id"`
	StateID         int    `json:"state_id"`
	City            string `json:"city"`
	StreetName      string `json:"street_name"`
	StreetNo        string `json:"street_no"`
	Building        string `json:"building,omitempty"`
	Floor           string `json:"floor,omitempty"`
	Apartment       string `json:"apartment,omitempty"`
}

func (app *App) createClient(w http.ResponseWriter, r *http.Request) {
	// JWT Authentication
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || len(authHeader) < 8 {
		sendErrorResponse(w, "Authorization header missing or invalid", http.StatusUnauthorized)
		return
	}

	tokenString := authHeader[7:] // Remove "Bearer "

	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(JWT_SECRET), nil
	})

	if err != nil {
		sendErrorResponse(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	var req CreateClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Basic Go-level validation before calling PostgreSQL function
	if err := validateCreateClientRequest(&req); err != nil {
		sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Start a transaction
	tx, err := app.DB.Begin()
	if err != nil {
		sendErrorResponse(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}

	// Ensure we rollback if something goes wrong
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Call the PostgreSQL function
	var result string
	query := "SELECT create_client($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)"
	err = tx.QueryRow(query,
		claims.UserID, req.Name, req.CIF, req.DOB, req.TradeRegisterNo,
		req.CountryID, req.StateID, req.City, req.StreetName, req.StreetNo,
		nullIfEmpty(req.Building), nullIfEmpty(req.Floor), nullIfEmpty(req.Apartment)).Scan(&result)

	if err != nil {
		sendErrorResponse(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if client creation was successful
	if result != "OK" {
		tx.Rollback()
		sendErrorResponse(w, result, http.StatusBadRequest)
		return
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		sendErrorResponse(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// Get the created client details
	var client Client
	clientQuery := `SELECT id, name, cif, 
                          TO_CHAR(dob, 'YYYY-MM-DD') as dob,
                          trade_register_no, country_id, state_id, 
                          city, street_name, street_no, 
                          COALESCE(building, '') as building, 
                          COALESCE(floor, '') as floor, 
                          COALESCE(apartment, '') as apartment,
                          TO_CHAR(created_on, 'YYYY-MM-DD') as created_on,
                          TO_CHAR(updated_on, 'YYYY-MM-DD') as updated_on,
                          created_by, updated_by
                   FROM clients 
                   WHERE UPPER(cif) = UPPER($1) AND created_by = $2
                   ORDER BY id DESC 
                   LIMIT 1`

	err = app.DB.QueryRow(clientQuery, req.CIF, claims.UserID).Scan(
		&client.ID, &client.Name, &client.CIF, &client.DOB,
		&client.TradeRegisterNo, &client.CountryID, &client.StateID,
		&client.City, &client.StreetName, &client.StreetNo, &client.Building,
		&client.Floor, &client.Apartment, &client.CreatedOn, &client.UpdatedOn,
		&client.CreatedBy, &client.UpdatedBy)

	if err != nil {
		// Client was created but couldn't fetch details
		sendSuccessResponse(w, "Client created successfully", map[string]interface{}{
			"status": "OK",
			"cif":    req.CIF,
			"name":   req.Name,
		})
		return
	}

	sendSuccessResponse(w, "Client created successfully", client)
}

// validateCreateClientRequest performs basic Go-level validation
func validateCreateClientRequest(req *CreateClientRequest) error {
	// Required fields validation
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(req.CIF) == "" {
		return fmt.Errorf("CIF is required")
	}
	if strings.TrimSpace(req.DOB) == "" {
		return fmt.Errorf("date of birth is required")
	}
	if strings.TrimSpace(req.TradeRegisterNo) == "" {
		return fmt.Errorf("trade register number is required")
	}
	if req.CountryID <= 0 {
		return fmt.Errorf("valid country ID is required")
	}
	if req.StateID <= 0 {
		return fmt.Errorf("valid state ID is required")
	}
	if strings.TrimSpace(req.City) == "" {
		return fmt.Errorf("city is required")
	}
	if strings.TrimSpace(req.StreetName) == "" {
		return fmt.Errorf("street name is required")
	}
	if strings.TrimSpace(req.StreetNo) == "" {
		return fmt.Errorf("street number is required")
	}

	// Length validations
	if len(req.Name) > 128 {
		return fmt.Errorf("name cannot exceed 128 characters")
	}
	if len(req.CIF) > 13 {
		return fmt.Errorf("CIF cannot exceed 13 characters")
	}
	if len(req.TradeRegisterNo) > 16 {
		return fmt.Errorf("trade register number cannot exceed 16 characters")
	}
	if len(req.City) > 64 {
		return fmt.Errorf("city cannot exceed 64 characters")
	}
	if len(req.StreetName) > 64 {
		return fmt.Errorf("street name cannot exceed 64 characters")
	}
	if len(req.StreetNo) > 16 {
		return fmt.Errorf("street number cannot exceed 16 characters")
	}
	if len(req.Building) > 16 {
		return fmt.Errorf("building cannot exceed 16 characters")
	}
	if len(req.Floor) > 8 {
		return fmt.Errorf("floor cannot exceed 8 characters")
	}
	if len(req.Apartment) > 8 {
		return fmt.Errorf("apartment cannot exceed 8 characters")
	}

	// Date format validation
	if _, err := time.Parse("2006-01-02", req.DOB); err != nil {
		return fmt.Errorf("date of birth must be in YYYY-MM-DD format")
	}

	return nil
}

// nullIfEmpty returns nil if string is empty, otherwise returns the string
func nullIfEmpty(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
