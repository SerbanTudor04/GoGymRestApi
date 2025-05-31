package server

import (
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

// Add this struct to your existing code
type CreateGymRequest struct {
	Name            string `json:"name"`
	MaxPeople       int    `json:"max_people"`
	MaxReservations int    `json:"max_reservations"`
}

type Gym struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	Members         int    `json:"members"`
	MaxPeople       int    `json:"max_people,omitempty"`
	MaxReservations int    `json:"max_reservations,omitempty"`
}

func (app *App) createGym(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	tokenString := authHeader[7:] // Remove "Bearer "

	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(JWT_SECRET), nil
	})

	if err != nil {
		sendErrorResponse(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	var req CreateGymRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" {
		sendErrorResponse(w, "Name is required", http.StatusBadRequest)
		return
	}
	if req.MaxPeople <= 0 {
		sendErrorResponse(w, "Max people must be greater than 0", http.StatusBadRequest)
		return
	}
	if req.MaxReservations <= 0 {
		sendErrorResponse(w, "Max reservations must be greater than 0", http.StatusBadRequest)
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
	query := "SELECT create_gym($1, $2, $3, $4)"
	err = tx.QueryRow(query, req.Name, req.MaxPeople, req.MaxReservations, claims.UserID).Scan(&result)
	if err != nil {
		sendErrorResponse(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if gym creation was successful
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

	// Get the created gym details
	var gym Gym
	gymQuery := `SELECT g.id, g.name, g.members, gs.max_people, gs.max_resevations 
                 FROM gyms g 
                 JOIN gym_stats gs ON g.id = gs.gym_id 
                 WHERE UPPER(g.name) = UPPER($1)
                 ORDER BY g.id DESC 
                 LIMIT 1`

	err = app.DB.QueryRow(gymQuery, req.Name).Scan(
		&gym.ID, &gym.Name, &gym.Members, &gym.MaxPeople, &gym.MaxReservations)

	if err != nil {
		// Gym was created but couldn't fetch details
		sendSuccessResponse(w, "Gym created successfully", map[string]interface{}{
			"status": "OK",
			"name":   req.Name,
		})
		return
	}

	sendSuccessResponse(w, "Gym created successfully", gym)
}

func (app *App) getGyms(w http.ResponseWriter, r *http.Request) {

	authHeader := r.Header.Get("Authorization")
	tokenString := authHeader[7:] // Remove "Bearer "

	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(JWT_SECRET), nil
	})
	if err != nil {
		sendErrorResponse(w, "Invalid token", http.StatusUnauthorized)
		return
	}
	var gym Gym
	gymQuery := `select  g.id, g.name, g.members, s.max_people, s.max_resevations
				from gyms g
				inner join gym_stats s on g.id = s.gym_id
				inner join user_gyms us on us.gym_id = g.id
				where us.user_id = $1`

	var gyms []Gym
	rows, err := app.DB.Query(gymQuery, claims.UserID)
	if err != nil {
		sendErrorResponse(w, "Failed to fetch gyms. "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&gym.ID, &gym.Name, &gym.Members, &gym.MaxPeople, &gym.MaxReservations)
		fmt.Println(gym.ID)
		gyms = append(gyms, gym)
	}

	sendSuccessResponse(w, "Gyms fetched successfully", gyms)
}

type AddUserToGymRequest struct {
	UserID int `json:"user_id"`
	GymID  int `json:"gym_id"`
}

type UserGym struct {
	ID        int    `json:"id"`
	UserID    int    `json:"user_id"`
	GymID     int    `json:"gym_id"`
	CreatedOn string `json:"created_on"`
}

func (app *App) addUserToGym(w http.ResponseWriter, r *http.Request) {
	var req AddUserToGymRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.UserID <= 0 {
		sendErrorResponse(w, "Valid user_id is required", http.StatusBadRequest)
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
	query := "SELECT ADD_USER_TO_GYM($1, $2)"
	err = tx.QueryRow(query, req.UserID, req.GymID).Scan(&result)
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

	// Get the created user-gym relationship details (optional)
	var userGym UserGym
	userGymQuery := `SELECT id, user_id, gym_id, TO_CHAR(crated_on, 'YYYY-MM-DD HH24:MI:SS') as created_on 
                     FROM user_gyms 
                     WHERE user_id = $1 AND gym_id = $2 
                     ORDER BY id DESC 
                     LIMIT 1`

	err = app.DB.QueryRow(userGymQuery, req.UserID, req.GymID).Scan(
		&userGym.ID, &userGym.UserID, &userGym.GymID, &userGym.CreatedOn)

	if err != nil {
		// User was added to gym but couldn't fetch details
		sendSuccessResponse(w, "User added to gym successfully", map[string]interface{}{
			"status":  "OK",
			"user_id": req.UserID,
			"gym_id":  req.GymID,
		})
		return
	}

	sendSuccessResponse(w, "User added to gym successfully", userGym)
}

// Alternative implementation using path parameters instead of JSON body
func (app *App) addUserToGymByPath(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	userIDStr := vars["user_id"]
	gymIDStr := vars["gym_id"]

	userID, err := strconv.Atoi(userIDStr)
	if err != nil || userID <= 0 {
		sendErrorResponse(w, "Invalid user_id parameter", http.StatusBadRequest)
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
	query := "SELECT ADD_USER_TO_GYM($1, $2)"
	err = tx.QueryRow(query, userID, gymID).Scan(&result)
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

	sendSuccessResponse(w, "User added to gym successfully", map[string]interface{}{
		"status":  "OK",
		"user_id": userID,
		"gym_id":  gymID,
	})
}

// Add these structs to your existing code
type AddMembershipToGymRequest struct {
	MembershipID int `json:"membership_id"`
	GymID        int `json:"gym_id"`
}

type MembershipGym struct {
	ID           int    `json:"id"`
	MembershipID int    `json:"membership_id"`
	GymID        int    `json:"gym_id"`
	CreatedBy    int    `json:"created_by"`
	UpdatedBy    int    `json:"updated_by"`
	CreatedOn    string `json:"created_on"`
	UpdatedOn    string `json:"updated_on"`
}

func (app *App) addMembershipToGym(w http.ResponseWriter, r *http.Request) {
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

	var req AddMembershipToGymRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.MembershipID <= 0 {
		sendErrorResponse(w, "Valid membership_id is required", http.StatusBadRequest)
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
	query := "SELECT add_membership_to_gym($1, $2, $3)"
	err = tx.QueryRow(query, req.MembershipID, req.GymID, claims.UserID).Scan(&result)
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

	// Get the created membership-gym relationship details (optional)
	var membershipGym MembershipGym
	membershipGymQuery := `SELECT id, membership_id, gym_id, created_by, updated_by,
                          TO_CHAR(created_on, 'YYYY-MM-DD HH24:MI:SS') as created_on,
                          TO_CHAR(updated_on, 'YYYY-MM-DD HH24:MI:SS') as updated_on
                          FROM membership_gyms 
                          WHERE membership_id = $1 AND gym_id = $2 
                          ORDER BY id DESC 
                          LIMIT 1`

	err = app.DB.QueryRow(membershipGymQuery, req.MembershipID, req.GymID).Scan(
		&membershipGym.ID, &membershipGym.MembershipID, &membershipGym.GymID,
		&membershipGym.CreatedBy, &membershipGym.UpdatedBy,
		&membershipGym.CreatedOn, &membershipGym.UpdatedOn)

	if err != nil {
		// Membership was added to gym but couldn't fetch details
		sendSuccessResponse(w, "Membership added to gym successfully", map[string]interface{}{
			"status":        "OK",
			"membership_id": req.MembershipID,
			"gym_id":        req.GymID,
		})
		return
	}

	sendSuccessResponse(w, "Membership added to gym successfully", membershipGym)
}

// Alternative implementation using path parameters instead of JSON body
func (app *App) addMembershipToGymByPath(w http.ResponseWriter, r *http.Request) {
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

	membershipIDStr := vars["membership_id"]
	gymIDStr := vars["gym_id"]

	membershipID, err := strconv.Atoi(membershipIDStr)
	if err != nil || membershipID <= 0 {
		sendErrorResponse(w, "Invalid membership_id parameter", http.StatusBadRequest)
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
	query := "SELECT add_membership_to_gym($1, $2, $3)"
	err = tx.QueryRow(query, membershipID, gymID, claims.UserID).Scan(&result)
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

	sendSuccessResponse(w, "Membership added to gym successfully", map[string]interface{}{
		"status":        "OK",
		"membership_id": membershipID,
		"gym_id":        gymID,
	})
}

// Add these structs to your existing code
type AddMachineToGymRequest struct {
	MachineID int `json:"machine_id"`
	GymID     int `json:"gym_id"`
}

type GymMachine struct {
	ID        int    `json:"id"`
	GymID     int    `json:"gym_id"`
	MachineID int    `json:"machine_id"`
	CreatedBy int    `json:"created_by"`
	UpdatedBy int    `json:"updated_by"`
	CreatedOn string `json:"created_on"`
	UpdatedOn string `json:"updated_on"`
}

func (app *App) addMachineToGym(w http.ResponseWriter, r *http.Request) {
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

	var req AddMachineToGymRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.MachineID <= 0 {
		sendErrorResponse(w, "Valid machine_id is required", http.StatusBadRequest)
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
	query := "SELECT add_machine_to_gym($1, $2, $3)"
	err = tx.QueryRow(query, req.MachineID, req.GymID, claims.UserID).Scan(&result)
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

	// Get the created gym-machine relationship details (optional)
	var gymMachine GymMachine
	gymMachineQuery := `SELECT id, gym_id, machine_id, created_by, updated_by,
                        TO_CHAR(created_on, 'YYYY-MM-DD HH24:MI:SS') as created_on,
                        TO_CHAR(updated_on, 'YYYY-MM-DD HH24:MI:SS') as updated_on
                        FROM gym_machines 
                        WHERE gym_id = $1 AND machine_id = $2 
                        ORDER BY id DESC 
                        LIMIT 1`

	err = app.DB.QueryRow(gymMachineQuery, req.GymID, req.MachineID).Scan(
		&gymMachine.ID, &gymMachine.GymID, &gymMachine.MachineID,
		&gymMachine.CreatedBy, &gymMachine.UpdatedBy,
		&gymMachine.CreatedOn, &gymMachine.UpdatedOn)

	if err != nil {
		// Machine was added to gym but couldn't fetch details
		sendSuccessResponse(w, "Machine added to gym successfully", map[string]interface{}{
			"status":     "OK",
			"machine_id": req.MachineID,
			"gym_id":     req.GymID,
		})
		return
	}

	sendSuccessResponse(w, "Machine added to gym successfully", gymMachine)
}

// Alternative implementation using path parameters instead of JSON body
func (app *App) addMachineToGymByPath(w http.ResponseWriter, r *http.Request) {
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

	machineIDStr := vars["machine_id"]
	gymIDStr := vars["gym_id"]

	machineID, err := strconv.Atoi(machineIDStr)
	if err != nil || machineID <= 0 {
		sendErrorResponse(w, "Invalid machine_id parameter", http.StatusBadRequest)
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
	query := "SELECT add_machine_to_gym($1, $2, $3)"
	err = tx.QueryRow(query, machineID, gymID, claims.UserID).Scan(&result)
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

	sendSuccessResponse(w, "Machine added to gym successfully", map[string]interface{}{
		"status":     "OK",
		"machine_id": machineID,
		"gym_id":     gymID,
	})
}

type GymStats struct {
	ID              int `json:"id"`
	GymID           int `json:"gym_id"`
	CurrentPeople   int `json:"current_people"`
	CurrentCombined int `json:"current_combined"`
	MaxPeople       int `json:"max_people"`
	MaxReservations int `json:"max_reservations"`
}

// Additional helper function to get current gym occupancy
func (app *App) getGymStats(w http.ResponseWriter, r *http.Request) {
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
	gymIDStr := vars["gym_id"]

	gymID, err := strconv.Atoi(gymIDStr)
	if err != nil || gymID <= 0 {
		sendErrorResponse(w, "Invalid gym_id parameter", http.StatusBadRequest)
		return
	}

	var gymStats GymStats
	gymStatsQuery := `SELECT id, gym_id, current_people, current_combined, max_people, max_reservations
                      FROM gym_stats 
                      WHERE gym_id = $1`

	err = app.DB.QueryRow(gymStatsQuery, gymID).Scan(
		&gymStats.ID, &gymStats.GymID, &gymStats.CurrentPeople,
		&gymStats.CurrentCombined, &gymStats.MaxPeople, &gymStats.MaxReservations)

	if err != nil {
		sendErrorResponse(w, "Gym not found or stats unavailable", http.StatusNotFound)
		return
	}

	sendSuccessResponse(w, "Gym stats retrieved successfully", gymStats)
}
