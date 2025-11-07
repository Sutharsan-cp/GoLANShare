# GoLANshare - Quick Start Guide

## 🚀 **Get Started in 3 Steps**

### **Step 1: Start the Server**

**Windows:**
```cmd
# Double-click start.bat or run in terminal
start.bat
```

**Linux/Mac:**
```bash
# Make executable and run
chmod +x start.sh
./start.sh
```

### **Step 2: Open in Browser**
```
http://localhost:8081
```

**Login:**
- Username: `admin`
- Password: `admin`

### **Step 3: Start Sharing!**

## 📱 **Sharing Files Between Devices**

### **On Device A (Sender):**
1. Start GoLANshare
2. Wait for devices to appear (usually 5-10 seconds)
3. You'll see other devices in "Available Devices" section

### **On Device B (Receiver):**
1. Start GoLANshare on another device
2. Both devices will discover each other automatically

### **Send Files:**
1. On Device A, click **"Share"** button on Device B's card
2. Choose **"Share Files"**
3. Select files to send
4. Device B receives a notification
5. Device B clicks **"Accept"**
6. Transfer starts automatically!

## 💬 **Sending Messages**

1. Click **"Message"** button on any device
2. Type your message
3. Click **"Send Message"**
4. Recipient receives instantly!

## 📋 **Quick Share Features**

### **Share Clipboard:**
1. Copy text to clipboard
2. Click **"Share Clipboard"** in Private Sharing panel
3. Select target device
4. Done!

### **Share Multiple Files:**
1. Click **"Share Files"** quick action
2. Select device
3. Choose multiple files
4. Send!

## 🔍 **Finding Devices**

### **Automatic Discovery:**
- Devices appear automatically within 5-15 seconds
- Shows: Name, IP, OS, Connection Status
- Only shows devices running GoLANshare

### **Manual Refresh:**
- Click **"Scan Again"** button
- Or press **Ctrl+R**

### **Connection Status:**
- 🟢 Green = Connected and ready
- 🔴 Red = Not connected
- Click **"Connect"** to establish connection

## 📊 **Dashboard Overview**

### **Stats Cards:**
- **Devices Available**: Number of GoLANshare devices found
- **Active Transfers**: Ongoing file transfers
- **Messages**: Unread messages
- **Shared Files**: Files available for download

### **Device Cards Show:**
- Device icon (based on OS)
- Device name
- IP address and OS
- Connection status
- Action buttons (Share, Message, Connect)

## 🎯 **Common Tasks**

### **Upload Files for Sharing:**
1. Drag files to upload zone
2. Or click **"Select Files"**
3. Files appear in "Shared Files" section
4. Anyone can download them

### **Download Shared Files:**
1. Go to "Shared Files" section
2. Click **"Download"** on any file
3. File downloads to your device

### **View Transfer Progress:**
1. Check "File Transfers" section
2. See real-time progress bars
3. View speed and ETA
4. Pause/Resume/Cancel as needed

### **Check Share History:**
1. Click **"History"** button
2. See all sent and received shares
3. View status (completed, rejected, pending)

### **Manage Pending Requests:**
1. Click **"Pending"** button
2. See incoming share requests
3. Accept or Reject each request

## ⚙️ **Settings**

### **Access Settings:**
1. Click **"Configure"** in Settings panel
2. Adjust preferences:
   - Security (auth, encryption)
   - Storage (auto-delete, compression)
   - Network (port, upload size)
   - Interface (theme, notifications)
3. Click **"Save Settings"**

### **Change Theme:**
1. Open Settings
2. Select theme: Auto, Light, or Dark
3. Save

## 🔧 **Troubleshooting**

### **No Devices Showing:**
```
✓ Check: Both devices on same WiFi/network
✓ Check: Firewall allows port 8081
✓ Check: GoLANshare running on both devices
✓ Try: Click "Scan Again" button
✓ Try: Restart GoLANshare
```

### **Connection Failed:**
```
✓ Check: Device is online (green indicator)
✓ Check: Network is stable
✓ Try: Click "Connect" button again
✓ Try: Refresh page (F5)
```

### **Transfer Stuck:**
```
✓ Check: Network connection
✓ Check: Disk space available
✓ Try: Click "Retry" button
✓ Try: Re-upload file
```

### **Can't Login:**
```
✓ Use: admin / admin
✓ Try: Clear browser cache
✓ Check: config.json has require_auth: false
```

## 🌐 **Network Setup**

### **Find Your IP:**
- Shown in header: `10.50.18.20:8081`
- Or check network settings

### **Access from Other Devices:**
```
http://[YOUR-IP]:8081
Example: http://10.50.18.20:8081
```

### **Firewall (Windows):**
```cmd
netsh advfirewall firewall add rule name="GoLANshare" dir=in action=allow protocol=TCP localport=8081
```

### **Firewall (Linux):**
```bash
sudo ufw allow 8081
```

## 📱 **Mobile Access**

### **Connect Mobile Device:**
1. Ensure mobile on same WiFi
2. Open browser on mobile
3. Go to `http://[PC-IP]:8081`
4. Login and start sharing!

### **Install as App:**
1. Open in mobile browser
2. Menu > "Add to Home Screen"
3. Use like native app

### **QR Code (Coming Soon):**
1. Click device's "Send File"
2. Scan QR with mobile
3. Instant connection

## ⌨️ **Keyboard Shortcuts**

- **Ctrl+U**: Open file picker
- **Ctrl+R**: Refresh all data
- **Ctrl+/**: Focus chat input
- **Esc**: Close modals

## 💡 **Tips & Tricks**

### **Fast Sharing:**
1. Keep GoLANshare open on all devices
2. Devices stay connected
3. Share instantly without waiting

### **Bulk Sharing:**
1. Select multiple files at once
2. Send to one device
3. All files transfer together

### **Organize Files:**
1. Use descriptive filenames
2. Check "Shared Files" regularly
3. Download what you need
4. Files auto-delete after 24h

### **Stay Connected:**
1. Keep devices on same network
2. Don't close GoLANshare
3. Check connection status (green dot)
4. Reconnect if needed

## 🎉 **You're Ready!**

**GoLANshare is now running and ready to share files!**

### **Quick Checklist:**
- ✅ Server running on port 8081
- ✅ Logged in to web interface
- ✅ Devices discovered automatically
- ✅ Ready to share files and messages

### **Next Steps:**
1. Start GoLANshare on another device
2. Wait for device discovery
3. Click "Share" and send your first file!

### **Need Help?**
- Check FEATURES.md for complete feature list
- Check README.md for detailed documentation
- Check logs in `logs/golanshare.log`

**Happy Sharing! 🚀**
