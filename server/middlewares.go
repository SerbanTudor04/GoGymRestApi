package server

import (
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.RequestURI, time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Middleware to validate JWT tokens
func (app *App) authenticateJWT(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			sendErrorResponse(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Expected format: "Bearer <token>"
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			sendErrorResponse(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := authHeader[7:]

		// Parse and validate token
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(JWT_SECRET), nil
		})

		if err != nil || !token.Valid {
			sendErrorResponse(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Add user info to request context if needed
		// You can use context.WithValue to store user info for use in handlers

		next.ServeHTTP(w, r)
	}
}

// Add this middleware function to your existing JWT code
func (app *App) authenticateJWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			sendErrorResponse(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Expected format: "Bearer <token>"
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			sendErrorResponse(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := authHeader[7:]

		// Parse and validate token
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(JWT_SECRET), nil
		})

		if err != nil || !token.Valid {
			sendErrorResponse(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Optionally, you can add user info to request context here
		// ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		// r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// Rate limiting middleware with per-IP limiting
func (app *App) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)

		app.mu.Lock()
		limiter, exists := app.limiters[ip]
		if !exists {
			limiter = rate.NewLimiter(rate.Every(rateLimitWindow/rateLimitRequests), rateLimitBurst)
			app.limiters[ip] = limiter
		}
		app.mu.Unlock()

		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Security headers middleware
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add security headers
		for header, value := range securityHeaders {
			w.Header().Set(header, value)
		}

		// Remove server information
		w.Header().Del("Server")

		next.ServeHTTP(w, r)
	})
}

// Request timeout middleware
func timeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

// Enhanced CORS middleware with better security
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Define allowed origins (customize as needed)
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:8080",
			// Add your frontend domains here
		}

		// Check if origin is allowed
		isAllowed := false
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				isAllowed = true
				break
			}
		}

		if isAllowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Enhanced logging middleware with more details
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom response writer to capture status code
		wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapper, r)

		duration := time.Since(start)
		ip := getClientIP(r)

		log.Printf(
			"%s %s %s %d %v %s",
			ip,
			r.Method,
			r.RequestURI,
			wrapper.statusCode,
			duration,
			r.UserAgent(),
		)
	})
}

// Custom response writer to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Get client IP address considering proxies
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		if net.ParseIP(xri) != nil {
			return xri
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}

// Cleanup old rate limiters periodically (add this to your app initialization)
func (app *App) startCleanupRoutine() {
	ticker := time.NewTicker(time.Hour)
	go func() {
		for {
			select {
			case <-ticker.C:
				app.cleanupLimiters()
			}
		}
	}()
}

func (app *App) cleanupLimiters() {
	app.mu.Lock()
	defer app.mu.Unlock()

	// Remove limiters that haven't been used recently
	for ip, limiter := range app.limiters {
		// If the limiter has full tokens, it hasn't been used recently
		if limiter.Tokens() == float64(rateLimitBurst) {
			delete(app.limiters, ip)
		}
	}
}
