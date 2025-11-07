# GoLANshare - Secure Local File Sharing

A blazing-fast, secure file sharing tool for local networks with end-to-end encryption and real-time collaboration features.

## Features

- 🚀 **LAN-speed transfers** - Direct device-to-device transfers
- 🔒 **End-to-end encryption** - Your files stay private
- 📁 **Folder sharing** - Preserve directory structures
- 🔄 **Resumable transfers** - Continue interrupted transfers
- 🔍 **Auto device discovery** - Find devices on your network
- 💬 **Built-in chat** - Communicate while sharing files
- 📱 **Progressive Web App** - Works on all devices
- 🎯 **QR code sharing** - Quick connect with mobile devices

## Installation

### Prerequisites
- Go 1.19 or higher
- Modern web browser

### Quick Start

1. **Clone or download** the project files
2. **Run the application**:
   - Windows: Double-click `start.bat`
   - Linux/Mac: Run `./start.sh` (make executable with `chmod +x start.sh`)
3. **Open your browser** to `http://localhost:8081`

### Manual Setup

```bash
# 1. Navigate to backend directory
cd backend

# 2. Install dependencies
go mod download

# 3. Build the application
go build -o golanshare

# 4. Run the server
./golanshare
```

## How to Use

### 1. Starting the Application

**Windows:**
```cmd
# Double-click start.bat or run in command prompt
start.bat
```

**Linux/Mac:**
```bash
# Make executable and run
chmod +x start.sh
./start.sh
```

### 2. Accessing the Interface

1. Open your web browser
2. Navigate to `http://localhost:8081`
3. Login with default credentials:
   - Username: `admin`
   - Password: `admin`

### 3. Sharing Files

#### Upload Files
1. **Drag & Drop**: Drag files directly onto the upload zone
2. **Browse**: Click "Select Files" to choose files
3. **Multiple Files**: Select multiple files at once
4. **Large Files**: Files over 10MB are automatically chunked for reliable transfer

#### Quick Actions
- **📸 Screenshot**: Capture and share screenshots (coming soon)
- **📝 Share Text**: Share text snippets with other devices
- **📂 Share Folder**: Share entire folders (coming soon)

### 4. Connecting Devices

#### Automatic Discovery
- GoLANshare automatically discovers other devices on your network
- Devices appear in the "Available Devices" section
- Shows device name, IP address, and operating system

#### Manual Connection
1. Note your device's IP address from the interface
2. On another device, open a browser to `http://[IP]:8081`
3. Use the same login credentials

#### QR Code Connection
1. Click "Send File" next to any discovered device
2. Scan the QR code with your mobile device
3. Mobile device will connect automatically

### 5. Real-time Chat

- Use the built-in chat to communicate with connected devices
- Messages are synchronized across all connected devices
- Perfect for coordinating file transfers

### 6. Managing Transfers

#### Monitor Progress
- View all active transfers in the "File Transfers" section
- See progress bars, transfer speeds, and status
- Track completed, paused, and failed transfers

#### Transfer Controls
- **Pause All**: Pause all active transfers
- **Resume All**: Resume paused transfers
- **Clear Completed**: Remove completed transfers from the list

### 7. Advanced Features

#### Resumable Transfers
- Large file transfers can be resumed if interrupted
- Chunked uploads ensure reliability
- Automatic retry on network issues

#### Security
- Optional authentication (can be disabled for demo mode)
- Files are stored temporarily and auto-deleted after 24 hours
- Local network only - no internet connection required

## Configuration

### Config File (`config.json`)

```json
{
  "port": "8081",
  "host": "0.0.0.0",
  "enable_tls": false,
  "require_auth": true,
  "storage_path": "./storage",
  "upload_path": "./storage/uploads",
  "max_upload_size": 1073741824
}
```

### Settings
- **Port**: Change the server port (default: 8081)
- **Authentication**: Enable/disable login requirement
- **Storage Path**: Where uploaded files are stored
- **Max Upload Size**: Maximum file size limit (1GB default)
- **TLS**: Enable HTTPS (requires certificates)

## Network Setup

### Same Network
- All devices must be on the same local network (WiFi/Ethernet)
- Ensure firewall allows connections on the chosen port
- Router should allow device-to-device communication

### Firewall Configuration

**Windows:**
```cmd
# Allow GoLANshare through Windows Firewall
netsh advfirewall firewall add rule name="GoLANshare" dir=in action=allow protocol=TCP localport=8081
```

**Linux (UFW):**
```bash
# Allow port 8081
sudo ufw allow 8081
```

**macOS:**
- Go to System Preferences > Security & Privacy > Firewall
- Add GoLANshare to allowed applications

## Troubleshooting

### Common Issues

**1. Cannot access from other devices**
- Check firewall settings
- Verify all devices are on same network
- Try accessing via IP address directly

**2. Files not uploading**
- Check available disk space
- Verify file size is under the limit
- Try smaller files first

**3. Devices not discovered**
- Ensure mDNS/Bonjour is enabled
- Check network allows multicast
- Try manual connection via IP

**4. Login issues**
- Use default credentials: admin/admin
- Check if authentication is enabled in config
- Clear browser cache and cookies

### Performance Tips

1. **Use wired connections** for fastest transfers
2. **Close other network applications** during large transfers
3. **Use chunked uploads** for files over 10MB
4. **Monitor network usage** to avoid congestion

## Development

### Building from Source

```bash
# Clone repository
git clone <repository-url>
cd golanshare

# Install Go dependencies
cd backend
go mod download

# Build
go build -o golanshare

# Run
./golanshare
```

### Project Structure

```
golanshare/
├── backend/           # Go backend server
│   ├── main.go       # Main server file
│   ├── auth.go       # Authentication
│   ├── handlers.go   # HTTP handlers
│   ├── db.go         # Database operations
│   ├── discovery.go  # Device discovery
│   ├── transfer.go   # File transfer logic
│   └── utils.go      # Utilities
├── frontend/         # Web interface
│   ├── index.html    # Main HTML
│   ├── css/style.css # Styles
│   ├── js/app.js     # JavaScript app
│   └── manifest.json # PWA manifest
├── config.json       # Configuration
├── start.bat         # Windows startup
└── start.sh          # Linux/Mac startup
```

## Security Notes

- GoLANshare is designed for **local network use only**
- Files are **not encrypted at rest** by default
- **Change default passwords** in production
- **Enable TLS** for sensitive environments
- Files **auto-delete after 24 hours**

## License

This project is open source. See LICENSE file for details.

## Support

For issues and questions:
1. Check the troubleshooting section above
2. Verify your network configuration
3. Check firewall settings
4. Try with different devices/browsers