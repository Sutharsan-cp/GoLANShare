package main

import (
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type FileInfo struct {
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	Type        string    `json:"type"`
	Modified    time.Time `json:"modified"`
	URL         string    `json:"url"`
	Version     int       `json:"version"`
	Expiry      time.Time `json:"expiry"`
	Hash        string    `json:"hash"`
	MimeType    string    `json:"mime_type"`
	Downloads   int       `json:"downloads"`
	Thumbnail   string    `json:"thumbnail,omitempty"`
	IsEncrypted bool      `json:"is_encrypted"`
}

type Transfer struct {
	ID          string    `json:"id"`
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size"`
	Progress    int       `json:"progress"`
	Status      string    `json:"status"`
	Speed       float64   `json:"speed"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	FromDevice  string    `json:"from_device"`
	ToDevice    string    `json:"to_device"`
	Error       string    `json:"error,omitempty"`
	BytesTransferred int64 `json:"bytes_transferred"`
	ETA         int64     `json:"eta"`
	Type        string    `json:"type"` // upload, download, p2p
}

var transfers = make(map[string]*Transfer)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" {
		respondWithError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Rate limiting check
	clientIP := getClientIP(r)
	if !checkRateLimit(clientIP) {
		respondWithError(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	
	// Limit upload size
	r.Body = http.MaxBytesReader(w, r.Body, config.MaxUploadSize)
	if err := r.ParseMultipartForm(config.MaxUploadSize); err != nil {
		log.Printf("Upload error from %s: %v", clientIP, err)
		respondWithError(w, "File too large or invalid request", http.StatusRequestEntityTooLarge)
		return
	}
	
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		respondWithError(w, "No files provided", http.StatusBadRequest)
		return
	}
	
	// Check disk space
	if !checkDiskSpace() {
		respondWithError(w, "Insufficient disk space", http.StatusInsufficientStorage)
		return
	}
	
	results := make([]FileInfo, 0)
	errors := make([]string, 0)
	
	for _, fileHeader := range files {
		result, err := processUploadedFile(fileHeader, clientIP)
		if err != nil {
			log.Printf("Error processing file %s: %v", fileHeader.Filename, err)
			errors = append(errors, fmt.Sprintf("%s: %v", fileHeader.Filename, err))
			continue
		}
		results = append(results, *result)
	}
	
	response := map[string]interface{}{
		"files": results,
		"count": len(results),
	}
	
	if len(errors) > 0 {
		response["errors"] = errors
		response["partial_success"] = true
	}
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
	
	log.Printf("Upload completed: %d files from %s", len(results), clientIP)
}

func processUploadedFile(fileHeader *multipart.FileHeader, clientIP string) (*FileInfo, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()
	
	// Sanitize and validate filename
	safeName := sanitizeFilename(fileHeader.Filename)
	if safeName == "" {
		return nil, fmt.Errorf("invalid filename")
	}
	
	// Check for malicious files
	if isBlockedFile(safeName) {
		return nil, fmt.Errorf("file type not allowed")
	}
	
	// Generate unique filename if exists
	finalPath := getUniqueFilePath(safeName)
	finalName := filepath.Base(finalPath)
	
	// Create file with proper permissions
	dst, err := os.OpenFile(finalPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("error creating file: %v", err)
	}
	defer dst.Close()
	
	// Calculate hash while copying
	hash := md5.New()
	tee := io.TeeReader(file, hash)
	
	bytesWritten, err := io.Copy(dst, tee)
	if err != nil {
		os.Remove(finalPath) // Cleanup on error
		return nil, fmt.Errorf("error saving file: %v", err)
	}
	
	fileHash := hex.EncodeToString(hash.Sum(nil))
	mimeType := mime.TypeByExtension(filepath.Ext(finalName))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	
	fInfo := FileInfo{
		Name:        finalName,
		Size:        bytesWritten,
		Type:        getFileType(finalName),
		Modified:    time.Now(),
		URL:         "/files/" + finalName,
		Version:     1,
		Expiry:      time.Now().Add(24 * time.Hour),
		Hash:        fileHash,
		MimeType:    mimeType,
		Downloads:   0,
		IsEncrypted: false,
	}
	
	// Generate thumbnail for images
	if strings.HasPrefix(mimeType, "image/") {
		go generateThumbnail(finalPath, finalName)
	}
	
	if err := insertFileInfo(fInfo); err != nil {
		log.Printf("Error inserting file info: %v", err)
	}
	
	// Create transfer record
	transfer := &Transfer{
		ID:               generateID(),
		FileName:         finalName,
		FileSize:         bytesWritten,
		Progress:         100,
		Status:           "completed",
		StartedAt:        time.Now(),
		CompletedAt:      time.Now(),
		FromDevice:       clientIP,
		BytesTransferred: bytesWritten,
		Type:             "upload",
	}
	
	transfers[transfer.ID] = transfer
	if err := insertTransfer(*transfer); err != nil {
		log.Printf("Error inserting transfer: %v", err)
	}
	
	// Compress if enabled
	if getSetting("compress") == "true" {
		go compressFile(finalPath)
	}
	
	return &fInfo, nil
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "GET" && r.Method != "HEAD" {
		respondWithError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	path := strings.TrimPrefix(r.URL.Path, "/api/download/")
	if path == "" {
		respondWithError(w, "File name required", http.StatusBadRequest)
		return
	}
	
	// Enhanced security check
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") || strings.Contains(cleanPath, "/") || strings.Contains(cleanPath, "\\") {
		log.Printf("Security violation: attempted path traversal from %s: %s", getClientIP(r), path)
		respondWithError(w, "Invalid file path", http.StatusBadRequest)
		return
	}
	
	filePath := filepath.Join(config.UploadPath, cleanPath)
	
	// Verify file is within upload directory
	absUploadPath, _ := filepath.Abs(config.UploadPath)
	absFilePath, _ := filepath.Abs(filePath)
	if !strings.HasPrefix(absFilePath, absUploadPath) {
		log.Printf("Security violation: path traversal attempt from %s", getClientIP(r))
		respondWithError(w, "Access denied", http.StatusForbidden)
		return
	}
	
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		respondWithError(w, "File not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error stating file %s: %v", filePath, err)
		respondWithError(w, "Error accessing file", http.StatusInternalServerError)
		return
	}
	
	// Check if file has expired
	if isFileExpired(cleanPath) {
		log.Printf("Attempted download of expired file: %s", cleanPath)
		respondWithError(w, "File has expired", http.StatusGone)
		return
	}
	
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Error opening file %s: %v", filePath, err)
		respondWithError(w, "Error opening file", http.StatusInternalServerError)
		return
	}
	defer file.Close()
	
	// Set security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'none'")
	
	// Set content type
	mimeType := mime.TypeByExtension(filepath.Ext(cleanPath))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", mimeType)
	
	// Set cache headers
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("ETag", fmt.Sprintf("\"%d-%d\"", fileInfo.Size(), fileInfo.ModTime().Unix()))
	
	// Check if client has cached version
	if match := r.Header.Get("If-None-Match"); match != "" {
		if strings.Contains(match, fmt.Sprintf("%d-%d", fileInfo.Size(), fileInfo.ModTime().Unix())) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	
	// Support for range requests (resumable downloads)
	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		ranges := parseRange(rangeHeader, fileInfo.Size())
		if len(ranges) > 0 {
			start, end := ranges[0].start, ranges[0].end
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileInfo.Size()))
			w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
			w.WriteHeader(http.StatusPartialContent)
			
			if r.Method != "HEAD" {
				file.Seek(start, 0)
				io.CopyN(w, file, end-start+1)
			}
			
			// Log partial download
			log.Printf("Partial download: %s (bytes %d-%d) to %s", cleanPath, start, end, getClientIP(r))
			return
		}
	}
	
	// Regular download
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
	
	// Set filename for download
	disposition := "attachment"
	if r.URL.Query().Get("inline") == "true" {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=\"%s\"", disposition, filepath.Base(cleanPath)))
	
	if r.Method != "HEAD" {
		// Track download
		go incrementDownloadCount(cleanPath)
		
		// Create transfer record
		transfer := &Transfer{
			ID:               generateID(),
			FileName:         cleanPath,
			FileSize:         fileInfo.Size(),
			Progress:         0,
			Status:           "downloading",
			StartedAt:        time.Now(),
			ToDevice:         getClientIP(r),
			BytesTransferred: 0,
			Type:             "download",
		}
		transfers[transfer.ID] = transfer
		
		// Copy with progress tracking
		written, err := io.Copy(w, file)
		if err != nil {
			log.Printf("Error during download of %s: %v", cleanPath, err)
			transfer.Status = "failed"
			transfer.Error = err.Error()
		} else {
			transfer.Status = "completed"
			transfer.Progress = 100
			transfer.BytesTransferred = written
		}
		transfer.CompletedAt = time.Now()
		
		// Update transfer record
		if err := insertTransfer(*transfer); err != nil {
			log.Printf("Error inserting transfer record: %v", err)
		}
		
		log.Printf("Download completed: %s (%d bytes) to %s", cleanPath, written, getClientIP(r))
	}
}

type httpRange struct {
	start, end int64
}

func parseRange(rangeHeader string, size int64) []httpRange {
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return nil
	}
	
	rangeStr := strings.TrimPrefix(rangeHeader, "bytes=")
	ranges := strings.Split(rangeStr, ",")
	
	var result []httpRange
	for _, r := range ranges {
		r = strings.TrimSpace(r)
		parts := strings.Split(r, "-")
		if len(parts) != 2 {
			continue
		}
		
		var start, end int64
		var err error
		
		if parts[0] == "" {
			// Suffix range: -500
			if end, err = strconv.ParseInt(parts[1], 10, 64); err != nil {
				continue
			}
			start = size - end
			end = size - 1
		} else if parts[1] == "" {
			// Prefix range: 500-
			if start, err = strconv.ParseInt(parts[0], 10, 64); err != nil {
				continue
			}
			end = size - 1
		} else {
			// Full range: 500-999
			if start, err = strconv.ParseInt(parts[0], 10, 64); err != nil {
				continue
			}
			if end, err = strconv.ParseInt(parts[1], 10, 64); err != nil {
				continue
			}
		}
		
		// Validate range
		if start < 0 {
			start = 0
		}
		if end >= size {
			end = size - 1
		}
		if start <= end {
			result = append(result, httpRange{start, end})
		}
	}
	
	return result
}

func isFileExpired(filename string) bool {
	// Check if file has expired based on database record
	files := getAllFiles()
	for _, f := range files {
		if f.Name == filename {
			return time.Now().After(f.Expiry)
		}
	}
	return false
}

func incrementDownloadCount(filename string) {
	// Increment download counter in database
	if err := execDB("UPDATE files SET downloads = downloads + 1 WHERE name = ?", filename); err != nil {
		log.Printf("Error incrementing download count for %s: %v", filename, err)
	}
}

func devicesHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Get client IP for logging
	clientIP := getClientIP(r)
	
	// Return devices with debug logging
	devs := getDevices()
	log.Printf("Client %s requested devices. Found %d devices:", clientIP, len(devs))
	for i, dev := range devs {
		log.Printf("  Device %d: %s (%s) - %s [Self: %t]", i+1, dev.Name, dev.IP, dev.Status, dev.IsSelf)
	}
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(devs); err != nil {
		log.Printf("Error encoding devices: %v", err)
	}
}

func transfersHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	trans := getAllTransfers()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(trans); err != nil {
		log.Printf("Error encoding transfers: %v", err)
	}
}

func settingsHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method == "GET" {
		settings := getAllSettings()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(settings); err != nil {
			log.Printf("Error encoding settings: %v", err)
		}
		return
	}
	
	if r.Method == "POST" {
		var newSettings map[string]string
		if err := json.NewDecoder(r.Body).Decode(&newSettings); err != nil {
			http.Error(w, "Invalid configuration", http.StatusBadRequest)
			return
		}
		
		for k, v := range newSettings {
			if err := setSetting(k, v); err != nil {
				log.Printf("Error setting %s: %v", k, err)
			}
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		return
	}
	
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func getFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg":
		return "image"
	case ".mp4", ".avi", ".mov", ".wmv", ".webm", ".mkv", ".flv":
		return "video"
	case ".mp3", ".wav", ".ogg", ".flac", ".aac", ".m4a":
		return "audio"
	case ".pdf", ".doc", ".docx", ".txt", ".rtf", ".odt", ".xls", ".xlsx", ".ppt", ".pptx":
		return "document"
	case ".zip", ".rar", ".7z", ".tar", ".gz", ".bz2":
		return "archive"
	default:
		return "other"
	}
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func compressFile(path string) {
	gzPath := path + ".gz"
	f, err := os.Open(path)
	if err != nil {
		log.Printf("Error opening file for compression: %v", err)
		return
	}
	defer f.Close()
	
	gzF, err := os.Create(gzPath)
	if err != nil {
		log.Printf("Error creating gzip file: %v", err)
		return
	}
	defer gzF.Close()
	
	gz := gzip.NewWriter(gzF)
	if _, err := io.Copy(gz, f); err != nil {
		log.Printf("Error compressing file: %v", err)
		return
	}
	
	if err := gz.Close(); err != nil {
		log.Printf("Error closing gzip writer: %v", err)
		return
	}
	
	// Replace original with compressed version
	if err := os.Remove(path); err != nil {
		log.Printf("Error removing original file: %v", err)
		return
	}
	
	if err := os.Rename(gzPath, path); err != nil {
		log.Printf("Error renaming compressed file: %v", err)
		return
	}
	
	log.Printf("Compressed file: %s", path)
}

func filesHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	files := getAllFiles()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(files); err != nil {
		log.Printf("Error encoding files: %v", err)
	}
}

// Rate limiting map
var rateLimitMap = make(map[string][]time.Time)

func respondWithError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   true,
		"message": message,
		"code":    code,
	})
}

func checkRateLimit(clientIP string) bool {
	now := time.Now()
	windowStart := now.Add(-time.Minute) // 1 minute window
	
	// Clean old entries
	if requests, exists := rateLimitMap[clientIP]; exists {
		validRequests := make([]time.Time, 0)
		for _, reqTime := range requests {
			if reqTime.After(windowStart) {
				validRequests = append(validRequests, reqTime)
			}
		}
		rateLimitMap[clientIP] = validRequests
	}
	
	// Check limit (100 requests per minute)
	if len(rateLimitMap[clientIP]) >= 100 {
		return false
	}
	
	// Add current request
	rateLimitMap[clientIP] = append(rateLimitMap[clientIP], now)
	return true
}

func checkDiskSpace() bool {
	// Simple disk space check - in production, implement proper disk space monitoring
	return true // Placeholder
}

func sanitizeFilename(filename string) string {
	// Remove path separators and dangerous characters
	safeName := filepath.Base(filename)
	
	// Remove or replace dangerous characters
	reg := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	safeName = reg.ReplaceAllString(safeName, "_")
	
	// Limit length
	if len(safeName) > 255 {
		ext := filepath.Ext(safeName)
		name := safeName[:255-len(ext)]
		safeName = name + ext
	}
	
	// Ensure not empty
	if safeName == "" || safeName == "." || safeName == ".." {
		return ""
	}
	
	return safeName
}

func isBlockedFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	blockedExts := []string{".exe", ".bat", ".cmd", ".com", ".pif", ".scr", ".vbs", ".js"}
	
	for _, blocked := range blockedExts {
		if ext == blocked {
			return true
		}
	}
	return false
}

func getUniqueFilePath(filename string) string {
	basePath := filepath.Join(config.UploadPath, filename)
	
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return basePath
	}
	
	// File exists, generate unique name
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	
	for i := 1; i < 1000; i++ {
		newName := fmt.Sprintf("%s_%d%s", name, i, ext)
		newPath := filepath.Join(config.UploadPath, newName)
		
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
	}
	
	// Fallback with timestamp
	timestamp := time.Now().Unix()
	newName := fmt.Sprintf("%s_%d%s", name, timestamp, ext)
	return filepath.Join(config.UploadPath, newName)
}

func generateThumbnail(filePath, filename string) {
	// Placeholder for thumbnail generation
	// In production, implement image thumbnail generation
	log.Printf("Generating thumbnail for %s", filename)
}

func getClientIP(r *http.Request) string {
	// Get IP from X-Forwarded-For header if behind proxy
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}
	
	// Get IP from X-Real-IP header
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	
	// Fallback to remote address
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// Private Sharing Structures
type ShareRequest struct {
	ID          string    `json:"id"`
	FromDevice  string    `json:"from_device"`
	ToDevice    string    `json:"to_device"`
	FromName    string    `json:"from_name"`
	ToName      string    `json:"to_name"`
	Type        string    `json:"type"` // file, message, folder
	Content     string    `json:"content,omitempty"`
	Files       []string  `json:"files,omitempty"`
	Message     string    `json:"message,omitempty"`
	Status      string    `json:"status"` // pending, accepted, rejected, completed
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	Size        int64     `json:"size,omitempty"`
	Preview     string    `json:"preview,omitempty"`
}

var shareRequests = make(map[string]*ShareRequest)
var shareHistory = make([]ShareRequest, 0)

// Private send handler - initiate sharing to specific device
func privateSendHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" {
		respondWithError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var request struct {
		ToDeviceID string   `json:"to_device_id"`
		Type       string   `json:"type"`
		Files      []string `json:"files,omitempty"`
		Message    string   `json:"message,omitempty"`
		Content    string   `json:"content,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Find target device
	targetDevice := findDeviceByID(request.ToDeviceID)
	if targetDevice == nil {
		respondWithError(w, "Target device not found", http.StatusNotFound)
		return
	}
	
	// Create share request
	shareReq := &ShareRequest{
		ID:         generateID(),
		FromDevice: selfDevice.ID,
		ToDevice:   request.ToDeviceID,
		FromName:   selfDevice.Name,
		ToName:     targetDevice.Name,
		Type:       request.Type,
		Content:    request.Content,
		Files:      request.Files,
		Message:    request.Message,
		Status:     "pending",
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(5 * time.Minute), // 5 minute expiry
	}
	
	// Calculate size for files
	if request.Type == "file" && len(request.Files) > 0 {
		totalSize := int64(0)
		for _, fileName := range request.Files {
			filePath := filepath.Join(config.UploadPath, fileName)
			if stat, err := os.Stat(filePath); err == nil {
				totalSize += stat.Size()
			}
		}
		shareReq.Size = totalSize
	}
	
	shareRequests[shareReq.ID] = shareReq
	
	// Send notification to target device via WebSocket
	notification := map[string]interface{}{
		"type":         "share_request",
		"share_request": shareReq,
	}
	
	broadcastToDevice(targetDevice.ID, notification)
	
	// Log the share request
	log.Printf("Share request sent from %s to %s: %s", shareReq.FromName, shareReq.ToName, shareReq.Type)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"request_id":   shareReq.ID,
		"message":      "Share request sent successfully",
		"expires_in":   300, // 5 minutes
	})
}

// Private receive handler - get pending requests for current device
func privateReceiveHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "GET" {
		respondWithError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Get pending requests for this device
	pendingRequests := make([]*ShareRequest, 0)
	now := time.Now()
	
	for id, req := range shareRequests {
		// Remove expired requests
		if now.After(req.ExpiresAt) {
			delete(shareRequests, id)
			continue
		}
		
		// Check if request is for this device
		if req.ToDevice == selfDevice.ID && req.Status == "pending" {
			pendingRequests = append(pendingRequests, req)
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"requests": pendingRequests,
		"count":    len(pendingRequests),
	})
}

// Share request handler - get specific request details
func shareRequestHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "GET" {
		respondWithError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	requestID := r.URL.Query().Get("id")
	if requestID == "" {
		respondWithError(w, "Request ID required", http.StatusBadRequest)
		return
	}
	
	shareReq, exists := shareRequests[requestID]
	if !exists {
		respondWithError(w, "Share request not found", http.StatusNotFound)
		return
	}
	
	// Check if request has expired
	if time.Now().After(shareReq.ExpiresAt) {
		delete(shareRequests, requestID)
		respondWithError(w, "Share request has expired", http.StatusGone)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(shareReq)
}

// Share accept handler - accept a share request
func shareAcceptHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" {
		respondWithError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var request struct {
		RequestID string `json:"request_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	shareReq, exists := shareRequests[request.RequestID]
	if !exists {
		respondWithError(w, "Share request not found", http.StatusNotFound)
		return
	}
	
	// Check if request has expired
	if time.Now().After(shareReq.ExpiresAt) {
		delete(shareRequests, request.RequestID)
		respondWithError(w, "Share request has expired", http.StatusGone)
		return
	}
	
	// Update request status
	shareReq.Status = "accepted"
	
	// Add to history
	shareHistory = append(shareHistory, *shareReq)
	
	// Notify sender
	notification := map[string]interface{}{
		"type":    "share_accepted",
		"request": shareReq,
		"message": fmt.Sprintf("%s accepted your share request", selfDevice.Name),
	}
	
	broadcastToDevice(shareReq.FromDevice, notification)
	
	// Start transfer process based on type
	switch shareReq.Type {
	case "file":
		go processFileShare(shareReq)
	case "message":
		go processMessageShare(shareReq)
	}
	
	log.Printf("Share request accepted: %s from %s", shareReq.Type, shareReq.FromName)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Share request accepted",
		"request": shareReq,
	})
}

// Share reject handler - reject a share request
func shareRejectHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" {
		respondWithError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var request struct {
		RequestID string `json:"request_id"`
		Reason    string `json:"reason,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	shareReq, exists := shareRequests[request.RequestID]
	if !exists {
		respondWithError(w, "Share request not found", http.StatusNotFound)
		return
	}
	
	// Update request status
	shareReq.Status = "rejected"
	
	// Add to history
	shareHistory = append(shareHistory, *shareReq)
	
	// Remove from pending requests
	delete(shareRequests, request.RequestID)
	
	// Notify sender
	notification := map[string]interface{}{
		"type":    "share_rejected",
		"request": shareReq,
		"message": fmt.Sprintf("%s rejected your share request", selfDevice.Name),
		"reason":  request.Reason,
	}
	
	broadcastToDevice(shareReq.FromDevice, notification)
	
	log.Printf("Share request rejected: %s from %s", shareReq.Type, shareReq.FromName)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Share request rejected",
	})
}

// Share history handler - get sharing history
func shareHistoryHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "GET" {
		respondWithError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Filter history for current device
	deviceHistory := make([]ShareRequest, 0)
	for _, req := range shareHistory {
		if req.FromDevice == selfDevice.ID || req.ToDevice == selfDevice.ID {
			deviceHistory = append(deviceHistory, req)
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"history": deviceHistory,
		"count":   len(deviceHistory),
	})
}

// Device connect handler - establish connection with specific device
func deviceConnectHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" {
		respondWithError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var request struct {
		DeviceID string `json:"device_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	device := findDeviceByID(request.DeviceID)
	if device == nil {
		respondWithError(w, "Device not found", http.StatusNotFound)
		return
	}
	
	// Test connection to device
	connected := testDeviceConnection(device.IP, device.Port)
	device.IsConnected = connected
	
	if connected {
		// Send connection notification
		notification := map[string]interface{}{
			"type":    "device_connected",
			"device":  selfDevice,
			"message": fmt.Sprintf("%s wants to connect", selfDevice.Name),
		}
		
		broadcastToDevice(device.ID, notification)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   connected,
		"connected": connected,
		"device":    device,
		"message":   fmt.Sprintf("Connection %s", map[bool]string{true: "successful", false: "failed"}[connected]),
	})
}

// Device disconnect handler
func deviceDisconnectHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" {
		respondWithError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var request struct {
		DeviceID string `json:"device_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	device := findDeviceByID(request.DeviceID)
	if device != nil {
		device.IsConnected = false
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Disconnected successfully",
	})
}

// Device info handler - get detailed device information
func deviceInfoHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "GET" {
		respondWithError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Get client IP for logging
	clientIP := getClientIP(r)
	log.Printf("Device info requested by client: %s", clientIP)
	
	// Return current device info with client IP
	deviceInfo := map[string]interface{}{
		"device":     selfDevice,
		"client_ip":  clientIP,
		"version":    "2.0.0",
		"features":   []string{"file_sharing", "messaging", "private_sharing", "group_sharing"},
		"max_file_size": config.MaxUploadSize,
		"supported_types": []string{"image", "video", "audio", "document", "archive", "other"},
		"total_devices": len(getDevices()),
		"connected_clients": len(hub.clients),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deviceInfo)
}

// Helper functions
func findDeviceByID(deviceID string) *Device {
	for i, device := range devices {
		if device.ID == deviceID {
			return &devices[i]
		}
	}
	return nil
}

func testDeviceConnection(ip string, port int) bool {
	if port == 0 {
		port = 8081 // Default port
	}
	
	address := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func broadcastToDevice(deviceID string, message map[string]interface{}) {
	// Find device and send WebSocket message
	device := findDeviceByID(deviceID)
	if device == nil {
		return
	}
	
	// Convert message to JSON
	messageData, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling broadcast message: %v", err)
		return
	}
	
	// Send via WebSocket hub
	hub.broadcast <- messageData
}

func processFileShare(shareReq *ShareRequest) {
	// Process file sharing in background
	log.Printf("Processing file share: %d files from %s to %s", len(shareReq.Files), shareReq.FromName, shareReq.ToName)
	
	// Create transfer records for each file
	for _, fileName := range shareReq.Files {
		transfer := &Transfer{
			ID:               generateID(),
			FileName:         fileName,
			FileSize:         shareReq.Size / int64(len(shareReq.Files)), // Approximate
			Progress:         0,
			Status:           "pending",
			StartedAt:        time.Now(),
			FromDevice:       shareReq.FromDevice,
			ToDevice:         shareReq.ToDevice,
			BytesTransferred: 0,
			Type:             "private_share",
		}
		
		transfers[transfer.ID] = transfer
		if err := insertTransfer(*transfer); err != nil {
			log.Printf("Error inserting transfer record: %v", err)
		}
	}
	
	// Update share request status
	shareReq.Status = "completed"
}

func processMessageShare(shareReq *ShareRequest) {
	// Process message sharing
	log.Printf("Processing message share from %s to %s: %s", shareReq.FromName, shareReq.ToName, shareReq.Message)
	
	// Create chat message
	chatMsg := ChatMessage{
		Type:      "private_message",
		Username:  shareReq.FromName,
		Message:   shareReq.Message,
		Timestamp: time.Now(),
	}
	
	// Send to specific device
	notification := map[string]interface{}{
		"type":    "private_message",
		"message": chatMsg,
		"from":    shareReq.FromDevice,
	}
	
	broadcastToDevice(shareReq.ToDevice, notification)
	
	// Update share request status
	shareReq.Status = "completed"
}

// Device scan handler - trigger manual device scan
func deviceScanHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" {
		respondWithError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Trigger immediate network scan
	go func() {
		localIP, err := getLocalIP()
		if err != nil {
			log.Printf("Failed to get local IP for scan: %v", err)
			return
		}
		
		log.Printf("Manual device scan triggered")
		scanSubnet(localIP)
	}()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Device scan started",
	})
}

// Device add handler - manually add device by IP
func deviceAddHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" {
		respondWithError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var request struct {
		IP   string `json:"ip"`
		Name string `json:"name,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if request.IP == "" {
		respondWithError(w, "IP address required", http.StatusBadRequest)
		return
	}
	
	// Check if it's a GoLANshare instance
	go checkGoLANshareInstance(request.IP)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Checking device at " + request.IP,
	})
}

// Device test handler - test if device discovery is working
func deviceTestHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Device test endpoint called")
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "GET" {
		respondWithError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Add some demo devices for testing if no real devices found
	nonSelfDevices := 0
	for _, device := range devices {
		if !device.IsSelf {
			nonSelfDevices++
		}
	}
	
	if nonSelfDevices == 0 { // No non-self devices found
		demoDevices := []Device{
			{
				ID:          "demo-device-1",
				Name:        "Demo-Windows-PC",
				IP:          "192.168.1.100",
				LastSeen:    time.Now(),
				OS:          "Windows",
				Status:      "online",
				IsSelf:      false,
				Port:        8081,
				Version:     "2.0.0",
				IsConnected: false,
				DeviceType:  "Windows",
			},
			{
				ID:          "demo-device-2",
				Name:        "Demo-MacBook",
				IP:          "192.168.1.101",
				LastSeen:    time.Now(),
				OS:          "macOS",
				Status:      "online",
				IsSelf:      false,
				Port:        8081,
				Version:     "2.0.0",
				IsConnected: false,
				DeviceType:  "macOS",
			},
		}
		
		log.Printf("Adding demo devices for testing...")
		for _, device := range demoDevices {
			log.Printf("Adding demo device: %s (%s)", device.Name, device.IP)
			updateOrAddDevice(device)
		}
		
		log.Printf("Added demo devices for testing")
	}
	
	// Return test response to help with device discovery
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service":    "GoLANshare",
		"version":    "2.0.0",
		"device":     selfDevice,
		"devices":    getDevices(),
		"timestamp":  time.Now(),
		"message":    "GoLANshare device discovery test endpoint",
	})
}

// File delete handler - delete shared files
func fileDeleteHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "DELETE" && r.Method != "POST" {
		respondWithError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var request struct {
		Filename string `json:"filename"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if request.Filename == "" {
		respondWithError(w, "Filename required", http.StatusBadRequest)
		return
	}
	
	// Security check - prevent path traversal
	cleanFilename := filepath.Base(request.Filename)
	if cleanFilename != request.Filename || strings.Contains(cleanFilename, "..") {
		respondWithError(w, "Invalid filename", http.StatusBadRequest)
		return
	}
	
	// Check if file exists
	filePath := filepath.Join(config.UploadPath, cleanFilename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		respondWithError(w, "File not found", http.StatusNotFound)
		return
	}
	
	// Delete file from filesystem
	if err := os.Remove(filePath); err != nil {
		log.Printf("Error deleting file %s: %v", filePath, err)
		respondWithError(w, "Failed to delete file", http.StatusInternalServerError)
		return
	}
	
	// Delete from database
	if err := deleteFile(cleanFilename); err != nil {
		log.Printf("Error deleting file from database %s: %v", cleanFilename, err)
	}
	
	log.Printf("File deleted: %s by %s", cleanFilename, getCurrentUser(r))
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "File deleted successfully",
		"filename": cleanFilename,
	})
}

// Debug handler to show connected clients
func debugClientsHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "GET" {
		respondWithError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	clientIP := getClientIP(r)
	log.Printf("Debug clients info requested by: %s", clientIP)
	
	// Get all devices
	allDevices := getDevices()
	
	// Get WebSocket clients count
	clientsCount := len(hub.clients)
	
	debugInfo := map[string]interface{}{
		"requesting_client_ip": clientIP,
		"total_devices": len(allDevices),
		"websocket_clients": clientsCount,
		"devices": allDevices,
		"self_device": selfDevice,
		"server_info": map[string]interface{}{
			"host": config.Host,
			"port": config.Port,
			"auth_required": config.RequireAuth,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(debugInfo)
}


