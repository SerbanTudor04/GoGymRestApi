package server

import (
	"database/sql"
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"
)

type User struct {
	ID             int    `json:"id"`
	FullName       string `json:"full_name"`
	Username       string `json:"username"`
	PasswordHashed string `json:"password_hashed,omitempty"` // omitempty for security
	CIF            int    `json:"cif"`
	Email          string `json:"email"`
	CreatedOn      string `json:"created_on"`
	UpdatedOn      string `json:"updated_on"`
}

func (app *App) getMe(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from JWT token
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

	var user User
	query := `SELECT id, full_name, username, cif, email, 
				  TO_CHAR(created_on, 'YYYY-MM-DD') as created_on, 
				  TO_CHAR(updated_on, 'YYYY-MM-DD') as updated_on 
				  FROM users WHERE id=$1`
	err = app.DB.QueryRow(query, claims.UserID).Scan(
		&user.ID, &user.FullName, &user.Username,
		&user.CIF, &user.Email, &user.CreatedOn, &user.UpdatedOn)
	if err != nil {
		if err == sql.ErrNoRows {
			sendErrorResponse(w, "User not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to fetch user", http.StatusInternalServerError)
		}
		return
	}

	sendSuccessResponse(w, "User retrieved successfully", user)
}

type RegisterRequest struct {
	FullName string `json:"full_name"`
	Username string `json:"username"`
	Password string `json:"password"`
	CIF      int    `json:"cif"`
	Email    string `json:"email"`
}

func (app *App) registerUser(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Username == "" {
		sendErrorResponse(w, "Username is required", http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		sendErrorResponse(w, "Password is required", http.StatusBadRequest)
		return
	}
	if req.Email == "" {
		sendErrorResponse(w, "Email is required", http.StatusBadRequest)
		return
	}
	if req.FullName == "" {
		sendErrorResponse(w, "Full name is required", http.StatusBadRequest)
		return
	}
	if req.CIF <= 0 {
		sendErrorResponse(w, "Valid CIF is required", http.StatusBadRequest)
		return
	}

	// Additional validation
	if len(req.Password) < 6 {
		sendErrorResponse(w, "Password must be at least 6 characters long", http.StatusBadRequest)
		return
	}

	// Hash password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		sendErrorResponse(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Call PostgreSQL REGISTER_USER function
	var result string
	query := "SELECT REGISTER_USER($1, $2, $3, $4, $5)"
	tx, err := app.DB.Begin()
	if err != nil {
		sendErrorResponse(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	err = tx.QueryRow(query, req.Username, string(hashedPassword), req.Email, req.FullName, req.CIF).Scan(&result)
	if err != nil {
		sendErrorResponse(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		err = tx.Rollback()
		if err != nil {
			sendErrorResponse(w, "Failed to rollback transaction", http.StatusInternalServerError)
		}
		return
	}
	err = tx.Commit()
	if err != nil {
		sendErrorResponse(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// Check if registration was successful
	if result != "OK" {
		sendErrorResponse(w, result, http.StatusBadRequest)
		return
	}

	// Get the created user details (without password hash)
	var user User
	userQuery := `SELECT id, full_name, username, cif, email, 
				  TO_CHAR(created_on, 'YYYY-MM-DD') as created_on, 
				  TO_CHAR(updated_on, 'YYYY-MM-DD') as updated_on 
				  FROM users WHERE UPPER(username) = UPPER($1)`

	err = app.DB.QueryRow(userQuery, req.Username).Scan(
		&user.ID, &user.FullName, &user.Username, &user.CIF,
		&user.Email, &user.CreatedOn, &user.UpdatedOn)

	if err != nil {
		// Registration succeeded but couldn't fetch user details
		sendSuccessResponse(w, "User registered successfully", map[string]string{
			"status":   "OK",
			"username": req.Username,
		})
		return
	}

	sendSuccessResponse(w, "User registered successfully", user)
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

const JWT_SECRET = "your-secret-key-change-this-in-production"

func (app *App) loginUser(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Username == "" {
		sendErrorResponse(w, "Username is required", http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		sendErrorResponse(w, "Password is required", http.StatusBadRequest)
		return
	}

	// Get user from database with password hash
	var user User
	var passwordHash string
	query := `SELECT id, full_name, username, password_hashed, cif, email, 
              TO_CHAR(created_on, 'YYYY-MM-DD') as created_on, 
              TO_CHAR(updated_on, 'YYYY-MM-DD') as updated_on 
              FROM users WHERE UPPER(username) = UPPER($1)`

	err := app.DB.QueryRow(query, req.Username).Scan(
		&user.ID, &user.FullName, &user.Username, &passwordHash,
		&user.CIF, &user.Email, &user.CreatedOn, &user.UpdatedOn)

	if err != nil {
		if err == sql.ErrNoRows {
			sendErrorResponse(w, "Invalid username or password", http.StatusUnauthorized)
		} else {
			sendErrorResponse(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password))
	if err != nil {
		sendErrorResponse(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	token, err := generateJWTToken(user.ID, user.Username)
	if err != nil {
		sendErrorResponse(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Prepare response (don't include password hash)
	user.PasswordHashed = ""
	response := LoginResponse{
		Token: token,
		User:  user,
	}

	sendSuccessResponse(w, "Login successful", response)
}

func generateJWTToken(userID int, username string) (string, error) {
	// Create claims
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // Token expires in 24 hours
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "GoGym",
			Subject:   username,
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret
	tokenString, err := token.SignedString([]byte(JWT_SECRET))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
