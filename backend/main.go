package main

import (
        "context"
        "crypto/tls"
        "encoding/json"
        "fmt"
        "io/ioutil"
        "log"
        "net"
        "net/http"
        "os"
        "os/signal"
        "path/filepath"
        "runtime"
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
        // Setup logging
        setupLogging()
        
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

        // Server setup with production configurations
        addr := fmt.Sprintf("%s:%s", config.Host, config.Port)
        

        
        server := &http.Server{
                Addr:           addr,
                Handler:        http.DefaultServeMux,
                ReadTimeout:    60 * time.Second,
                WriteTimeout:   60 * time.Second,
                IdleTimeout:    120 * time.Second,
                MaxHeaderBytes: 1 << 20, // 1MB
        }
        
        // Configure TLS if enabled
        if config.EnableTLS {
                tlsConfig := &tls.Config{
                        MinVersion:               tls.VersionTLS12,
                        CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
                        PreferServerCipherSuites: true,
                        CipherSuites: []uint16{
                                tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
                                tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
                                tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
                        },
                }
                server.TLSConfig = tlsConfig
        }

        protocol := "http"
        if config.EnableTLS {
                protocol = "https"
        }
        
        log.Printf("GoLANshare server running on %s://%s", protocol, addr)
        log.Printf("Frontend directory: %s", frontendDir)
        log.Printf("Go version: %s", runtime.Version())
        log.Printf("OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
        
        // Get and display network interfaces
        displayNetworkInfo()

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
    http.HandleFunc("/api/files", filesHandler)
    http.HandleFunc("/api/files/delete", authMiddleware(fileDeleteHandler))
    
    // Private sharing endpoints
    http.HandleFunc("/api/share/send", authMiddleware(privateSendHandler))
    http.HandleFunc("/api/share/receive", authMiddleware(privateReceiveHandler))
    http.HandleFunc("/api/share/request", authMiddleware(shareRequestHandler))
    http.HandleFunc("/api/share/accept", authMiddleware(shareAcceptHandler))
    http.HandleFunc("/api/share/reject", authMiddleware(shareRejectHandler))
    http.HandleFunc("/api/share/history", authMiddleware(shareHistoryHandler))
    
    // Device interaction endpoints
    http.HandleFunc("/api/device/connect", authMiddleware(deviceConnectHandler))
    http.HandleFunc("/api/device/disconnect", authMiddleware(deviceDisconnectHandler))
    http.HandleFunc("/api/device/info", deviceInfoHandler)
    http.HandleFunc("/api/device/scan", authMiddleware(deviceScanHandler))
    http.HandleFunc("/api/device/add", authMiddleware(deviceAddHandler))
    http.HandleFunc("/api/device/test", deviceTestHandler)
    http.HandleFunc("/api/debug/clients", debugClientsHandler)
    
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
        
        // Set cache control headers to prevent stale content in iframe
        w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
        w.Header().Set("Pragma", "no-cache")
        w.Header().Set("Expires", "0")
        
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
        (*w).Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
        (*w).Header().Set("Access-Control-Allow-Credentials", "true")
        (*w).Header().Set("Access-Control-Max-Age", "86400")
}

func displayNetworkInfo() {
        interfaces, err := net.Interfaces()
        if err != nil {
                log.Printf("Error getting network interfaces: %v", err)
                return
        }
        
        log.Println("Available network interfaces:")
        for _, iface := range interfaces {
                if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
                        continue
                }
                
                addrs, err := iface.Addrs()
                if err != nil {
                        continue
                }
                
                for _, addr := range addrs {
                        var ip net.IP
                        switch v := addr.(type) {
                        case *net.IPNet:
                                ip = v.IP
                        case *net.IPAddr:
                                ip = v.IP
                        }
                        
                        if ip == nil || ip.IsLoopback() || ip.To4() == nil {
                                continue
                        }
                        
                        log.Printf("  %s: %s", iface.Name, ip.String())
                        if config.EnableTLS {
                                log.Printf("    Access via: https://%s:%s", ip.String(), config.Port)
                        } else {
                                log.Printf("    Access via: http://%s:%s", ip.String(), config.Port)
                        }
                }
        }
}

func setupLogging() {
        // Create logs directory
        os.MkdirAll("logs", 0755)
        
        // Set log format
        log.SetFlags(log.LstdFlags | log.Lshortfile)
        log.Println("Production GoLANshare - Logging initialized")
}