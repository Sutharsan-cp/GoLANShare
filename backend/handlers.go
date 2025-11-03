package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type FileInfo struct {
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	Type      string    `json:"type"`
	Modified  time.Time `json:"modified"`
	URL       string    `json:"url"`
	Version   int       `json:"version"`
	Expiry    time.Time `json:"expiry"`
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
}

var transfers = make(map[string]*Transfer)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Limit upload size
	r.Body = http.MaxBytesReader(w, r.Body, config.MaxUploadSize)
	if err := r.ParseMultipartForm(config.MaxUploadSize); err != nil {
		http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		return
	}
	
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		http.Error(w, "No files provided", http.StatusBadRequest)
		return
	}
	
	results := make([]FileInfo, 0)
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, "Error retrieving file", http.StatusBadRequest)
			return
		}
		defer file.Close()
		
		// Sanitize filename
		safeName := filepath.Base(fileHeader.Filename)
		if safeName == "" || safeName == "." || safeName == ".." {
			http.Error(w, "Invalid filename", http.StatusBadRequest)
			return
		}
		
		filePath := filepath.Join(config.UploadPath, safeName)
		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Error creating file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()
		
		_, err = io.Copy(dst, file)
		if err != nil {
			http.Error(w, "Error saving file", http.StatusInternalServerError)
			return
		}
		
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			http.Error(w, "Error getting file info", http.StatusInternalServerError)
			return
		}
		
		fInfo := FileInfo{
			Name:     safeName,
			Size:     fileInfo.Size(),
			Type:     getFileType(safeName),
			Modified: time.Now(),
			URL:      "/files/" + safeName,
			Version:  1,
			Expiry:   time.Now().Add(24 * time.Hour),
		}
		
		if err := insertFileInfo(fInfo); err != nil {
			log.Printf("Error inserting file info: %v", err)
		}
		
		results = append(results, fInfo)
		
		transfer := &Transfer{
			ID:          generateID(),
			FileName:    safeName,
			FileSize:    fileHeader.Size,
			Progress:    100,
			Status:      "completed",
			StartedAt:   time.Now(),
			CompletedAt: time.Now(),
			FromDevice:  getClientIP(r),
		}
		transfers[transfer.ID] = transfer
		
		if err := insertTransfer(*transfer); err != nil {
			log.Printf("Error inserting transfer: %v", err)
		}
		
		// Compress if enabled
		if getSetting("compress") == "true" {
			go compressFile(filePath)
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	path := strings.TrimPrefix(r.URL.Path, "/api/download/")
	if path == "" {
		http.Error(w, "File name required", http.StatusBadRequest)
		return
	}
	
	// Security check
	if strings.Contains(path, "..") || strings.Contains(path, "/") || strings.Contains(path, "\\") {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}
	
	filePath := filepath.Join(config.UploadPath, path)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}
	
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Error opening file", http.StatusInternalServerError)
		return
	}
	defer file.Close()
	
	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Error getting file info", http.StatusInternalServerError)
		return
	}
	
	// Support for range requests (resumable downloads)
	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		ranges := parseRange(rangeHeader, stat.Size())
		if len(ranges) > 0 {
			start, end := ranges[0].start, ranges[0].end
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, stat.Size()))
			w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
			w.WriteHeader(http.StatusPartialContent)
			
			file.Seek(start, 0)
			io.CopyN(w, file, end-start+1)
			return
		}
	}
	
	// Regular download
	w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filepath.Base(filePath)+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	
	io.Copy(w, file)
}

type httpRange struct {
	start, end int64
}

func parseRange(rangeHeader string, size int64) []httpRange {
	if strings.HasPrefix(rangeHeader, "bytes=") {
		rangeStr := strings.TrimPrefix(rangeHeader, "bytes=")
		parts := strings.Split(rangeStr, "-")
		if len(parts) == 2 {
			start, err1 := strconv.ParseInt(parts[0], 10, 64)
			end, err2 := strconv.ParseInt(parts[1], 10, 64)
			
			if err1 == nil && err2 == nil {
				if end == 0 {
					end = size - 1
				}
				if start >= 0 && end < size && start <= end {
					return []httpRange{{start, end}}
				}
			}
		}
	}
	return nil
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
	
	// Return only real devices, no demo data
	devs := getDevices()
	
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
	return r.RemoteAddr
}