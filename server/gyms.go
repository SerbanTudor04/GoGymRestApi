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

// Update Gym Request struct
type UpdateGymRequest struct {
	Name            string `json:"name,omitempty"`
	Address         string `json:"address,omitempty"`
	Phone           string `json:"phone,omitempty"`
	Email           string `json:"email,omitempty"`
	MaxPeople       int    `json:"max_people,omitempty"`
	MaxReservations int    `json:"max_reservations,omitempty"`
}

// Update Gym function
func (app *App) updateGym(w http.ResponseWriter, r *http.Request) {
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

	// Get gym ID from URL
	vars := mux.Vars(r)
	gymIDStr := vars["gym_id"]
	gymID, err := strconv.Atoi(gymIDStr)
	if err != nil || gymID <= 0 {
		sendErrorResponse(w, "Invalid gym_id parameter", http.StatusBadRequest)
		return
	}

	var req UpdateGymRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Check if user has permission to update this gym
	var userRole string
	roleQuery := `SELECT role FROM user_gyms WHERE user_id = $1 AND gym_id = $2`
	err = app.DB.QueryRow(roleQuery, claims.UserID, gymID).Scan(&userRole)
	if err != nil {
		sendErrorResponse(w, "Gym not found or access denied", http.StatusForbidden)
		return
	}

	if userRole != "admin" {
		sendErrorResponse(w, "Insufficient permissions. Admin role required", http.StatusForbidden)
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

	// Update gym basic information
	updateFields := make([]string, 0)
	args := make([]interface{}, 0)
	argIndex := 1

	if req.Name != "" {
		updateFields = append(updateFields, "name = $"+strconv.Itoa(argIndex))
		args = append(args, req.Name)
		argIndex++
	}
	if req.Address != "" {
		updateFields = append(updateFields, "address = $"+strconv.Itoa(argIndex))
		args = append(args, req.Address)
		argIndex++
	}
	if req.Phone != "" {
		updateFields = append(updateFields, "phone = $"+strconv.Itoa(argIndex))
		args = append(args, req.Phone)
		argIndex++
	}
	if req.Email != "" {
		updateFields = append(updateFields, "email = $"+strconv.Itoa(argIndex))
		args = append(args, req.Email)
		argIndex++
	}

	// Always update updated_by and updated_on
	updateFields = append(updateFields, "updated_by = $"+strconv.Itoa(argIndex))
	args = append(args, claims.UserID)
	argIndex++

	// Add gym_id as the last parameter for WHERE clause
	args = append(args, gymID)

	if len(updateFields) > 1 { // More than just updated_by
		gymUpdateQuery := `UPDATE gyms SET ` +
			updateFields[0]
		for i := 1; i < len(updateFields); i++ {
			gymUpdateQuery += ", " + updateFields[i]
		}
		gymUpdateQuery += " WHERE id = $" + strconv.Itoa(argIndex)

		_, err = tx.Exec(gymUpdateQuery, args...)
		if err != nil {
			sendErrorResponse(w, "Failed to update gym: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Update gym stats if provided
	if req.MaxPeople > 0 || req.MaxReservations > 0 {
		statsUpdateFields := make([]string, 0)
		statsArgs := make([]interface{}, 0)
		statsArgIndex := 1

		if req.MaxPeople > 0 {
			statsUpdateFields = append(statsUpdateFields, "max_people = $"+strconv.Itoa(statsArgIndex))
			statsArgs = append(statsArgs, req.MaxPeople)
			statsArgIndex++
		}
		if req.MaxReservations > 0 {
			statsUpdateFields = append(statsUpdateFields, "max_resevations = $"+strconv.Itoa(statsArgIndex))
			statsArgs = append(statsArgs, req.MaxReservations)
			statsArgIndex++
		}

		// Add gym_id for WHERE clause
		statsArgs = append(statsArgs, gymID)

		statsUpdateQuery := `UPDATE gym_stats SET ` +
			statsUpdateFields[0]
		for i := 1; i < len(statsUpdateFields); i++ {
			statsUpdateQuery += ", " + statsUpdateFields[i]
		}
		statsUpdateQuery += " WHERE gym_id = $" + strconv.Itoa(statsArgIndex)

		_, err = tx.Exec(statsUpdateQuery, statsArgs...)
		if err != nil {
			sendErrorResponse(w, "Failed to update gym stats: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		sendErrorResponse(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// Get updated gym details
	var gym Gym
	gymQuery := `SELECT g.id, g.name, g.members, gs.max_people, gs.max_resevations 
                 FROM gyms g 
                 JOIN gym_stats gs ON g.id = gs.gym_id 
                 WHERE g.id = $1`

	err = app.DB.QueryRow(gymQuery, gymID).Scan(
		&gym.ID, &gym.Name, &gym.Members, &gym.MaxPeople, &gym.MaxReservations)

	if err != nil {
		sendSuccessResponse(w, "Gym updated successfully", map[string]interface{}{
			"status": "OK",
			"gym_id": gymID,
		})
		return
	}

	sendSuccessResponse(w, "Gym updated successfully", gym)
}

// Delete Gym function
func (app *App) deleteGym(w http.ResponseWriter, r *http.Request) {
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

	// Get gym ID from URL
	vars := mux.Vars(r)
	gymIDStr := vars["gym_id"]
	gymID, err := strconv.Atoi(gymIDStr)
	if err != nil || gymID <= 0 {
		sendErrorResponse(w, "Invalid gym_id parameter", http.StatusBadRequest)
		return
	}

	// Check if user has permission to delete this gym
	var userRole string
	var gymName string
	roleQuery := `SELECT ug.role, g.name 
	              FROM user_gyms ug 
	              JOIN gyms g ON ug.gym_id = g.id 
	              WHERE ug.user_id = $1 AND ug.gym_id = $2`
	err = app.DB.QueryRow(roleQuery, claims.UserID, gymID).Scan(&userRole, &gymName)
	if err != nil {
		sendErrorResponse(w, "Gym not found or access denied", http.StatusForbidden)
		return
	}

	if userRole != "admin" {
		sendErrorResponse(w, "Insufficient permissions. Admin role required", http.StatusForbidden)
		return
	}

	// Check if gym has active client memberships
	var activeMemberships int
	membershipQuery := `SELECT COUNT(*) FROM client_memberships cm
	                   JOIN membership_gyms mg ON cm.membership_id = mg.membership_id
	                   WHERE mg.gym_id = $1 AND cm.status = 'active' 
	                   AND CURRENT_DATE BETWEEN cm.starting_from AND cm.ending_on`
	err = app.DB.QueryRow(membershipQuery, gymID).Scan(&activeMemberships)
	if err == nil && activeMemberships > 0 {
		sendErrorResponse(w, "Cannot delete gym with active client memberships", http.StatusConflict)
		return
	}

	// Check if gym has people currently checked in
	var currentPeople int
	occupancyQuery := `SELECT current_people FROM gym_stats WHERE gym_id = $1`
	err = app.DB.QueryRow(occupancyQuery, gymID).Scan(&currentPeople)
	if err == nil && currentPeople > 0 {
		sendErrorResponse(w, "Cannot delete gym with people currently checked in", http.StatusConflict)
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

	// Delete gym stats
	_, err = tx.Exec("DELETE FROM gym_stats WHERE gym_id = $1", gymID)
	if err != nil {
		sendErrorResponse(w, "Failed to delete gym stats: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete user-gym relationships
	_, err = tx.Exec("DELETE FROM user_gyms WHERE gym_id = $1", gymID)
	if err != nil {
		sendErrorResponse(w, "Failed to delete user-gym relationships: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete membership-gym relationships
	_, err = tx.Exec("DELETE FROM membership_gyms WHERE gym_id = $1", gymID)
	if err != nil {
		sendErrorResponse(w, "Failed to delete membership-gym relationships: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete gym-machine relationships
	_, err = tx.Exec("DELETE FROM gym_machines WHERE gym_id = $1", gymID)
	if err != nil {
		sendErrorResponse(w, "Failed to delete gym-machine relationships: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete client passes
	_, err = tx.Exec("DELETE FROM client_passes WHERE gym_id = $1", gymID)
	if err != nil {
		sendErrorResponse(w, "Failed to delete client passes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Finally, delete the gym itself
	result, err := tx.Exec("DELETE FROM gyms WHERE id = $1", gymID)
	if err != nil {
		sendErrorResponse(w, "Failed to delete gym: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if gym was actually deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		sendErrorResponse(w, "Gym not found", http.StatusNotFound)
		return
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		sendErrorResponse(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	sendSuccessResponse(w, "Gym deleted successfully", map[string]interface{}{
		"status":   "OK",
		"gym_id":   gymID,
		"gym_name": gymName,
		"message":  "Gym and all related data have been permanently deleted",
	})
}

// Remove User from Gym
func (app *App) removeUserFromGym(w http.ResponseWriter, r *http.Request) {
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
	gymIDStr := vars["gym_id"]
	userIDStr := vars["user_id"]

	gymID, err := strconv.Atoi(gymIDStr)
	if err != nil || gymID <= 0 {
		sendErrorResponse(w, "Invalid gym_id parameter", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil || userID <= 0 {
		sendErrorResponse(w, "Invalid user_id parameter", http.StatusBadRequest)
		return
	}

	// Check if requesting user has permission (admin or removing themselves)
	var requestingUserRole string
	roleQuery := `SELECT role FROM user_gyms WHERE user_id = $1 AND gym_id = $2`
	err = app.DB.QueryRow(roleQuery, claims.UserID, gymID).Scan(&requestingUserRole)
	if err != nil {
		sendErrorResponse(w, "Gym not found or access denied", http.StatusForbidden)
		return
	}

	// Allow if user is admin or removing themselves
	if requestingUserRole != "admin" && claims.UserID != userID {
		sendErrorResponse(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	// Check if target user exists in gym
	var targetUserRole string
	var targetUserName string
	targetQuery := `SELECT ug.role, u.username 
	               FROM user_gyms ug 
	               JOIN users u ON ug.user_id = u.id 
	               WHERE ug.user_id = $1 AND ug.gym_id = $2`
	err = app.DB.QueryRow(targetQuery, userID, gymID).Scan(&targetUserRole, &targetUserName)
	if err != nil {
		sendErrorResponse(w, "User not found in this gym", http.StatusNotFound)
		return
	}

	// Prevent removing the last admin
	if targetUserRole == "admin" {
		var adminCount int
		adminCountQuery := `SELECT COUNT(*) FROM user_gyms WHERE gym_id = $1 AND role = 'admin'`
		err = app.DB.QueryRow(adminCountQuery, gymID).Scan(&adminCount)
		if err == nil && adminCount <= 1 {
			sendErrorResponse(w, "Cannot remove the last admin from the gym", http.StatusConflict)
			return
		}
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

	// Remove user from gym
	result, err := tx.Exec("DELETE FROM user_gyms WHERE user_id = $1 AND gym_id = $2", userID, gymID)
	if err != nil {
		sendErrorResponse(w, "Failed to remove user from gym: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		sendErrorResponse(w, "User-gym relationship not found", http.StatusNotFound)
		return
	}

	// Update gym members count
	_, err = tx.Exec("UPDATE gyms SET members = members - 1 WHERE id = $1 AND members > 0", gymID)
	if err != nil {
		sendErrorResponse(w, "Failed to update gym members count: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		sendErrorResponse(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	sendSuccessResponse(w, "User removed from gym successfully", map[string]interface{}{
		"status":    "OK",
		"gym_id":    gymID,
		"user_id":   userID,
		"username":  targetUserName,
		"user_role": targetUserRole,
	})
}

// Remove Membership from Gym
func (app *App) removeMembershipFromGym(w http.ResponseWriter, r *http.Request) {
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
	gymIDStr := vars["gym_id"]
	membershipIDStr := vars["membership_id"]

	gymID, err := strconv.Atoi(gymIDStr)
	if err != nil || gymID <= 0 {
		sendErrorResponse(w, "Invalid gym_id parameter", http.StatusBadRequest)
		return
	}

	membershipID, err := strconv.Atoi(membershipIDStr)
	if err != nil || membershipID <= 0 {
		sendErrorResponse(w, "Invalid membership_id parameter", http.StatusBadRequest)
		return
	}

	// Check if user has admin permission for this gym
	var userRole string
	roleQuery := `SELECT role FROM user_gyms WHERE user_id = $1 AND gym_id = $2`
	err = app.DB.QueryRow(roleQuery, claims.UserID, gymID).Scan(&userRole)
	if err != nil {
		sendErrorResponse(w, "Gym not found or access denied", http.StatusForbidden)
		return
	}

	if userRole != "admin" {
		sendErrorResponse(w, "Insufficient permissions. Admin role required", http.StatusForbidden)
		return
	}

	// Check if there are active client memberships using this membership type
	var activeClientMemberships int
	clientMembershipQuery := `SELECT COUNT(*) FROM client_memberships 
	                         WHERE membership_id = $1 AND status = 'active' 
	                         AND CURRENT_DATE BETWEEN starting_from AND ending_on`
	err = app.DB.QueryRow(clientMembershipQuery, membershipID).Scan(&activeClientMemberships)
	if err == nil && activeClientMemberships > 0 {
		sendErrorResponse(w, "Cannot remove membership type with active client memberships", http.StatusConflict)
		return
	}

	// Remove membership from gym
	result, err := app.DB.Exec("DELETE FROM membership_gyms WHERE membership_id = $1 AND gym_id = $2",
		membershipID, gymID)
	if err != nil {
		sendErrorResponse(w, "Failed to remove membership from gym: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		sendErrorResponse(w, "Membership-gym relationship not found", http.StatusNotFound)
		return
	}

	sendSuccessResponse(w, "Membership removed from gym successfully", map[string]interface{}{
		"status":        "OK",
		"gym_id":        gymID,
		"membership_id": membershipID,
	})
}

// Remove Machine from Gym
func (app *App) removeMachineFromGym(w http.ResponseWriter, r *http.Request) {
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
	gymIDStr := vars["gym_id"]
	machineIDStr := vars["machine_id"]

	gymID, err := strconv.Atoi(gymIDStr)
	if err != nil || gymID <= 0 {
		sendErrorResponse(w, "Invalid gym_id parameter", http.StatusBadRequest)
		return
	}

	machineID, err := strconv.Atoi(machineIDStr)
	if err != nil || machineID <= 0 {
		sendErrorResponse(w, "Invalid machine_id parameter", http.StatusBadRequest)
		return
	}

	// Check if user has admin permission for this gym
	var userRole string
	roleQuery := `SELECT role FROM user_gyms WHERE user_id = $1 AND gym_id = $2`
	err = app.DB.QueryRow(roleQuery, claims.UserID, gymID).Scan(&userRole)
	if err != nil {
		sendErrorResponse(w, "Gym not found or access denied", http.StatusForbidden)
		return
	}

	if userRole != "admin" {
		sendErrorResponse(w, "Insufficient permissions. Admin role required", http.StatusForbidden)
		return
	}

	// Remove machine from gym
	result, err := app.DB.Exec("DELETE FROM gym_machines WHERE machine_id = $1 AND gym_id = $2",
		machineID, gymID)
	if err != nil {
		sendErrorResponse(w, "Failed to remove machine from gym: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		sendErrorResponse(w, "Machine-gym relationship not found", http.StatusNotFound)
		return
	}

	sendSuccessResponse(w, "Machine removed from gym successfully", map[string]interface{}{
		"status":     "OK",
		"gym_id":     gymID,
		"machine_id": machineID,
	})
}
