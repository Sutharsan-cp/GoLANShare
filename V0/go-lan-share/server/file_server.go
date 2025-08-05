package main

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "crypto/sha256"
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

    "github.com/gorilla/mux"
    "github.com/gorilla/websocket"
)

type FileServer struct {
    uploadDir   string
    downloadDir string
    clients     map[*websocket.Conn]*Client
    upgrader    websocket.Upgrader
}

type Client struct {
    conn     *websocket.Conn
    id       string
    name     string
    ip       string
    lastSeen time.Time
}

type FileTransfer struct {
    ID       string    `json:"id"`
    FileName string    `json:"fileName"`
    FileSize int64     `json:"fileSize"`
    From     string    `json:"from"`
    To       string    `json:"to"`
    Status   string    `json:"status"`
    Progress float64   `json:"progress"`
    Created  time.Time `json:"created"`
}

type Message struct {
    Type    string      `json:"type"`
    Data    interface{} `json:"data"`
    From    string      `json:"from"`
    To      string      `json:"to"`
    Created time.Time   `json:"created"`
}

func NewFileServer() *FileServer {
    return &FileServer{
        uploadDir:   "./uploads",
        downloadDir: "./downloads",
        clients:     make(map[*websocket.Conn]*Client),
        upgrader: websocket.Upgrader{
            CheckOrigin: func(r *http.Request) bool {
                return true // Allow all origins in development
            },
        },
    }
}

func (fs *FileServer) Start() {
    // Create directories if they don't exist
    os.MkdirAll(fs.uploadDir, 0755)
    os.MkdirAll(fs.downloadDir, 0755)

    r := mux.NewRouter()

    // API routes
    r.HandleFunc("/api/upload", fs.handleUpload).Methods("POST")
    r.HandleFunc("/api/download/{fileId}", fs.handleDownload).Methods("GET")
    r.HandleFunc("/api/files", fs.handleListFiles).Methods("GET")
    r.HandleFunc("/api/devices", fs.handleListDevices).Methods("GET")
    r.HandleFunc("/ws", fs.handleWebSocket)

    // Serve static files
    r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir(fs.uploadDir))))

    // CORS middleware
    r.Use(func(next http.Handler) http.Handler {
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
    })

    fmt.Println("LAN File Share Pro Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", r))
}

func (fs *FileServer) handleUpload(w http.ResponseWriter, r *http.Request) {
    // Parse multipart form
    err := r.ParseMultipartForm(100 << 20) // 100 MB max
    if err != nil {
        http.Error(w, "Failed to parse form", http.StatusBadRequest)
        return
    }

    file, header, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "Failed to get file", http.StatusBadRequest)
        return
    }
    defer file.Close()

    // Generate unique file ID
    fileId := generateFileID()
    fileName := header.Filename
    fileSize := header.Size

    // Check if encryption is requested
    encrypt := r.FormValue("encrypt") == "true"
    
    // Create file path
    filePath := filepath.Join(fs.uploadDir, fileId+"_"+fileName)
    
    // Create destination file
    dst, err := os.Create(filePath)
    if err != nil {
        http.Error(w, "Failed to create file", http.StatusInternalServerError)
        return
    }
    defer dst.Close()

    // Copy file data
    var written int64
    if encrypt {
        written, err = fs.encryptAndCopy(dst, file)
    } else {
        written, err = io.Copy(dst, file)
    }
    
    if err != nil {
        http.Error(w, "Failed to save file", http.StatusInternalServerError)
        return
    }

    // Create transfer record
    transfer := FileTransfer{
        ID:       fileId,
        FileName: fileName,
        FileSize: fileSize,
        From:     r.RemoteAddr,
        Status:   "completed",
        Progress: 100.0,
        Created:  time.Now(),
    }

    // Broadcast file upload notification
    fs.broadcastMessage(Message{
        Type:    "file_uploaded",
        Data:    transfer,
        From:    r.RemoteAddr,
        Created: time.Now(),
    })

    // Return response
    response := map[string]interface{}{
        "success":   true,
        "fileId":    fileId,
        "fileName":  fileName,
        "fileSize":  written,
        "encrypted": encrypt,
        "url":       fmt.Sprintf("/api/download/%s", fileId),
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (fs *FileServer) handleDownload(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    fileId := vars["fileId"]

    // Find file in upload directory
    files, err := filepath.Glob(filepath.Join(fs.uploadDir, fileId+"_*"))
    if err != nil || len(files) == 0 {
        http.Error(w, "File not found", http.StatusNotFound)
        return
    }

    filePath := files[0]
    fileName := strings.TrimPrefix(filepath.Base(filePath), fileId+"_")

    // Open file
    file, err := os.Open(filePath)
    if err != nil {
        http.Error(w, "Failed to open file", http.StatusInternalServerError)
        return
    }
    defer file.Close()

    // Get file info
    fileInfo, err := file.Stat()
    if err != nil {
        http.Error(w, "Failed to get file info", http.StatusInternalServerError)
        return
    }

    // Set headers
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
    w.Header().Set("Content-Type", "application/octet-stream")
    w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

    // Stream file to client
    io.Copy(w, file)
}

func (fs *FileServer) handleListFiles(w http.ResponseWriter, r *http.Request) {
    files, err := filepath.Glob(filepath.Join(fs.uploadDir, "*"))
    if err != nil {
        http.Error(w, "Failed to list files", http.StatusInternalServerError)
        return
    }

    var fileList []map[string]interface{}
    for _, filePath := range files {
        fileInfo, err := os.Stat(filePath)
        if err != nil {
            continue
        }

        fileName := filepath.Base(filePath)
        parts := strings.SplitN(fileName, "_", 2)
        if len(parts) != 2 {
            continue
        }

        fileList = append(fileList, map[string]interface{}{
            "id":       parts[0],
            "name":     parts[1],
            "size":     fileInfo.Size(),
            "modified": fileInfo.ModTime(),
        })
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(fileList)
}

func (fs *FileServer) handleListDevices(w http.ResponseWriter, r *http.Request) {
    var devices []map[string]interface{}
    
    for _, client := range fs.clients {
        devices = append(devices, map[string]interface{}{
            "id":       client.id,
            "name":     client.name,
            "ip":       client.ip,
            "status":   "online",
            "lastSeen": client.lastSeen,
        })
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(devices)
}

func (fs *FileServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := fs.upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("WebSocket upgrade failed: %v", err)
        return
    }
    defer conn.Close()

    // Create client
    client := &Client{
        conn:     conn,
        id:       generateClientID(),
        name:     fmt.Sprintf("Device-%s", generateClientID()[:8]),
        ip:       r.RemoteAddr,
        lastSeen: time.Now(),
    }

    fs.clients[conn] = client

    // Send welcome message
    welcomeMsg := Message{
        Type: "welcome",
        Data: map[string]interface{}{
            "clientId": client.id,
            "message":  "Connected to LAN File Share Pro",
        },
        Created: time.Now(),
    }
    conn.WriteJSON(welcomeMsg)

    // Broadcast device list update
    fs.broadcastDeviceList()

    // Handle messages
    for {
        var msg Message
        err := conn.ReadJSON(&msg)
        if err != nil {
            log.Printf("WebSocket read error: %v", err)
            break
        }

        client.lastSeen = time.Now()
        fs.handleWebSocketMessage(client, msg)
    }

    // Clean up
    delete(fs.clients, conn)
    fs.broadcastDeviceList()
}

func (fs *FileServer) handleWebSocketMessage(client *Client, msg Message) {
    switch msg.Type {
    case "chat":
        // Broadcast chat message
        chatMsg := Message{
            Type: "chat",
            Data: map[string]interface{}{
                "message": msg.Data,
                "from":    client.name,
            },
            From:    client.id,
            Created: time.Now(),
        }
        fs.broadcastMessage(chatMsg)

    case "device_name_update":
        if name, ok := msg.Data.(string); ok {
            client.name = name
            fs.broadcastDeviceList()
        }
    }
}

func (fs *FileServer) broadcastMessage(msg Message) {
    for conn := range fs.clients {
        err := conn.WriteJSON(msg)
        if err != nil {
            log.Printf("WebSocket write error: %v", err)
            conn.Close()
            delete(fs.clients, conn)
        }
    }
}

func (fs *FileServer) broadcastDeviceList() {
    var devices []map[string]interface{}
    
    for _, client := range fs.clients {
        devices = append(devices, map[string]interface{}{
            "id":       client.id,
            "name":     client.name,
            "ip":       client.ip,
            "status":   "online",
            "lastSeen": client.lastSeen,
        })
    }

    msg := Message{
        Type: "device_list_update",
        Data: devices,
        Created: time.Now(),
    }

    fs.broadcastMessage(msg)
}

func (fs *FileServer) encryptAndCopy(dst io.Writer, src io.Reader) (int64, error) {
    // Generate a random key for AES encryption
    key := make([]byte, 32)
    if _, err := rand.Read(key); err != nil {
        return 0, err
    }

    // Create AES cipher
    block, err := aes.NewCipher(key)
    if err != nil {
        return 0, err
    }

    // Generate IV
    iv := make([]byte, aes.BlockSize)
    if _, err := rand.Read(iv); err != nil {
        return 0, err
    }

    // Create stream cipher
    stream := cipher.NewCFBEncrypter(block, iv)

    // Write IV first
    if _, err := dst.Write(iv); err != nil {
        return 0, err
    }

    // Encrypt and write data
    writer := &cipher.StreamWriter{S: stream, W: dst}
    written, err := io.Copy(writer, src)
    
    return written + int64(len(iv)), err
}

func generateFileID() string {
    hash := sha256.New()
    hash.Write([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
    return fmt.Sprintf("%x", hash.Sum(nil))[:16]
}

func generateClientID() string {
    hash := sha256.New()
    hash.Write([]byte(fmt.Sprintf("%d%d", time.Now().UnixNano(), rand.Int())))
    return fmt.Sprintf("%x", hash.Sum(nil))[:16]
}

func main() {
    server := NewFileServer()
    server.Start()
}
