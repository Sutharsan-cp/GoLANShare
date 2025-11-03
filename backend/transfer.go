package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func handleChunkedUpload(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := r.URL.Query().Get("token")
	filename := r.URL.Query().Get("filename")
	chunkStr := r.URL.Query().Get("chunk")
	totalChunksStr := r.URL.Query().Get("totalChunks")
	chunkHash := r.URL.Query().Get("hash")

	if token == "" || filename == "" || chunkStr == "" || totalChunksStr == "" {
		http.Error(w, "Missing parameters", http.StatusBadRequest)
		return
	}

	chunk, err := strconv.Atoi(chunkStr)
	if err != nil {
		http.Error(w, "Invalid chunk number", http.StatusBadRequest)
		return
	}

	totalChunks, err := strconv.Atoi(totalChunksStr)
	if err != nil {
		http.Error(w, "Invalid total chunks", http.StatusBadRequest)
		return
	}

	if _, exists := transfers[token]; !exists {
		transfers[token] = &Transfer{
			ID:        token,
			FileName:  filename,
			Progress:  0,
			Status:    "uploading",
			StartedAt: time.Now(),
			FromDevice: getClientIP(r),
		}
		if err := insertTransfer(*transfers[token]); err != nil {
			log.Printf("Error inserting transfer: %v", err)
		}
	}

	chunkDir := filepath.Join(config.StoragePath, "chunks", token)
	if err := os.MkdirAll(chunkDir, 0755); err != nil {
		http.Error(w, "Error creating chunk directory", http.StatusInternalServerError)
		return
	}

	chunkPath := filepath.Join(chunkDir, fmt.Sprintf("chunk_%d", chunk))
	chunkFile, err := os.Create(chunkPath)
	if err != nil {
		http.Error(w, "Error creating chunk file", http.StatusInternalServerError)
		return
	}
	defer chunkFile.Close()

	// Read chunk data with limit
	r.Body = http.MaxBytesReader(w, r.Body, 100*1024*1024) // 100MB per chunk max
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading chunk", http.StatusBadRequest)
		return
	}

	if _, err := chunkFile.Write(body); err != nil {
		http.Error(w, "Error writing chunk", http.StatusInternalServerError)
		return
	}

	// Verify hash if provided
	if chunkHash != "" {
		sum := sha256.Sum256(body)
		computedHash := hex.EncodeToString(sum[:])
		if computedHash != chunkHash {
			os.Remove(chunkPath)
			http.Error(w, "Hash mismatch", http.StatusBadRequest)
			return
		}
	}

	// Update progress
	progress := ((chunk + 1) * 100) / totalChunks
	if progress > 100 {
		progress = 100
	}
	
	transfers[token].Progress = progress
	transfers[token].FileSize += int64(len(body))

	if err := updateTransfer(*transfers[token]); err != nil {
		log.Printf("Error updating transfer progress: %v", err)
	}

	// If this is the last chunk, combine them
	if chunk == totalChunks-1 {
		if err := combineChunks(token, filename); err != nil {
			http.Error(w, "Error combining chunks: "+err.Error(), http.StatusInternalServerError)
			return
		}
		transfers[token].Status = "completed"
		transfers[token].Progress = 100
		transfers[token].CompletedAt = time.Now()
		if err := updateTransfer(*transfers[token]); err != nil {
			log.Printf("Error updating completed transfer: %v", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success", 
		"progress": progress,
		"chunk": chunk,
	})
}

func combineChunks(token, filename string) error {
	chunkDir := filepath.Join(config.StoragePath, "chunks", token)
	finalPath := filepath.Join(config.UploadPath, filename)
	
	// Ensure upload directory exists
	if err := os.MkdirAll(config.UploadPath, 0755); err != nil {
		return fmt.Errorf("create upload directory: %v", err)
	}

	finalFile, err := os.Create(finalPath)
	if err != nil {
		return fmt.Errorf("create final file: %v", err)
	}
	defer finalFile.Close()

	// Read chunk files
	files, err := os.ReadDir(chunkDir)
	if err != nil {
		return fmt.Errorf("read chunk directory: %v", err)
	}

	var chunks []int
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "chunk_") {
			num, err := strconv.Atoi(strings.TrimPrefix(f.Name(), "chunk_"))
			if err == nil {
				chunks = append(chunks, num)
			}
		}
	}

	if len(chunks) == 0 {
		return fmt.Errorf("no chunks found")
	}

	sort.Ints(chunks)

	// Combine chunks in order
	for _, i := range chunks {
		chunkPath := filepath.Join(chunkDir, fmt.Sprintf("chunk_%d", i))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return fmt.Errorf("open chunk %d: %v", i, err)
		}

		if _, err := io.Copy(finalFile, chunkFile); err != nil {
			chunkFile.Close()
			return fmt.Errorf("copy chunk %d: %v", i, err)
		}
		chunkFile.Close()

		// Remove chunk after combining
		if err := os.Remove(chunkPath); err != nil {
			log.Printf("Error removing chunk %s: %v", chunkPath, err)
		}
	}

	// Clean up chunk directory
	if err := os.RemoveAll(chunkDir); err != nil {
		log.Printf("Error removing chunk directory: %v", err)
	}

	// Save file info to database
	fileInfo, err := os.Stat(finalPath)
	if err != nil {
		return fmt.Errorf("stat final file: %v", err)
	}

	fInfo := FileInfo{
		Name:     filename,
		Size:     fileInfo.Size(),
		Type:     getFileType(filename),
		Modified: time.Now(),
		URL:      "/files/" + filename,
		Version:  1,
		Expiry:   time.Now().Add(24 * time.Hour),
	}

	if err := insertFileInfo(fInfo); err != nil {
		log.Printf("Error inserting file info: %v", err)
	}

	return nil
}

func resumeTransfer(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Token required", http.StatusBadRequest)
		return
	}

	transfer, exists := transfers[token]
	if !exists {
		// Try to load from database
		allTransfers := getAllTransfers()
		for _, t := range allTransfers {
			if t.ID == token {
				transfer = &t
				transfers[token] = &t
				exists = true
				break
			}
		}
	}

	if !exists {
		http.Error(w, "Transfer not found", http.StatusNotFound)
		return
	}

	chunkDir := filepath.Join(config.StoragePath, "chunks", token)
	var uploadedChunks []int
	
	if _, err := os.Stat(chunkDir); err == nil {
		files, err := os.ReadDir(chunkDir)
		if err == nil {
			for _, f := range files {
				if strings.HasPrefix(f.Name(), "chunk_") {
					num, err := strconv.Atoi(strings.TrimPrefix(f.Name(), "chunk_"))
					if err == nil {
						uploadedChunks = append(uploadedChunks, num)
					}
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"transfer":       transfer,
		"uploadedChunks": uploadedChunks,
	})
}

func generateResumableToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}