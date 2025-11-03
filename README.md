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
   - Linux/Mac: Run `./start.sh`
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