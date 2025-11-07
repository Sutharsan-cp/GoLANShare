# GoLANshare Demo Guide

## Quick Demo Steps

### 1. Start the Application

**Windows:**
```cmd
# Double-click start.bat or run:
start.bat
```

**Linux/Mac:**
```bash
chmod +x start.sh
./start.sh
```

### 2. Access the Web Interface

1. Open your browser to: `http://localhost:8081`
2. Login with default credentials:
   - Username: `admin`
   - Password: `admin`

### 3. Demo Features

#### File Upload Demo
1. **Drag & Drop Test**: Drag any file onto the upload zone
2. **Browse Upload**: Click "Select Files" and choose multiple files
3. **Large File Test**: Try uploading a file over 10MB to see chunked upload

#### Device Discovery Demo
1. **Same Device**: Open another browser tab to simulate another device
2. **Network Discovery**: If you have other devices on the same network, they should appear automatically
3. **QR Code**: Click "Send File" next to any device to see QR code generation

#### Chat Demo
1. **Multi-tab Chat**: Open multiple browser tabs and chat between them
2. **Real-time Updates**: Messages appear instantly across all connected clients

#### File Sharing Demo
1. **Upload Files**: Upload some test files
2. **View Shared Files**: Check the "Shared Files" section
3. **Download**: Download files from the shared files list
4. **Share Links**: Use the share button to copy file links

### 4. Network Testing

#### Test with Multiple Devices
1. **Find Your IP**: Note your computer's IP address (shown in the interface)
2. **Connect from Phone**: On your phone, browse to `http://[YOUR-IP]:8081`
3. **Cross-Device Transfer**: Upload files from one device, download from another

#### QR Code Connection
1. **Generate QR**: Click "Send File" next to any discovered device
2. **Mobile Scan**: Use your phone's camera to scan the QR code
3. **Instant Access**: Phone will connect automatically to GoLANshare

### 5. Advanced Features Demo

#### Resumable Transfers
1. **Start Large Upload**: Begin uploading a large file (>50MB)
2. **Interrupt Transfer**: Close browser or disconnect network
3. **Resume**: Reconnect and see transfer resume automatically

#### Real-time Monitoring
1. **Transfer Progress**: Watch real-time progress bars during uploads
2. **Device Status**: See devices go online/offline in real-time
3. **Chat Notifications**: Receive instant chat messages

### 6. Configuration Demo

#### Settings
1. **Access Config**: Edit `config.json` to change settings
2. **Port Change**: Change port from 8081 to another port
3. **Auth Toggle**: Disable authentication for easier access

#### Security Features
1. **File Expiry**: Files auto-delete after 24 hours
2. **Local Only**: All transfers stay on local network
3. **No Internet**: Works completely offline

## Demo Scenarios

### Scenario 1: Office File Sharing
- **Setup**: Multiple computers on office network
- **Use Case**: Share presentations, documents, images quickly
- **Demo**: Upload presentation on laptop, download on meeting room computer

### Scenario 2: Mobile Photo Transfer
- **Setup**: Phone and computer on same WiFi
- **Use Case**: Transfer photos from phone to computer
- **Demo**: Use QR code to connect phone, upload photos, download on computer

### Scenario 3: Collaborative Work
- **Setup**: Team members on same network
- **Use Case**: Share files and coordinate via chat
- **Demo**: Multiple people uploading files, chatting about project

### Scenario 4: Large File Transfer
- **Setup**: Two computers with large files to transfer
- **Use Case**: Transfer video files, backups, large datasets
- **Demo**: Upload large file with chunked transfer, resume if interrupted

## Performance Testing

### Speed Test
1. **Small Files**: Upload multiple small files (< 1MB each)
2. **Medium Files**: Upload files 10-100MB
3. **Large Files**: Upload files > 100MB
4. **Concurrent**: Multiple users uploading simultaneously

### Network Load
1. **Monitor Usage**: Check network utilization during transfers
2. **Multiple Streams**: Start several transfers simultaneously
3. **Bandwidth Limit**: Test with limited network bandwidth

## Troubleshooting Demo

### Common Issues
1. **Firewall**: Demonstrate firewall blocking and resolution
2. **Network**: Show what happens when devices are on different networks
3. **Browser**: Test with different browsers (Chrome, Firefox, Safari, Edge)

### Recovery
1. **Connection Loss**: Disconnect network during transfer
2. **Browser Crash**: Close browser during upload
3. **Server Restart**: Restart server with active transfers

## Demo Tips

### Preparation
- Have test files ready (various sizes: 1KB, 1MB, 10MB, 100MB+)
- Prepare multiple devices (laptop, phone, tablet)
- Ensure all devices are on same network
- Have QR code scanner ready on mobile devices

### Presentation
- Start with simple file upload to show basic functionality
- Progress to advanced features like device discovery
- Show real-world scenarios that audience can relate to
- Demonstrate error handling and recovery

### Interactive Elements
- Let audience upload their own files
- Have them connect their phones via QR code
- Show collaborative features with multiple participants
- Demonstrate chat functionality for coordination

This demo showcases GoLANshare as a complete, production-ready file sharing solution for local networks!