package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte(getJWTSecret())

func getJWTSecret() string {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		return secret
	}
	// Fallback for development
	return "golanshare-development-secret-key-change-in-production"
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token   string `json:"token"`
	Status  string `json:"status"`
	Message string `json:"message"`
	User    string `json:"user,omitempty"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email,omitempty"`
}

// Enable CORS for requests

// Login handler with demo mode support
func loginHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var authReq AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&authReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if authReq.Username == "" || authReq.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Demo mode: accept any credentials when auth is disabled
	if !config.RequireAuth {
		token, err := generateJWT(authReq.Username)
		if err != nil {
			http.Error(w, "Error generating token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuthResponse{
			Token:   token,
			Status:  "success",
			Message: "Logged in successfully (Demo Mode)",
			User:    authReq.Username,
		})
		return
	}

	// Normal authentication flow
	hashedPass := getUserPassword(authReq.Username)
	if hashedPass == "" {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPass), []byte(authReq.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := generateJWT(authReq.Username)
	if err != nil {
		http.Error(w, "Error generating token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		Token:   token,
		Status:  "success",
		Message: "Logged in successfully",
		User:    authReq.Username,
	})
}

// Register handler for new users
func registerHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var regReq RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&regReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if regReq.Username == "" || regReq.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	if len(regReq.Username) < 3 {
		http.Error(w, "Username must be at least 3 characters", http.StatusBadRequest)
		return
	}

	if len(regReq.Password) < 6 {
		http.Error(w, "Password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	// Check if user already exists
	existingPass := getUserPassword(regReq.Username)
	if existingPass != "" {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(regReq.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error creating user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Save user to database
	if err := execDB("INSERT INTO users (username, password) VALUES (?, ?)", regReq.Username, string(hashedPassword)); err != nil {
		http.Error(w, "Error saving user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate token for immediate login
	token, err := generateJWT(regReq.Username)
	if err != nil {
		http.Error(w, "Error generating token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(AuthResponse{
		Token:   token,
		Status:  "success",
		Message: "User registered successfully",
		User:    regReq.Username,
	})
}

// Verify token validity
func verifyHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tokenString := extractTokenFromHeader(r)
	if tokenString == "" {
		http.Error(w, "Authorization token required", http.StatusUnauthorized)
		return
	}

	claims, err := verifyToken(tokenString)
	if err != nil {
		http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"message":  "Token is valid",
		"user":     claims.Username,
		"valid":    true,
	})
}

// Logout handler (mainly for client-side token removal)
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Logged out successfully",
	})
}

// Generate JWT token
func generateJWT(username string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Subject:   username,
			ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// Verify JWT token
func verifyToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token parsing error: %v", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Check if token is expired
	if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
		return nil, fmt.Errorf("token expired")
	}

	return claims, nil
}

// Extract token from Authorization header
func extractTokenFromHeader(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	// Handle "Bearer <token>" format
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return authHeader
}

// Authentication middleware
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(&w)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Skip authentication if not required
		if !config.RequireAuth {
			ctx := context.WithValue(r.Context(), "username", "demo-user")
			next(w, r.WithContext(ctx))
			return
		}

		tokenString := extractTokenFromHeader(r)
		if tokenString == "" {
			http.Error(w, "Authorization token required", http.StatusUnauthorized)
			return
		}

		claims, err := verifyToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// Add username to context for downstream handlers
		ctx := context.WithValue(r.Context(), "username", claims.Username)
		next(w, r.WithContext(ctx))
	}
}

// Get current user from context
func getCurrentUser(r *http.Request) string {
	if user, ok := r.Context().Value("username").(string); ok {
		return user
	}
	return "unknown"
}

// Change password handler
func changePasswordHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get current user from context
	currentUser := getCurrentUser(r)
	if currentUser == "unknown" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var changeReq struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&changeReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate new password
	if len(changeReq.NewPassword) < 6 {
		http.Error(w, "New password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	// Verify current password
	hashedPass := getUserPassword(currentUser)
	if hashedPass == "" {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPass), []byte(changeReq.CurrentPassword)); err != nil {
		http.Error(w, "Current password is incorrect", http.StatusUnauthorized)
		return
	}

	// Hash new password
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(changeReq.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error updating password: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Update password in database
	if err := execDB("UPDATE users SET password = ? WHERE username = ?", string(newHashedPassword), currentUser); err != nil {
		http.Error(w, "Error updating password: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Password updated successfully",
	})
}

// Get user profile
func profileHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	currentUser := getCurrentUser(r)
	if currentUser == "unknown" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Get user info (you can extend this with more user data)
	profile := map[string]interface{}{
		"username": currentUser,
		"joined_at": time.Now().Format("2006-01-02"),
		"role":     "user",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"profile": profile,
	})
}

// Get user password from database


// Database execution helper


// Auth status handler - returns whether auth is required
func authStatusHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"require_auth": config.RequireAuth,
		"demo_mode":    !config.RequireAuth,
		"message":      "Authentication status",
	})
}

// Simple auth handler for backward compatibility
func authHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Route to appropriate handler based on path
	path := r.URL.Path
	switch {
	case path == "/api/auth" && r.Method == "POST":
		loginHandler(w, r)
	case path == "/api/auth/register" && r.Method == "POST":
		registerHandler(w, r)
	case path == "/api/auth/verify" && r.Method == "GET":
		verifyHandler(w, r)
	case path == "/api/auth/logout" && r.Method == "POST":
		logoutHandler(w, r)
	case path == "/api/auth/status" && r.Method == "GET":
		authStatusHandler(w, r)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

