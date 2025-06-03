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
                          c.created_by, c.updated_by, 
                          (select q.name from countries q where q.id=c.country_id) country_name,
                          (select q.name from states q where q.id=c.state_id) state_name
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
			&client.CreatedBy, &client.UpdatedBy, &client.CountryName, &client.StateName)

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

// Add these structs to your existing code
type AddClientMembershipRequest struct {
	ClientID     int    `json:"client_id"`
	MembershipID int    `json:"membership_id"`
	ValidFrom    string `json:"valid_from"` // Expected format: "2024-01-15" (YYYY-MM-DD)
}

type ClientMembership struct {
	ID           int    `json:"id"`
	ClientID     int    `json:"client_id"`
	MembershipID int    `json:"membership_id"`
	StartingFrom string `json:"starting_from"`
	EndingOn     string `json:"ending_on"`
	Status       string `json:"status"`
	CreatedBy    int    `json:"created_by"`
	UpdatedBy    int    `json:"updated_by"`
	CreatedOn    string `json:"created_on"`
	UpdatedOn    string `json:"updated_on"`
}

func (app *App) addClientMembership(w http.ResponseWriter, r *http.Request) {
	// Authenticate user
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
		sendErrorResponse(w, "Invalid authorization header", http.StatusUnauthorized)
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

	var req AddClientMembershipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ClientID <= 0 {
		sendErrorResponse(w, "Valid client_id is required", http.StatusBadRequest)
		return
	}
	if req.MembershipID <= 0 {
		sendErrorResponse(w, "Valid membership_id is required", http.StatusBadRequest)
		return
	}
	if req.ValidFrom == "" {
		sendErrorResponse(w, "Valid valid_from date is required (format: YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	// Validate date format (basic validation)
	if len(req.ValidFrom) != 10 {
		sendErrorResponse(w, "Invalid date format. Expected YYYY-MM-DD", http.StatusBadRequest)
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
	query := "SELECT add_client_membership($1, $2, $3, $4)"
	err = tx.QueryRow(query, req.ClientID, req.MembershipID, req.ValidFrom, claims.UserID).Scan(&result)
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

	// Get the created client membership details (optional)
	var clientMembership ClientMembership
	clientMembershipQuery := `SELECT id, client_id, membership_id, 
                             TO_CHAR(starting_from, 'YYYY-MM-DD') as starting_from,
                             TO_CHAR(ending_on, 'YYYY-MM-DD') as ending_on,
                             status, created_by, updated_by,
                             TO_CHAR(created_on, 'YYYY-MM-DD HH24:MI:SS') as created_on,
                             TO_CHAR(updated_on, 'YYYY-MM-DD HH24:MI:SS') as updated_on
                             FROM client_memberships 
                             WHERE client_id = $1 AND membership_id = $2 
                               AND starting_from = $3
                             ORDER BY id DESC 
                             LIMIT 1`

	err = app.DB.QueryRow(clientMembershipQuery, req.ClientID, req.MembershipID, req.ValidFrom).Scan(
		&clientMembership.ID, &clientMembership.ClientID, &clientMembership.MembershipID,
		&clientMembership.StartingFrom, &clientMembership.EndingOn, &clientMembership.Status,
		&clientMembership.CreatedBy, &clientMembership.UpdatedBy,
		&clientMembership.CreatedOn, &clientMembership.UpdatedOn)

	if err != nil {
		// Client membership was created but couldn't fetch details
		sendSuccessResponse(w, "Client membership added successfully", map[string]interface{}{
			"status":        "OK",
			"client_id":     req.ClientID,
			"membership_id": req.MembershipID,
			"valid_from":    req.ValidFrom,
		})
		return
	}

	sendSuccessResponse(w, "Client membership added successfully", clientMembership)
}

// Alternative implementation using path parameters instead of JSON body
func (app *App) addClientMembershipByPath(w http.ResponseWriter, r *http.Request) {
	// Authenticate user
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
		sendErrorResponse(w, "Invalid authorization header", http.StatusUnauthorized)
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

	vars := mux.Vars(r)

	clientIDStr := vars["client_id"]
	membershipIDStr := vars["membership_id"]
	validFrom := vars["valid_from"] // Expected format: 2024-01-15

	clientID, err := strconv.Atoi(clientIDStr)
	if err != nil || clientID <= 0 {
		sendErrorResponse(w, "Invalid client_id parameter", http.StatusBadRequest)
		return
	}

	membershipID, err := strconv.Atoi(membershipIDStr)
	if err != nil || membershipID <= 0 {
		sendErrorResponse(w, "Invalid membership_id parameter", http.StatusBadRequest)
		return
	}

	if validFrom == "" || len(validFrom) != 10 {
		sendErrorResponse(w, "Invalid valid_from parameter. Expected format: YYYY-MM-DD", http.StatusBadRequest)
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
	query := "SELECT add_client_membership($1, $2, $3, $4)"
	err = tx.QueryRow(query, clientID, membershipID, validFrom, claims.UserID).Scan(&result)
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

	sendSuccessResponse(w, "Client membership added successfully", map[string]interface{}{
		"status":        "OK",
		"client_id":     clientID,
		"membership_id": membershipID,
		"valid_from":    validFrom,
	})
}

// Add these structs to your existing code
type ClientCheckInRequest struct {
	ClientID int `json:"client_id"`
	GymID    int `json:"gym_id"`
}

type ClientPass struct {
	ID        int    `json:"id"`
	GymID     int    `json:"gym_id"`
	ClientID  int    `json:"client_id"`
	Action    string `json:"action"`
	CreatedBy int    `json:"created_by"`
	CreatedOn string `json:"created_on"`
}

func (app *App) doClientCheckInGym(w http.ResponseWriter, r *http.Request) {
	// Authenticate user
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
		sendErrorResponse(w, "Invalid authorization header", http.StatusUnauthorized)
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

	var req ClientCheckInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ClientID <= 0 {
		sendErrorResponse(w, "Valid client_id is required", http.StatusBadRequest)
		return
	}
	if req.GymID <= 0 {
		sendErrorResponse(w, "Valid gym_id is required", http.StatusBadRequest)
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
	query := "SELECT do_client_check_in_gym($1, $2, $3)"
	err = tx.QueryRow(query, req.ClientID, req.GymID, claims.UserID).Scan(&result)
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

	// Get the created client pass details and updated gym stats (optional)
	var clientPass ClientPass
	clientPassQuery := `SELECT id, gym_id, client_id, action, created_by,
                        TO_CHAR(created_on, 'YYYY-MM-DD HH24:MI:SS') as created_on
                        FROM client_passes 
                        WHERE client_id = $1 AND gym_id = $2 AND action = 'in'
                        ORDER BY id DESC 
                        LIMIT 1`

	err = app.DB.QueryRow(clientPassQuery, req.ClientID, req.GymID).Scan(
		&clientPass.ID, &clientPass.GymID, &clientPass.ClientID,
		&clientPass.Action, &clientPass.CreatedBy, &clientPass.CreatedOn)

	if err != nil {
		// Check-in was successful but couldn't fetch pass details
		sendSuccessResponse(w, "Client checked in successfully", map[string]interface{}{
			"status":    "OK",
			"client_id": req.ClientID,
			"gym_id":    req.GymID,
			"action":    "in",
		})
		return
	}

	// Also get updated gym stats
	var gymStats GymStats
	gymStatsQuery := `SELECT id, gym_id, current_people, current_combined, max_people, max_resevations
                      FROM gym_stats 
                      WHERE gym_id = $1`

	err = app.DB.QueryRow(gymStatsQuery, req.GymID).Scan(
		&gymStats.ID, &gymStats.GymID, &gymStats.CurrentPeople,
		&gymStats.CurrentCombined, &gymStats.MaxPeople, &gymStats.MaxReservations)

	responseData := map[string]interface{}{
		"client_pass": clientPass,
	}

	if err == nil {
		responseData["gym_stats"] = gymStats
	}

	sendSuccessResponse(w, "Client checked in successfully", responseData)
}

// Alternative implementation using path parameters instead of JSON body
func (app *App) doClientCheckInGymByPath(w http.ResponseWriter, r *http.Request) {
	// Authenticate user
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
		sendErrorResponse(w, "Invalid authorization header", http.StatusUnauthorized)
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

	vars := mux.Vars(r)

	clientIDStr := vars["client_id"]
	gymIDStr := vars["gym_id"]

	clientID, err := strconv.Atoi(clientIDStr)
	if err != nil || clientID <= 0 {
		sendErrorResponse(w, "Invalid client_id parameter", http.StatusBadRequest)
		return
	}

	gymID, err := strconv.Atoi(gymIDStr)
	if err != nil || gymID <= 0 {
		sendErrorResponse(w, "Invalid gym_id parameter", http.StatusBadRequest)
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
	query := "SELECT do_client_check_in_gym($1, $2, $3)"
	err = tx.QueryRow(query, clientID, gymID, claims.UserID).Scan(&result)
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

	sendSuccessResponse(w, "Client checked in successfully", map[string]interface{}{
		"status":    "OK",
		"client_id": clientID,
		"gym_id":    gymID,
		"action":    "in",
	})
}

// Add this struct to your existing code (reuse the same request struct as check-in)
type ClientCheckOutRequest struct {
	ClientID int `json:"client_id"`
	GymID    int `json:"gym_id"`
}

// ClientPass struct is already defined in the check-in implementation
// GymStats struct is already defined in the check-in implementation

func (app *App) doClientCheckOutGym(w http.ResponseWriter, r *http.Request) {
	// Authenticate user
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
		sendErrorResponse(w, "Invalid authorization header", http.StatusUnauthorized)
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

	var req ClientCheckOutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ClientID <= 0 {
		sendErrorResponse(w, "Valid client_id is required", http.StatusBadRequest)
		return
	}
	if req.GymID <= 0 {
		sendErrorResponse(w, "Valid gym_id is required", http.StatusBadRequest)
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
	query := "SELECT do_client_check_out_gym($1, $2, $3)"
	err = tx.QueryRow(query, req.ClientID, req.GymID, claims.UserID).Scan(&result)
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

	// Get the created client pass details and updated gym stats (optional)
	var clientPass ClientPass
	clientPassQuery := `SELECT id, gym_id, client_id, action, created_by,
                        TO_CHAR(created_on, 'YYYY-MM-DD HH24:MI:SS') as created_on
                        FROM client_passes 
                        WHERE client_id = $1 AND gym_id = $2 AND action = 'out'
                        ORDER BY id DESC 
                        LIMIT 1`

	err = app.DB.QueryRow(clientPassQuery, req.ClientID, req.GymID).Scan(
		&clientPass.ID, &clientPass.GymID, &clientPass.ClientID,
		&clientPass.Action, &clientPass.CreatedBy, &clientPass.CreatedOn)

	if err != nil {
		// Check-out was successful but couldn't fetch pass details
		sendSuccessResponse(w, "Client checked out successfully", map[string]interface{}{
			"status":    "OK",
			"client_id": req.ClientID,
			"gym_id":    req.GymID,
			"action":    "out",
		})
		return
	}

	// Also get updated gym stats
	var gymStats GymStats
	gymStatsQuery := `SELECT id, gym_id, current_people, current_combined, max_people, max_resevations
                      FROM gym_stats 
                      WHERE gym_id = $1`

	err = app.DB.QueryRow(gymStatsQuery, req.GymID).Scan(
		&gymStats.ID, &gymStats.GymID, &gymStats.CurrentPeople,
		&gymStats.CurrentCombined, &gymStats.MaxPeople, &gymStats.MaxReservations)

	responseData := map[string]interface{}{
		"client_pass": clientPass,
	}

	if err == nil {
		responseData["gym_stats"] = gymStats
	}

	sendSuccessResponse(w, "Client checked out successfully", responseData)
}

// Alternative implementation using path parameters instead of JSON body
func (app *App) doClientCheckOutGymByPath(w http.ResponseWriter, r *http.Request) {
	// Authenticate user
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
		sendErrorResponse(w, "Invalid authorization header", http.StatusUnauthorized)
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

	vars := mux.Vars(r)

	clientIDStr := vars["client_id"]
	gymIDStr := vars["gym_id"]

	clientID, err := strconv.Atoi(clientIDStr)
	if err != nil || clientID <= 0 {
		sendErrorResponse(w, "Invalid client_id parameter", http.StatusBadRequest)
		return
	}

	gymID, err := strconv.Atoi(gymIDStr)
	if err != nil || gymID <= 0 {
		sendErrorResponse(w, "Invalid gym_id parameter", http.StatusBadRequest)
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
	query := "SELECT do_client_check_out_gym($1, $2, $3)"
	err = tx.QueryRow(query, clientID, gymID, claims.UserID).Scan(&result)
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

	sendSuccessResponse(w, "Client checked out successfully", map[string]interface{}{
		"status":    "OK",
		"client_id": clientID,
		"gym_id":    gymID,
		"action":    "out",
	})
}

// Additional helper function to get client's current gym status
func (app *App) getClientGymStatus(w http.ResponseWriter, r *http.Request) {
	// Authenticate user
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
		sendErrorResponse(w, "Invalid authorization header", http.StatusUnauthorized)
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

	vars := mux.Vars(r)
	clientIDStr := vars["client_id"]
	gymIDStr := vars["gym_id"]

	clientID, err := strconv.Atoi(clientIDStr)
	if err != nil || clientID <= 0 {
		sendErrorResponse(w, "Invalid client_id parameter", http.StatusBadRequest)
		return
	}

	gymID, err := strconv.Atoi(gymIDStr)
	if err != nil || gymID <= 0 {
		sendErrorResponse(w, "Invalid gym_id parameter", http.StatusBadRequest)
		return
	}

	// Check if client is currently checked in today
	var lastPass ClientPass
	lastPassQuery := `SELECT id, gym_id, client_id, action, created_by,
                      TO_CHAR(created_on, 'YYYY-MM-DD HH24:MI:SS') as created_on
                      FROM client_passes 
                      WHERE client_id = $1 AND gym_id = $2 
                        AND DATE(created_on) = CURRENT_DATE
                      ORDER BY id DESC 
                      LIMIT 1`

	err = app.DB.QueryRow(lastPassQuery, clientID, gymID).Scan(
		&lastPass.ID, &lastPass.GymID, &lastPass.ClientID,
		&lastPass.Action, &lastPass.CreatedBy, &lastPass.CreatedOn)

	if err != nil {
		// No passes found for today
		sendSuccessResponse(w, "Client gym status retrieved", map[string]interface{}{
			"client_id":     clientID,
			"gym_id":        gymID,
			"status":        "not_visited_today",
			"last_action":   nil,
			"can_check_in":  true,
			"can_check_out": false,
		})
		return
	}

	// Determine current status based on last action
	status := "checked_out"
	canCheckIn := true
	canCheckOut := false

	if lastPass.Action == "in" {
		status = "checked_in"
		canCheckIn = false
		canCheckOut = true
	}

	sendSuccessResponse(w, "Client gym status retrieved", map[string]interface{}{
		"client_id":     clientID,
		"gym_id":        gymID,
		"status":        status,
		"last_action":   lastPass,
		"can_check_in":  canCheckIn,
		"can_check_out": canCheckOut,
	})
}

// Update Client Request struct
type UpdateClientRequest struct {
	Name            string `json:"name,omitempty"`
	CIF             string `json:"cif,omitempty"`
	DOB             string `json:"dob,omitempty"` // Format: "2006-01-02"
	TradeRegisterNo string `json:"trade_register_no,omitempty"`
	CountryID       int    `json:"country_id,omitempty"`
	StateID         int    `json:"state_id,omitempty"`
	City            string `json:"city,omitempty"`
	StreetName      string `json:"street_name,omitempty"`
	StreetNo        string `json:"street_no,omitempty"`
	Building        string `json:"building,omitempty"`
	Floor           string `json:"floor,omitempty"`
	Apartment       string `json:"apartment,omitempty"`
}

// Update Client function
func (app *App) updateClient(w http.ResponseWriter, r *http.Request) {
	// Authenticate user
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
		sendErrorResponse(w, "Invalid authorization header", http.StatusUnauthorized)
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

	// Get client ID from URL
	vars := mux.Vars(r)
	clientIDStr := vars["client_id"]
	clientID, err := strconv.Atoi(clientIDStr)
	if err != nil || clientID <= 0 {
		sendErrorResponse(w, "Invalid client_id parameter", http.StatusBadRequest)
		return
	}

	var req UpdateClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Check if user has permission to update this client
	var exists bool
	permissionQuery := `SELECT EXISTS(SELECT 1 FROM user_clients WHERE user_id = $1 AND client_id = $2)`
	err = app.DB.QueryRow(permissionQuery, claims.UserID, clientID).Scan(&exists)
	if err != nil || !exists {
		sendErrorResponse(w, "Client not found or access denied", http.StatusForbidden)
		return
	}

	// Validate update request
	if err := validateUpdateClientRequest(&req); err != nil {
		sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Start transaction
	tx, err := app.DB.Begin()
	if err != nil {
		sendErrorResponse(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Build dynamic update query
	updateFields := make([]string, 0)
	args := make([]interface{}, 0)
	argIndex := 1

	if req.Name != "" {
		updateFields = append(updateFields, "name = $"+strconv.Itoa(argIndex))
		args = append(args, req.Name)
		argIndex++
	}
	if req.CIF != "" {
		// Check if CIF already exists for another client
		var existingClientID int
		cifQuery := `SELECT id FROM clients WHERE UPPER(cif) = UPPER($1) AND id != $2`
		err = tx.QueryRow(cifQuery, req.CIF, clientID).Scan(&existingClientID)
		if err == nil {
			sendErrorResponse(w, "CIF already exists for another client", http.StatusConflict)
			return
		}
		updateFields = append(updateFields, "cif = $"+strconv.Itoa(argIndex))
		args = append(args, req.CIF)
		argIndex++
	}
	if req.DOB != "" {
		updateFields = append(updateFields, "dob = $"+strconv.Itoa(argIndex))
		args = append(args, req.DOB)
		argIndex++
	}
	if req.TradeRegisterNo != "" {
		updateFields = append(updateFields, "trade_register_no = $"+strconv.Itoa(argIndex))
		args = append(args, req.TradeRegisterNo)
		argIndex++
	}
	if req.CountryID > 0 {
		// Verify country exists
		var countryExists bool
		err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM countries WHERE id = $1)", req.CountryID).Scan(&countryExists)
		if err != nil || !countryExists {
			sendErrorResponse(w, "Invalid country_id", http.StatusBadRequest)
			return
		}
		updateFields = append(updateFields, "country_id = $"+strconv.Itoa(argIndex))
		args = append(args, req.CountryID)
		argIndex++
	}
	if req.StateID > 0 {
		// Verify state exists
		var stateExists bool
		err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM states WHERE id = $1)", req.StateID).Scan(&stateExists)
		if err != nil || !stateExists {
			sendErrorResponse(w, "Invalid state_id", http.StatusBadRequest)
			return
		}
		updateFields = append(updateFields, "state_id = $"+strconv.Itoa(argIndex))
		args = append(args, req.StateID)
		argIndex++
	}
	if req.City != "" {
		updateFields = append(updateFields, "city = $"+strconv.Itoa(argIndex))
		args = append(args, req.City)
		argIndex++
	}
	if req.StreetName != "" {
		updateFields = append(updateFields, "street_name = $"+strconv.Itoa(argIndex))
		args = append(args, req.StreetName)
		argIndex++
	}
	if req.StreetNo != "" {
		updateFields = append(updateFields, "street_no = $"+strconv.Itoa(argIndex))
		args = append(args, req.StreetNo)
		argIndex++
	}
	if req.Building != "" {
		updateFields = append(updateFields, "building = $"+strconv.Itoa(argIndex))
		args = append(args, nullIfEmpty(req.Building))
		argIndex++
	}
	if req.Floor != "" {
		updateFields = append(updateFields, "floor = $"+strconv.Itoa(argIndex))
		args = append(args, nullIfEmpty(req.Floor))
		argIndex++
	}
	if req.Apartment != "" {
		updateFields = append(updateFields, "apartment = $"+strconv.Itoa(argIndex))
		args = append(args, nullIfEmpty(req.Apartment))
		argIndex++
	}

	// Always update updated_by
	updateFields = append(updateFields, "updated_by = $"+strconv.Itoa(argIndex))
	args = append(args, claims.UserID)
	argIndex++

	// Add client_id as the last parameter for WHERE clause
	args = append(args, clientID)

	if len(updateFields) > 1 { // More than just updated_by
		clientUpdateQuery := `UPDATE clients SET ` +
			updateFields[0]
		for i := 1; i < len(updateFields); i++ {
			clientUpdateQuery += ", " + updateFields[i]
		}
		clientUpdateQuery += " WHERE id = $" + strconv.Itoa(argIndex)

		_, err = tx.Exec(clientUpdateQuery, args...)
		if err != nil {
			sendErrorResponse(w, "Failed to update client: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		sendErrorResponse(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// Get updated client details
	var client Client
	clientQuery := `SELECT c.id, c.name, c.cif, 
                          TO_CHAR(c.dob, 'YYYY-MM-DD') as dob,
                          c.trade_register_no, c.country_id, c.state_id, 
                          c.city, c.street_name, c.street_no, 
                          COALESCE(c.building, '') as building, 
                          COALESCE(c.floor, '') as floor, 
                          COALESCE(c.apartment, '') as apartment,
                          TO_CHAR(c.created_on, 'YYYY-MM-DD') as created_on,
                          TO_CHAR(c.updated_on, 'YYYY-MM-DD') as updated_on,
                          c.created_by, c.updated_by
                   FROM clients c
                   WHERE c.id = $1`

	err = app.DB.QueryRow(clientQuery, clientID).Scan(
		&client.ID, &client.Name, &client.CIF, &client.DOB,
		&client.TradeRegisterNo, &client.CountryID, &client.StateID,
		&client.City, &client.StreetName, &client.StreetNo, &client.Building,
		&client.Floor, &client.Apartment, &client.CreatedOn, &client.UpdatedOn,
		&client.CreatedBy, &client.UpdatedBy)

	if err != nil {
		sendSuccessResponse(w, "Client updated successfully", map[string]interface{}{
			"status":    "OK",
			"client_id": clientID,
		})
		return
	}

	sendSuccessResponse(w, "Client updated successfully", client)
}

// Delete Client function
func (app *App) deleteClient(w http.ResponseWriter, r *http.Request) {
	// Authenticate user
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
		sendErrorResponse(w, "Invalid authorization header", http.StatusUnauthorized)
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

	// Get client ID from URL
	vars := mux.Vars(r)
	clientIDStr := vars["client_id"]
	clientID, err := strconv.Atoi(clientIDStr)
	if err != nil || clientID <= 0 {
		sendErrorResponse(w, "Invalid client_id parameter", http.StatusBadRequest)
		return
	}

	// Check if user has permission to delete this client
	var clientName string
	var exists bool
	permissionQuery := `SELECT c.name, EXISTS(SELECT 1 FROM user_clients uc WHERE uc.user_id = $1 AND uc.client_id = $2)
	                   FROM clients c 
	                   WHERE c.id = $2`
	err = app.DB.QueryRow(permissionQuery, claims.UserID, clientID).Scan(&clientName, &exists)
	if err != nil || !exists {
		sendErrorResponse(w, "Client not found or access denied", http.StatusForbidden)
		return
	}

	// Check if client has active memberships
	var activeMemberships int
	membershipQuery := `SELECT COUNT(*) FROM client_memberships 
	                   WHERE client_id = $1 AND status = 'active' 
	                   AND CURRENT_DATE BETWEEN starting_from AND ending_on`
	err = app.DB.QueryRow(membershipQuery, clientID).Scan(&activeMemberships)
	if err == nil && activeMemberships > 0 {
		sendErrorResponse(w, "Cannot delete client with active memberships", http.StatusConflict)
		return
	}

	// Check if client is currently checked in to any gym
	var currentCheckins int
	checkinQuery := `SELECT COUNT(DISTINCT gym_id) FROM client_passes cp1
	                 WHERE cp1.client_id = $1 
	                 AND cp1.action = 'in'
	                 AND DATE(cp1.created_on) = CURRENT_DATE
	                 AND NOT EXISTS (
	                     SELECT 1 FROM client_passes cp2 
	                     WHERE cp2.client_id = cp1.client_id 
	                     AND cp2.gym_id = cp1.gym_id
	                     AND cp2.action = 'out'
	                     AND DATE(cp2.created_on) = CURRENT_DATE
	                     AND cp2.created_on > cp1.created_on
	                 )`
	err = app.DB.QueryRow(checkinQuery, clientID).Scan(&currentCheckins)
	if err == nil && currentCheckins > 0 {
		sendErrorResponse(w, "Cannot delete client who is currently checked in to gym(s)", http.StatusConflict)
		return
	}

	// Start transaction
	tx, err := app.DB.Begin()
	if err != nil {
		sendErrorResponse(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Delete related records (CASCADE should handle most, but let's be explicit)

	// Delete client passes
	_, err = tx.Exec("DELETE FROM client_passes WHERE client_id = $1", clientID)
	if err != nil {
		sendErrorResponse(w, "Failed to delete client passes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete client memberships
	_, err = tx.Exec("DELETE FROM client_memberships WHERE client_id = $1", clientID)
	if err != nil {
		sendErrorResponse(w, "Failed to delete client memberships: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete user-client relationships
	_, err = tx.Exec("DELETE FROM user_clients WHERE client_id = $1", clientID)
	if err != nil {
		sendErrorResponse(w, "Failed to delete user-client relationships: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Finally, delete the client itself
	result, err := tx.Exec("DELETE FROM clients WHERE id = $1", clientID)
	if err != nil {
		sendErrorResponse(w, "Failed to delete client: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if client was actually deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		sendErrorResponse(w, "Client not found", http.StatusNotFound)
		return
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		sendErrorResponse(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	sendSuccessResponse(w, "Client deleted successfully", map[string]interface{}{
		"status":      "OK",
		"client_id":   clientID,
		"client_name": clientName,
		"message":     "Client and all related data have been permanently deleted",
	})
}

// Remove User from Client
func (app *App) removeUserFromClient(w http.ResponseWriter, r *http.Request) {
	// Authenticate user
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
		sendErrorResponse(w, "Invalid authorization header", http.StatusUnauthorized)
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

	// Get parameters from URL
	vars := mux.Vars(r)
	clientIDStr := vars["client_id"]
	userIDStr := vars["user_id"]

	clientID, err := strconv.Atoi(clientIDStr)
	if err != nil || clientID <= 0 {
		sendErrorResponse(w, "Invalid client_id parameter", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil || userID <= 0 {
		sendErrorResponse(w, "Invalid user_id parameter", http.StatusBadRequest)
		return
	}

	// Check if requesting user has permission (must be associated with the client)
	var requestingUserExists bool
	permissionQuery := `SELECT EXISTS(SELECT 1 FROM user_clients WHERE user_id = $1 AND client_id = $2)`
	err = app.DB.QueryRow(permissionQuery, claims.UserID, clientID).Scan(&requestingUserExists)
	if err != nil || !requestingUserExists {
		sendErrorResponse(w, "Client not found or access denied", http.StatusForbidden)
		return
	}

	// Check if target user exists in client
	var targetUserName string
	var targetUserExists bool
	targetQuery := `SELECT u.username, EXISTS(SELECT 1 FROM user_clients uc WHERE uc.user_id = $1 AND uc.client_id = $2)
	               FROM users u 
	               WHERE u.id = $1`
	err = app.DB.QueryRow(targetQuery, userID, clientID).Scan(&targetUserName, &targetUserExists)
	if err != nil || !targetUserExists {
		sendErrorResponse(w, "User not found in this client", http.StatusNotFound)
		return
	}

	// Check if this would leave the client with no users
	var userCount int
	userCountQuery := `SELECT COUNT(*) FROM user_clients WHERE client_id = $1`
	err = app.DB.QueryRow(userCountQuery, clientID).Scan(&userCount)
	if err == nil && userCount <= 1 {
		sendErrorResponse(w, "Cannot remove the last user from the client", http.StatusConflict)
		return
	}

	// Remove user from client
	result, err := app.DB.Exec("DELETE FROM user_clients WHERE user_id = $1 AND client_id = $2", userID, clientID)
	if err != nil {
		sendErrorResponse(w, "Failed to remove user from client: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		sendErrorResponse(w, "User-client relationship not found", http.StatusNotFound)
		return
	}

	sendSuccessResponse(w, "User removed from client successfully", map[string]interface{}{
		"status":    "OK",
		"client_id": clientID,
		"user_id":   userID,
		"username":  targetUserName,
	})
}

// Remove Client Membership
func (app *App) removeClientMembership(w http.ResponseWriter, r *http.Request) {
	// Authenticate user
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
		sendErrorResponse(w, "Invalid authorization header", http.StatusUnauthorized)
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

	// Get parameters from URL
	vars := mux.Vars(r)
	clientIDStr := vars["client_id"]
	membershipIDStr := vars["membership_id"]

	clientID, err := strconv.Atoi(clientIDStr)
	if err != nil || clientID <= 0 {
		sendErrorResponse(w, "Invalid client_id parameter", http.StatusBadRequest)
		return
	}

	membershipID, err := strconv.Atoi(membershipIDStr)
	if err != nil || membershipID <= 0 {
		sendErrorResponse(w, "Invalid membership_id parameter", http.StatusBadRequest)
		return
	}

	// Check if user has permission for this client
	var userExists bool
	permissionQuery := `SELECT EXISTS(SELECT 1 FROM user_clients WHERE user_id = $1 AND client_id = $2)`
	err = app.DB.QueryRow(permissionQuery, claims.UserID, clientID).Scan(&userExists)
	if err != nil || !userExists {
		sendErrorResponse(w, "Client not found or access denied", http.StatusForbidden)
		return
	}

	// Check if membership is currently active
	var isActive bool
	var startDate, endDate string
	activeQuery := `SELECT 
	                   CASE WHEN status = 'active' AND CURRENT_DATE BETWEEN starting_from AND ending_on THEN true ELSE false END,
	                   TO_CHAR(starting_from, 'YYYY-MM-DD'),
	                   TO_CHAR(ending_on, 'YYYY-MM-DD')
	               FROM client_memberships 
	               WHERE client_id = $1 AND membership_id = $2
	               ORDER BY id DESC LIMIT 1`
	err = app.DB.QueryRow(activeQuery, clientID, membershipID).Scan(&isActive, &startDate, &endDate)
	if err != nil {
		sendErrorResponse(w, "Client membership not found", http.StatusNotFound)
		return
	}

	if isActive {
		sendErrorResponse(w, "Cannot remove active membership. Please deactivate first or wait until expiry.", http.StatusConflict)
		return
	}

	// Remove client membership
	result, err := app.DB.Exec("DELETE FROM client_memberships WHERE client_id = $1 AND membership_id = $2",
		clientID, membershipID)
	if err != nil {
		sendErrorResponse(w, "Failed to remove client membership: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		sendErrorResponse(w, "Client membership not found", http.StatusNotFound)
		return
	}

	sendSuccessResponse(w, "Client membership removed successfully", map[string]interface{}{
		"status":        "OK",
		"client_id":     clientID,
		"membership_id": membershipID,
		"start_date":    startDate,
		"end_date":      endDate,
	})
}

// Deactivate Client Membership (alternative to removal)
func (app *App) deactivateClientMembership(w http.ResponseWriter, r *http.Request) {
	// Authenticate user
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
		sendErrorResponse(w, "Invalid authorization header", http.StatusUnauthorized)
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

	// Get parameters from URL
	vars := mux.Vars(r)
	clientIDStr := vars["client_id"]
	membershipIDStr := vars["membership_id"]

	clientID, err := strconv.Atoi(clientIDStr)
	if err != nil || clientID <= 0 {
		sendErrorResponse(w, "Invalid client_id parameter", http.StatusBadRequest)
		return
	}

	membershipID, err := strconv.Atoi(membershipIDStr)
	if err != nil || membershipID <= 0 {
		sendErrorResponse(w, "Invalid membership_id parameter", http.StatusBadRequest)
		return
	}

	// Check if user has permission for this client
	var userExists bool
	permissionQuery := `SELECT EXISTS(SELECT 1 FROM user_clients WHERE user_id = $1 AND client_id = $2)`
	err = app.DB.QueryRow(permissionQuery, claims.UserID, clientID).Scan(&userExists)
	if err != nil || !userExists {
		sendErrorResponse(w, "Client not found or access denied", http.StatusForbidden)
		return
	}

	// Deactivate the membership
	result, err := app.DB.Exec(`UPDATE client_memberships 
	                           SET status = 'suspended', 
	                               updated_by = $3, 
	                               updated_on = CURRENT_TIMESTAMP
	                           WHERE client_id = $1 AND membership_id = $2 AND status = 'active'`,
		clientID, membershipID, claims.UserID)
	if err != nil {
		sendErrorResponse(w, "Failed to deactivate client membership: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		sendErrorResponse(w, "Active client membership not found", http.StatusNotFound)
		return
	}

	sendSuccessResponse(w, "Client membership deactivated successfully", map[string]interface{}{
		"status":        "OK",
		"client_id":     clientID,
		"membership_id": membershipID,
		"new_status":    "suspended",
	})
}

// Validation function for update requests
func validateUpdateClientRequest(req *UpdateClientRequest) error {
	// Length validations for provided fields
	if req.Name != "" && len(req.Name) > 128 {
		return fmt.Errorf("name cannot exceed 128 characters")
	}
	if req.CIF != "" && len(req.CIF) > 13 {
		return fmt.Errorf("CIF cannot exceed 13 characters")
	}
	if req.TradeRegisterNo != "" && len(req.TradeRegisterNo) > 16 {
		return fmt.Errorf("trade register number cannot exceed 16 characters")
	}
	if req.City != "" && len(req.City) > 64 {
		return fmt.Errorf("city cannot exceed 64 characters")
	}
	if req.StreetName != "" && len(req.StreetName) > 64 {
		return fmt.Errorf("street name cannot exceed 64 characters")
	}
	if req.StreetNo != "" && len(req.StreetNo) > 16 {
		return fmt.Errorf("street number cannot exceed 16 characters")
	}
	if req.Building != "" && len(req.Building) > 16 {
		return fmt.Errorf("building cannot exceed 16 characters")
	}
	if req.Floor != "" && len(req.Floor) > 8 {
		return fmt.Errorf("floor cannot exceed 8 characters")
	}
	if req.Apartment != "" && len(req.Apartment) > 8 {
		return fmt.Errorf("apartment cannot exceed 8 characters")
	}

	// Date format validation if provided
	if req.DOB != "" {
		if _, err := time.Parse("2006-01-02", req.DOB); err != nil {
			return fmt.Errorf("date of birth must be in YYYY-MM-DD format")
		}
	}

	// Required field validation (if provided, cannot be empty)
	if req.Name == "" && req.Name != "" {
		return fmt.Errorf("name cannot be empty")
	}
	if strings.TrimSpace(req.CIF) == "" && req.CIF != "" {
		return fmt.Errorf("CIF cannot be empty")
	}

	return nil
}

// Get Client by ID (for viewing client details)
func (app *App) getClientByID(w http.ResponseWriter, r *http.Request) {
	// Authenticate user
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
		sendErrorResponse(w, "Invalid authorization header", http.StatusUnauthorized)
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

	// Get client ID from URL
	vars := mux.Vars(r)
	clientIDStr := vars["client_id"]
	clientID, err := strconv.Atoi(clientIDStr)
	if err != nil || clientID <= 0 {
		sendErrorResponse(w, "Invalid client_id parameter", http.StatusBadRequest)
		return
	}

	// Get client details with country and state names
	var client Client
	clientQuery := `SELECT c.id, c.name, c.cif, 
                          TO_CHAR(c.dob, 'YYYY-MM-DD') as dob,
                          c.trade_register_no, c.country_id, c.state_id, 
                          c.city, c.street_name, c.street_no, 
                          COALESCE(c.building, '') as building, 
                          COALESCE(c.floor, '') as floor, 
                          COALESCE(c.apartment, '') as apartment,
                          TO_CHAR(c.created_on, 'YYYY-MM-DD') as created_on,
                          TO_CHAR(c.updated_on, 'YYYY-MM-DD') as updated_on,
                          c.created_by, c.updated_by,
                          COALESCE(co.name, '') as country_name,
                          COALESCE(s.name, '') as state_name
                   FROM clients c
                   INNER JOIN user_clients uc ON uc.client_id = c.id
                   LEFT JOIN countries co ON c.country_id = co.id
                   LEFT JOIN states s ON c.state_id = s.id
                   WHERE c.id = $1 AND uc.user_id = $2`

	err = app.DB.QueryRow(clientQuery, clientID, claims.UserID).Scan(
		&client.ID, &client.Name, &client.CIF, &client.DOB,
		&client.TradeRegisterNo, &client.CountryID, &client.StateID,
		&client.City, &client.StreetName, &client.StreetNo, &client.Building,
		&client.Floor, &client.Apartment, &client.CreatedOn, &client.UpdatedOn,
		&client.CreatedBy, &client.UpdatedBy, &client.CountryName, &client.StateName)

	if err != nil {
		sendErrorResponse(w, "Client not found or access denied", http.StatusNotFound)
		return
	}

	sendSuccessResponse(w, "Client retrieved successfully", client)
}
