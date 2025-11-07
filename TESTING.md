# GoLANshare - Testing & Troubleshooting Guide

## 🧪 **All Issues Fixed - Testing Guide**

### **✅ Fixed Issues:**
1. **Device Discovery** - Enhanced with multiple detection methods
2. **Clear Completed Button** - Now properly removes completed/cancelled transfers
3. **File Deletion** - Added delete functionality for shared files
4. **QR Code** - Enhanced with multiple services and connection testing

---

## 🔍 **Testing Device Discovery**

### **Method 1: Automatic Discovery (Same Computer)**
```bash
# Terminal 1: Start first instance
cd GoLANShare
start.bat

# Terminal 2: Start second instance on different port
cd GoLANShare/backend
go run . -port=8082
```

Then:
1. Open `http://localhost:8081` (Instance 1)
2. Open `http://localhost:8082` (Instance 2)
3. Wait 30-60 seconds
4. Check "Available Devices" section
5. Should see the other instance

### **Method 2: Manual Device Addition**
1. Click **"Add Device"** button
2. Enter IP: `127.0.0.1:8082` (or other instance IP)
3. Click **"Add Device"**
4. Device should appear in list

### **Method 3: Test Discovery Function**
1. Click **"Test Discovery"** button
2. Check browser console (F12)
3. Should show device information
4. Verify endpoints are working

### **Method 4: Network Discovery (Different Devices)**
```bash
# Device A (Main Computer)
start.bat
# Note the IP shown: e.g., 10.50.18.20:8081

# Device B (Another Computer/Phone)
# Open browser to: http://10.50.18.20:8081
```

---

## 📤 **Testing Transfer Controls**

### **Test Clear Completed Button:**
1. Upload some files (they complete immediately)
2. Click **"Clear Completed"** button
3. Completed transfers should disappear
4. Should show notification with count

### **Test Pause/Resume/Cancel:**
1. Upload a large file (>50MB)
2. During upload, click **"Pause"** - should pause immediately
3. Click **"Resume"** - should continue
4. Click **"Cancel"** - should stop and mark as cancelled

---

## 🗑️ **Testing File Deletion**

### **Delete Shared Files:**
1. Upload some files
2. Go to **"Shared Files"** section
3. Click **"Delete"** button on any file
4. Confirm deletion
5. File should disappear from list and filesystem

---

## 📱 **Testing QR Code Functionality**

### **Generate QR Code:**
1. If you have devices discovered, click any device card
2. Click **"Send File"** or use the share button
3. QR code modal should appear
4. QR code should load (tries multiple services)

### **Test QR Code:**
1. Use phone camera to scan QR code
2. Should open GoLANshare URL
3. Or click **"Test Connection"** button
4. Should verify the connection works

---

## 🔧 **Troubleshooting**

### **No Devices Found:**

**Step 1: Check Basic Connectivity**
```bash
# Test if service is running
curl http://localhost:8081/api/device/test

# Should return JSON with device info
```

**Step 2: Test Manual Addition**
1. Click **"Add Device"**
2. Enter `127.0.0.1` (localhost)
3. Should find itself as a device

**Step 3: Check Network Scanning**
```bash
# Check server logs for scanning activity
# Should see: "Scanning subnet..." messages
# Should see: "Started scanning X IP addresses"
```

**Step 4: Test Different Network**
```bash
# Find your IP
ipconfig

# Try adding your own IP manually
# Example: 192.168.1.100
```

### **Clear Completed Not Working:**

**Check Transfer Status:**
1. Upload files and wait for completion
2. Check transfer status shows "completed"
3. Click "Clear Completed"
4. Should remove completed/cancelled/failed transfers

### **File Deletion Not Working:**

**Check Permissions:**
1. Ensure files are not in use
2. Check file permissions
3. Try deleting different file types
4. Check server logs for errors

### **QR Code Not Loading:**

**Check Network:**
1. Ensure internet connection (uses online QR services)
2. Try **"Test Connection"** button
3. Manually copy URL and test in browser
4. Check browser console for errors

---

## 🚀 **Complete Testing Checklist**

### **✅ Device Discovery:**
- [ ] Automatic discovery works (wait 60 seconds)
- [ ] Manual device addition works
- [ ] Test discovery button shows device info
- [ ] Devices appear with correct information

### **✅ Transfer Controls:**
- [ ] Clear completed removes finished transfers
- [ ] Pause button stops active transfers
- [ ] Resume button continues paused transfers
- [ ] Cancel button stops and marks as cancelled

### **✅ File Management:**
- [ ] Files can be uploaded successfully
- [ ] Shared files list shows uploaded files
- [ ] Delete button removes files from list and disk
- [ ] Download button works for shared files

### **✅ QR Code:**
- [ ] QR code modal opens when clicking device actions
- [ ] QR code image loads successfully
- [ ] URL can be copied to clipboard
- [ ] Test connection button verifies connectivity

### **✅ Private Messaging:**
- [ ] Message button opens compose dialog
- [ ] Messages can be sent between devices
- [ ] Share requests work between devices
- [ ] File sharing works between specific devices

---

## 🎯 **Quick Test Commands**

### **Test All Endpoints:**
```bash
# Device info
curl http://localhost:8081/api/device/info

# Device test
curl http://localhost:8081/api/device/test

# Devices list
curl http://localhost:8081/api/devices

# Files list
curl http://localhost:8081/api/files
```

### **Test File Operations:**
```bash
# Upload a file (via UI)
# Then test download
curl http://localhost:8081/files/[filename]

# Test file deletion (via UI delete button)
```

---

## 📊 **Expected Results**

### **Device Discovery:**
- Should find devices within 30-60 seconds
- Manual addition should work immediately
- Test discovery should show device details

### **Transfer Controls:**
- Clear completed should remove 0+ transfers
- Pause/resume should work on active transfers
- Cancel should immediately stop transfers

### **File Operations:**
- Upload should work via drag-and-drop
- Delete should remove files completely
- Download should work for all file types

### **QR Codes:**
- Should generate within 2-3 seconds
- Should work with mobile camera apps
- Test connection should verify accessibility

---

## 🎉 **Success Indicators**

### **✅ Everything Working:**
- Devices appear in "Available Devices" section
- Transfer controls respond immediately
- Files can be deleted from shared files list
- QR codes generate and can be scanned
- Private messaging works between devices

### **🔧 Still Having Issues:**
1. Check server logs for error messages
2. Verify firewall allows port 8081
3. Ensure devices are on same network
4. Try manual device addition first
5. Test with localhost (127.0.0.1) first

---

## 📞 **Support**

If you're still having issues:

1. **Check Server Logs:**
   - Look for error messages
   - Verify scanning is happening
   - Check for connection attempts

2. **Browser Console:**
   - Press F12 to open developer tools
   - Check for JavaScript errors
   - Look for network request failures

3. **Network Configuration:**
   - Ensure firewall allows port 8081
   - Verify devices are on same subnet
   - Test with simple ping between devices

**All major functionality should now be working!** 🚀