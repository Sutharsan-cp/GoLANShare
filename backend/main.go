package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var config Config
var frontendDir string

type Config struct {
	Port          string `json:"port"`
	Host          string `json:"host"`
	EnableTLS     bool   `json:"enable_tls"`
	TLSCert       string `json:"tls_cert"`
	TLSKey        string `json:"tls_key"`
	RequireAuth   bool   `json:"require_auth"`
	StoragePath   string `json:"storage_path"`
	UploadPath    string `json:"upload_path"`
	MaxUploadSize int64  `json:"max_upload_size"`
}

func main() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("Error getting working directory:", err)
	}
	
	// Look for frontend in multiple possible locations
	frontendPaths := []string{
		filepath.Join(wd, "frontend"),
		filepath.Join(wd, "../frontend"),
		filepath.Join(wd, "./frontend"),
		wd, // current directory
	}
	
	for _, path := range frontendPaths {
		if _, err := os.Stat(path); err == nil {
			frontendDir = path
			break
		}
	}
	
	if frontendDir == "" {
		log.Printf("Warning: Frontend directory not found, using current directory")
		frontendDir = wd
	}

	loadConfig()

	// Create directories
	if err := os.MkdirAll(config.StoragePath, 0755); err != nil {
		log.Fatal("Error creating storage path:", err)
	}
	if err := os.MkdirAll(config.UploadPath, 0755); err != nil {
		log.Fatal("Error creating upload path:", err)
	}
	if err := os.MkdirAll(filepath.Join(config.StoragePath, "chunks"), 0755); err != nil {
		log.Fatal("Error creating chunks path:", err)
	}

	initDB()

	setupRoutes()

	// Start background services
	go startDiscovery()
	go hub.run()
	go startExpiryCron()

	// Server setup
	addr := fmt.Sprintf("%s:%s", config.Host, config.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      nil,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("GoLANshare server running on http://%s", addr)
	log.Printf("Frontend directory: %s", frontendDir)

	// Start server
	go func() {
		if config.EnableTLS {
			if err := server.ListenAndServeTLS(config.TLSCert, config.TLSKey); err != nil && err != http.ErrServerClosed {
				log.Fatal("Server error:", err)
			}
		} else {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatal("Server error:", err)
			}
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	log.Println("Server exiting")
}

func loadConfig() {
	// Try multiple possible config locations
	configPaths := []string{
		"config.json",
		"./config.json",
		"../config.json",
		"./../config.json",
	}
	
	var configFile []byte
	var err error
	
	for _, path := range configPaths {
		configFile, err = ioutil.ReadFile(path)
		if err == nil {
			log.Printf("Loaded config from: %s", path)
			break
		}
	}
	
	if err != nil {
		log.Printf("Warning: config.json not found, using defaults: %v", err)
		config = Config{
			Port:          "8081",
			Host:          "0.0.0.0",
			EnableTLS:     false,
			TLSCert:       "cert.pem",
			TLSKey:        "key.pem",
			RequireAuth:   false, // Default to false for easier testing
			StoragePath:   "./storage",
			UploadPath:    "./storage/uploads",
			MaxUploadSize: 1024 * 1024 * 1024, // 1GB
		}
		saveConfig()
		return
	}
	
	if err := json.Unmarshal(configFile, &config); err != nil {
		log.Printf("Error unmarshaling config, using defaults: %v", err)
		config = Config{
			Port:          "8081",
			Host:          "0.0.0.0",
			EnableTLS:     false,
			RequireAuth:   false,
			StoragePath:   "./storage",
			UploadPath:    "./storage/uploads",
			MaxUploadSize: 1024 * 1024 * 1024,
		}
	}
}

func saveConfig() {
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Printf("Error marshaling config: %v", err)
		return
	}
	
	if err := ioutil.WriteFile("config.json", configData, 0644); err != nil {
		log.Printf("Error saving config: %v", err)
	}
}

func setupRoutes() {
    // Auth routes
    http.HandleFunc("/api/auth", authHandler)
    http.HandleFunc("/api/auth/login", loginHandler)
    http.HandleFunc("/api/auth/register", registerHandler)
    http.HandleFunc("/api/auth/verify", verifyHandler)
    http.HandleFunc("/api/auth/logout", logoutHandler)
    http.HandleFunc("/api/auth/status", authStatusHandler)
    http.HandleFunc("/api/auth/profile", authMiddleware(profileHandler))
    http.HandleFunc("/api/auth/change-password", authMiddleware(changePasswordHandler))
    
    // File operations
    http.HandleFunc("/api/upload", authMiddleware(uploadHandler))
    http.HandleFunc("/api/upload/chunked", authMiddleware(handleChunkedUpload))
    http.HandleFunc("/api/download/", authMiddleware(downloadHandler))
    http.HandleFunc("/api/transfer/resume", authMiddleware(resumeTransfer))
    
    // Data endpoints
    http.HandleFunc("/api/devices", devicesHandler)
    http.HandleFunc("/api/transfers", transfersHandler)
    http.HandleFunc("/api/chat", chatHandler)
    http.HandleFunc("/api/settings", settingsHandler)
    
    // WebSocket
    http.HandleFunc("/ws", websocketHandler)
    
    // Static files
    http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(config.UploadPath))))
    http.HandleFunc("/", serveFrontend)
}

func serveFrontend(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Don't handle API routes and WebSocket here
	if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/ws") || strings.HasPrefix(r.URL.Path, "/files/") {
		http.NotFound(w, r)
		return
	}
	
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}
	
	fullPath := filepath.Join(frontendDir, path)
	
	// If file doesn't exist, serve index.html (for SPA routing)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		fullPath = filepath.Join(frontendDir, "index.html")
	}
	
	// Set appropriate content type
	ext := filepath.Ext(fullPath)
	switch ext {
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
	case ".html":
		w.Header().Set("Content-Type", "text/html")
	case ".json":
		w.Header().Set("Content-Type", "application/json")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	
	http.ServeFile(w, r, fullPath)
}

func enableCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}