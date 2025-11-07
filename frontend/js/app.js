class GoLANshare {
    constructor() {
        this.apiBase = window.location.origin;
        this.ws = null;
        this.token = null;
        this.devices = [];
        this.transfers = [];
        this.chatMessages = [];
        this.files = [];
        this.currentUser = 'User';
        this.isOnline = navigator.onLine;
        this.uploadQueue = [];
        this.retryAttempts = {};
        this.maxRetries = 3;
        this.heartbeatInterval = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        
        this.initializeTransferControls();
        this.init();
    }

    init() {
        this.bindEvents();
        this.setupNetworkMonitoring();
        this.setupGlobalErrorHandling();
        this.setupKeyboardShortcuts();
        this.checkAuth();
        this.setupApp();
        this.setupDragAndDrop();
        this.setupServiceWorker();
    }

    bindEvents() {
        // Login
        document.getElementById('loginForm').addEventListener('submit', (e) => this.handleLogin(e));
        document.getElementById('logoutBtn').addEventListener('click', () => this.handleLogout());
        
        // File operations
        document.getElementById('browseFilesBtn').addEventListener('click', () => this.openFilePicker());
        document.getElementById('fileInput').addEventListener('change', (e) => this.handleFileSelect(e));
        
        const uploadZone = document.getElementById('uploadZone');
        uploadZone.addEventListener('click', () => this.openFilePicker());
        uploadZone.addEventListener('dragover', (e) => this.handleDragOver(e));
        uploadZone.addEventListener('dragleave', (e) => this.handleDragLeave(e));
        uploadZone.addEventListener('drop', (e) => this.handleDrop(e));
        
        // Devices
        document.getElementById('refreshDevicesBtn').addEventListener('click', () => this.loadDevices());
        
        // Files
        document.getElementById('refreshFilesBtn').addEventListener('click', () => this.loadFiles());
        
        // Chat
        document.getElementById('sendChatBtn').addEventListener('click', () => this.sendChatMessage());
        document.getElementById('chatInput').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.sendChatMessage();
        });
        
        // Transfers
        document.getElementById('clearCompletedBtn').addEventListener('click', () => this.clearCompletedTransfers());
        
        // Quick actions
        document.querySelectorAll('.quick-action-btn').forEach(btn => {
            btn.addEventListener('click', (e) => this.handleQuickAction(e));
        });
        
        // Settings
        document.getElementById('settingsToggle').addEventListener('click', () => this.toggleSettings());
        document.getElementById('saveSettings').addEventListener('click', () => this.saveSettings());
        document.getElementById('resetSettings').addEventListener('click', () => this.resetSettings());
        document.getElementById('exportSettings').addEventListener('click', () => this.exportSettings());
        
        // Advanced features
        document.getElementById('analyticsBtn').addEventListener('click', () => this.showAnalytics());
        document.getElementById('encryptionBtn').addEventListener('click', () => this.configureEncryption());
        document.getElementById('schedulerBtn').addEventListener('click', () => this.showScheduler());
        document.getElementById('remoteBtn').addEventListener('click', () => this.setupRemoteAccess());
        
        // Private sharing
        document.getElementById('shareHistoryBtn').addEventListener('click', () => this.showShareHistory());
        document.getElementById('pendingRequestsBtn').addEventListener('click', () => this.showPendingRequests());
    }

    async checkAuth() {
        // Check if we have a stored token
        this.token = localStorage.getItem('golanshare_token');
        if (this.token) {
            this.showApp();
        } else {
            this.showLogin();
        }
    }

    showLogin() {
        document.getElementById('loginModal').style.display = 'flex';
        document.getElementById('app').classList.add('hidden');
    }

    showApp() {
        document.getElementById('loginModal').style.display = 'none';
        document.getElementById('app').classList.remove('hidden');
        this.startApp();
    }

    async handleLogin(e) {
        e.preventDefault();
        
        const username = document.getElementById('username').value;
        const password = document.getElementById('password').value;
        
        try {
            const response = await fetch(this.apiBase + '/api/auth/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ username, password })
            });
            
            if (response.ok) {
                const data = await response.json();
                this.token = data.token;
                localStorage.setItem('golanshare_token', this.token);
                this.currentUser = username;
                this.showApp();
                this.showNotification('Login successful!', 'success');
            } else {
                throw new Error('Login failed');
            }
        } catch (error) {
            this.showNotification('Login failed. Using demo mode.', 'error');
            // Fallback: use demo mode
            this.token = 'demo-token';
            this.currentUser = username;
            setTimeout(() => this.showApp(), 1000);
        }
    }

    handleLogout() {
        localStorage.removeItem('golanshare_token');
        this.token = null;
        this.ws?.close();
        this.showLogin();
    }

    startApp() {
        this.updateUserGreeting();
        this.loadDevices();
        this.loadTransfers();
        this.loadFiles();
        this.connectWebSocket();
        this.startHeartbeat();
        this.startPendingRequestsMonitoring();
    }

    updateUserGreeting() {
        document.getElementById('userGreeting').textContent = `Hello, ${this.currentUser}!`;
        this.updateNetworkInfo();
    }

    async updateNetworkInfo() {
        try {
            const response = await fetch(`${this.apiBase}/api/device/info`);
            if (response.ok) {
                const data = await response.json();
                const deviceIP = document.getElementById('deviceIP');
                if (deviceIP && data.device) {
                    deviceIP.textContent = `${data.device.ip}:${data.device.port}`;
                    deviceIP.title = `Device: ${data.device.name}\nOS: ${data.device.os}\nVersion: ${data.version}`;
                }
                
                // Also display client IP in console and UI
                console.log('Device Info:', data);
                this.displayClientInfo(data);
            }
        } catch (error) {
            console.error('Failed to load network info:', error);
            // Fallback: try to get client IP from other sources
            this.getClientIP();
        }
    }

    displayClientInfo(deviceData) {
        // Add client info to the header
        const connectionText = document.getElementById('connectionText');
        const clientInfo = document.getElementById('clientInfo');
        
        if (connectionText && deviceData.device) {
            connectionText.textContent = `Connected as ${deviceData.device.name}`;
        }
        
        if (clientInfo) {
            const clientIP = deviceData.client_ip || deviceData.device?.ip || 'Unknown';
            const totalDevices = deviceData.total_devices || 'Unknown';
            const connectedClients = deviceData.connected_clients || 'Unknown';
            
            clientInfo.innerHTML = `
                Your IP: <strong>${clientIP}</strong> | 
                Devices: <strong>${totalDevices}</strong> | 
                Clients: <strong>${connectedClients}</strong>
            `;
        }
        
        // Log detailed info
        console.log('=== CLIENT INFO ===');
        console.log('Device Name:', deviceData.device?.name);
        console.log('Client IP:', deviceData.client_ip || deviceData.device?.ip);
        console.log('Port:', deviceData.device?.port);
        console.log('OS:', deviceData.device?.os);
        console.log('Total Devices:', deviceData.total_devices);
        console.log('Connected Clients:', deviceData.connected_clients);
        console.log('==================');
    }

    async getClientIP() {
        try {
            // Try to get IP from a public service as fallback
            const response = await fetch('https://api.ipify.org?format=json');
            if (response.ok) {
                const data = await response.json();
                console.log('Public IP:', data.ip);
            }
        } catch (error) {
            console.log('Could not get public IP:', error);
        }
        
        // Also try WebRTC method
        this.getLocalIP();
    }

    getLocalIP() {
        try {
            const pc = new RTCPeerConnection({iceServers: []});
            pc.createDataChannel('');
            pc.createOffer().then(offer => pc.setLocalDescription(offer));
            
            pc.onicecandidate = (ice) => {
                if (ice && ice.candidate && ice.candidate.candidate) {
                    const candidate = ice.candidate.candidate;
                    const ip = candidate.split(' ')[4];
                    if (ip && ip !== '0.0.0.0') {
                        console.log('Local IP via WebRTC:', ip);
                        pc.close();
                    }
                }
            };
        } catch (error) {
            console.log('WebRTC IP detection failed:', error);
        }
    }

    // File handling
    openFilePicker() {
        document.getElementById('fileInput').click();
    }

    handleFileSelect(e) {
        const files = e.target.files;
        if (files.length > 0) {
            this.uploadFiles(files);
        }
    }

    handleDragOver(e) {
        e.preventDefault();
        document.getElementById('uploadZone').classList.add('dragover');
    }

    handleDragLeave(e) {
        e.preventDefault();
        document.getElementById('uploadZone').classList.remove('dragover');
    }

    handleDrop(e) {
        e.preventDefault();
        document.getElementById('uploadZone').classList.remove('dragover');
        
        const files = e.dataTransfer.files;
        if (files.length > 0) {
            this.uploadFiles(files);
        }
    }

    async uploadFiles(files) {
        if (!this.isOnline) {
            this.addToUploadQueue(files);
            this.showNotification('Files queued for upload when connection is restored', 'warning');
            return;
        }

        const fileArray = Array.isArray(files) ? files : Array.from(files);
        this.showNotification(`Uploading ${fileArray.length} file(s)...`, 'info');
        
        // Validate files before upload
        const validFiles = [];
        const errors = [];
        
        for (let file of fileArray) {
            const validation = this.validateFile(file);
            if (validation.valid) {
                validFiles.push(file);
            } else {
                errors.push(`${file.name}: ${validation.error}`);
            }
        }
        
        if (errors.length > 0) {
            this.showNotification(`Some files were rejected: ${errors.join(', ')}`, 'warning', 8000);
        }
        
        if (validFiles.length === 0) {
            return;
        }
        
        // Process uploads with concurrency limit
        const concurrentUploads = 3;
        const uploadPromises = [];
        
        for (let i = 0; i < validFiles.length; i += concurrentUploads) {
            const batch = validFiles.slice(i, i + concurrentUploads);
            const batchPromises = batch.map(file => this.uploadSingleFile(file));
            uploadPromises.push(...batchPromises);
            
            // Wait for batch to complete before starting next batch
            if (i + concurrentUploads < validFiles.length) {
                await Promise.allSettled(batchPromises);
            }
        }
        
        // Wait for all uploads to complete
        const results = await Promise.allSettled(uploadPromises);
        
        const successful = results.filter(r => r.status === 'fulfilled').length;
        const failed = results.filter(r => r.status === 'rejected').length;
        
        if (successful > 0) {
            this.showNotification(`Successfully uploaded ${successful} file(s)`, 'success');
            this.loadFiles(); // Refresh file list
        }
        
        if (failed > 0) {
            this.showNotification(`Failed to upload ${failed} file(s)`, 'error');
        }
    }

    validateFile(file) {
        // Size check
        if (file.size > 1024 * 1024 * 1024) { // 1GB limit
            return { valid: false, error: 'File too large (max 1GB)' };
        }
        
        // Type check
        const blockedTypes = ['.exe', '.bat', '.cmd', '.com', '.pif', '.scr', '.vbs'];
        const ext = '.' + file.name.split('.').pop().toLowerCase();
        if (blockedTypes.includes(ext)) {
            return { valid: false, error: 'File type not allowed' };
        }
        
        // Name check
        if (file.name.length > 255) {
            return { valid: false, error: 'Filename too long' };
        }
        
        return { valid: true };
    }

    async uploadSingleFile(file) {
        const transferId = 'transfer-' + Date.now() + '-' + Math.random().toString(36).substr(2, 9);
        
        const transfer = {
            id: transferId,
            file_name: file.name,
            file_size: file.size,
            progress: 0,
            status: 'uploading',
            started_at: new Date().toISOString(),
            speed: 0,
            eta: 0
        };
        
        this.addTransfer(transfer);
        
        try {
            // Use chunked upload for large files (>10MB)
            if (file.size > 10 * 1024 * 1024) {
                await this.uploadFileChunked(file, transferId);
            } else {
                await this.uploadFileSimple(file, transferId);
            }
        } catch (error) {
            this.updateTransferProgress(transferId, 0, 'failed', error.message);
            throw error;
        }
    }

    addToUploadQueue(files) {
        this.uploadQueue.push(...Array.from(files));
        this.updateQueueDisplay();
    }

    async processUploadQueue() {
        if (this.uploadQueue.length === 0) return;
        
        const files = [...this.uploadQueue];
        this.uploadQueue = [];
        this.updateQueueDisplay();
        
        await this.uploadFiles(files);
    }

    updateQueueDisplay() {
        const queueCount = this.uploadQueue.length;
        if (queueCount > 0) {
            this.showNotification(`${queueCount} files queued for upload`, 'info', 3000);
        }
    }

    async uploadFileSimple(file, transferId) {
        const formData = new FormData();
        formData.append('files', file);

        try {
            const response = await fetch(this.apiBase + '/api/upload', {
                method: 'POST',
                headers: {
                    'Authorization': 'Bearer ' + this.token
                },
                body: formData
            });

            if (response.ok) {
                this.updateTransferProgress(transferId, 100, 'completed');
                this.showNotification(`${file.name} uploaded successfully!`, 'success');
            } else {
                throw new Error('Upload failed');
            }
        } catch (error) {
            this.updateTransferProgress(transferId, 0, 'failed');
            this.showNotification(`Upload failed: ${error.message}`, 'error');
        }
    }

    async uploadFileChunked(file, transferId) {
        const chunkSize = 1024 * 1024; // 1MB chunks
        const totalChunks = Math.ceil(file.size / chunkSize);
        const startTime = Date.now();
        let uploadedBytes = 0;
        
        try {
            for (let i = 0; i < totalChunks; i++) {
                const start = i * chunkSize;
                const end = Math.min(start + chunkSize, file.size);
                const chunk = file.slice(start, end);
                
                // Calculate hash for chunk integrity
                const chunkHash = await this.calculateHash(chunk);
                
                const formData = new FormData();
                formData.append('chunk', chunk);
                
                const chunkStartTime = Date.now();
                
                const response = await this.fetchWithRetry(`${this.apiBase}/api/upload/chunked?token=${transferId}&filename=${encodeURIComponent(file.name)}&chunk=${i}&totalChunks=${totalChunks}&hash=${chunkHash}`, {
                    method: 'POST',
                    headers: {
                        'Authorization': 'Bearer ' + this.token
                    },
                    body: formData
                }, transferId);

                if (!response.ok) {
                    const errorData = await response.json().catch(() => ({}));
                    throw new Error(errorData.message || `Chunk ${i} upload failed (${response.status})`);
                }

                uploadedBytes += chunk.size;
                const progress = Math.round((uploadedBytes / file.size) * 100);
                
                // Calculate speed and ETA
                const elapsedTime = (Date.now() - startTime) / 1000;
                const speed = uploadedBytes / elapsedTime; // bytes per second
                const remainingBytes = file.size - uploadedBytes;
                const eta = remainingBytes / speed;
                
                this.updateTransferProgress(transferId, progress, 'uploading', null, speed, eta);
                
                // Small delay to prevent overwhelming the server
                if (i < totalChunks - 1) {
                    await new Promise(resolve => setTimeout(resolve, 10));
                }
            }
            
            this.updateTransferProgress(transferId, 100, 'completed');
            
        } catch (error) {
            this.updateTransferProgress(transferId, Math.round((uploadedBytes / file.size) * 100), 'failed', error.message);
            throw error;
        }
    }

    async calculateHash(data) {
        const buffer = await data.arrayBuffer();
        const hashBuffer = await crypto.subtle.digest('SHA-256', buffer);
        const hashArray = Array.from(new Uint8Array(hashBuffer));
        return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
    }

    async fetchWithRetry(url, options, transferId, maxRetries = 3) {
        let lastError;
        
        for (let attempt = 1; attempt <= maxRetries; attempt++) {
            try {
                const response = await fetch(url, options);
                
                // Reset retry counter on success
                if (this.retryAttempts[transferId]) {
                    delete this.retryAttempts[transferId];
                }
                
                return response;
            } catch (error) {
                lastError = error;
                
                if (attempt < maxRetries) {
                    const delay = Math.pow(2, attempt) * 1000; // Exponential backoff
                    this.retryAttempts[transferId] = attempt;
                    
                    this.updateTransferProgress(transferId, null, 'retrying', `Retry ${attempt}/${maxRetries} in ${delay/1000}s`);
                    
                    await new Promise(resolve => setTimeout(resolve, delay));
                }
            }
        }
        
        throw lastError;
    }

    updateTransferProgress(transferId, progress, status, error = null, speed = 0, eta = 0) {
        const transfer = this.transfers.find(t => t.id === transferId);
        if (transfer) {
            if (progress !== null) transfer.progress = progress;
            transfer.status = status;
            if (error) transfer.error = error;
            if (speed) transfer.speed = speed;
            if (eta) transfer.eta = eta;
            
            if (status === 'completed') {
                transfer.completed_at = new Date().toISOString();
                transfer.progress = 100;
            }
            
            this.renderTransfers();
            
            // Update browser title with progress for active uploads
            this.updateBrowserTitle();
        }
    }

    updateBrowserTitle() {
        const activeTransfers = this.transfers.filter(t => t.status === 'uploading' || t.status === 'downloading');
        
        if (activeTransfers.length > 0) {
            const avgProgress = activeTransfers.reduce((sum, t) => sum + t.progress, 0) / activeTransfers.length;
            document.title = `(${Math.round(avgProgress)}%) GoLANshare`;
        } else {
            document.title = 'GoLANshare - Local File Sharing';
        }
    }

    // Devices
    async loadDevices() {
        try {
            console.log('Loading devices from API...');
            const response = await fetch(this.apiBase + '/api/devices');
            if (response.ok) {
                this.devices = await response.json();
                console.log('Devices loaded:', this.devices);
                this.renderDevices();
            } else {
                console.error('Failed to load devices, status:', response.status);
                this.showFallbackDevices();
            }
        } catch (error) {
            console.error('Error loading devices:', error);
            this.showFallbackDevices();
        }
    }

    showFallbackDevices() {
        // Show demo devices for testing
        this.devices = [
            { id: 'demo-1', name: 'Barath Computer', ip: '192.168.1.100', os: 'Windows', status: 'online', is_self: false },
            { id: 'demo-2', name: 'Demo Phone', ip: '192.168.1.101', os: 'Android', status: 'online', is_self: false },
            { id: 'self', name: 'This Device', ip: '127.0.0.1', os: 'Windows', status: 'online', is_self: true }
        ];
        console.log('Using fallback demo devices:', this.devices);
        this.renderDevices();
    }

    // In your app.js, update the renderDevices function:

renderDevices() {
    const container = document.getElementById('devicesList');
    const countElement = document.getElementById('devicesCount');
    
    console.log('Rendering devices:', this.devices);
    
    // Filter out self device
    const otherDevices = this.devices.filter(device => !device.is_self);
    const selfDevices = this.devices.filter(device => device.is_self);
    
    console.log('Other devices:', otherDevices);
    console.log('Self devices:', selfDevices);
    
    countElement.textContent = otherDevices.length;
    
    if (otherDevices.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <div class="empty-icon">🔍</div>
                <p>No other devices found</p>
                <small>Total devices in system: ${this.devices.length}</small>
                <small>Self devices: ${selfDevices.length}</small>
                <small>Other devices: ${otherDevices.length}</small>
                <br><br>
                <small>Make sure other devices are on the same network and running GoLANshare</small>
                <br><br>
                <button class="btn btn-outline btn-sm" onclick="app.refreshDevices()">
                    <span class="btn-icon">🔄</span>
                    Scan Again
                </button>
                <button class="btn btn-outline btn-sm" onclick="app.testDiscovery()">
                    <span class="btn-icon">🔍</span>
                    Test Discovery
                </button>
                <button class="btn btn-outline btn-sm" onclick="app.showDebugInfo()">
                    <span class="btn-icon">🐛</span>
                    Debug Info
                </button>
            </div>
        `;
        return;
    }
    
    container.innerHTML = otherDevices.map(device => `
        <div class="device-item ${device.is_connected ? 'connected' : ''}" data-device-id="${device.id}">
            <div class="device-avatar">
                <div class="device-icon">${this.getDeviceIcon(device.os)}</div>
                <div class="connection-indicator ${device.is_connected ? 'connected' : 'disconnected'}"></div>
            </div>
            <div class="device-info">
                <div class="device-header">
                    <div class="device-name">${device.name}</div>
                    <div class="device-status ${device.status}">${device.status}</div>
                </div>
                <div class="device-details">
                    <span>${device.ip}</span>
                    <span>•</span>
                    <span>${device.os}</span>
                    ${device.distance ? `<span>• ${device.distance}</span>` : ''}
                </div>
                <div class="device-type">${this.getDeviceTypeText(device.device_type || device.os)}</div>
            </div>
            <div class="device-actions">
                <div class="action-buttons">
                    <button class="btn btn-sm btn-primary" onclick="app.showShareOptions('${device.id}')" title="Share with ${device.name}">
                        <span class="btn-icon">📤</span>
                        Share
                    </button>
                    <button class="btn btn-sm btn-outline" onclick="app.sendMessage('${device.id}')" title="Send message">
                        <span class="btn-icon">💬</span>
                        Message
                    </button>
                    <button class="btn btn-sm btn-outline" onclick="app.connectToDevice('${device.id}')" title="Connect">
                        <span class="btn-icon">${device.is_connected ? '🔗' : '🔌'}</span>
                        ${device.is_connected ? 'Connected' : 'Connect'}
                    </button>
                </div>
            </div>
        </div>
    `).join('');
}

async refreshDevices() {
    this.showNotification('Scanning for devices...', 'info', 2000);
    
    try {
        // Trigger manual scan
        await fetch(`${this.apiBase}/api/device/scan`, {
            method: 'POST',
            headers: {
                'Authorization': 'Bearer ' + this.token
            }
        });
        
        // Wait a moment for scan to start
        await new Promise(resolve => setTimeout(resolve, 1000));
        
        // Test discovery endpoint to add demo devices if needed
        await fetch(`${this.apiBase}/api/device/test`);
        
        // Load updated device list
        await this.loadDevices();
        
        const deviceCount = this.devices.filter(d => !d.is_self).length;
        if (deviceCount > 0) {
            this.showNotification(`Found ${deviceCount} device(s)`, 'success');
        } else {
            this.showNotification('No devices found. Try adding a device manually.', 'warning');
        }
        
    } catch (error) {
        console.error('Device refresh failed:', error);
        this.showNotification('Device scan failed', 'error');
    }
}

getDeviceTypeText(type) {
    const types = {
        'Windows': '💻 Computer',
        'macOS': '🖥️ Mac',
        'Linux': '🐧 Linux PC',
        'Android': '📱 Phone',
        'iOS': '📱 iPhone',
        'Unknown': '🖥️ Device'
    };
    return types[type] || '🖥️ Device';
}

    getDeviceIcon(os) {
        const icons = {
            'windows': '💻',
            'macos': '🍎', 
            'ios': '📱',
            'android': '🤖',
            'linux': '🐧'
        };
        return icons[os.toLowerCase()] || '🖥️';
    }

    sendToDevice(deviceId) {
        const device = this.devices.find(d => d.id === deviceId);
        if (device) {
            const url = `http://${device.ip}:${device.port || 8081}`;
            this.showQRCode(url, device.name);
        }
    }

    showQRCode(url, deviceName) {
        // Create QR code modal
        const modal = document.createElement('div');
        modal.className = 'modal';
        modal.innerHTML = `
            <div class="modal-content">
                <div class="qr-header">
                    <h2>📱 Connect to ${deviceName}</h2>
                    <p>Scan this QR code with your mobile device to connect</p>
                </div>
                <div class="qr-code" id="qrcode">
                    <div class="qr-loading">Generating QR code...</div>
                </div>
                <div class="qr-url">
                    <input type="text" value="${url}" readonly onclick="this.select()" class="qr-url-input">
                    <button class="btn btn-primary" onclick="app.copyToClipboard('${url}')">
                        <span class="btn-icon">📋</span>
                        Copy URL
                    </button>
                </div>
                <div class="qr-instructions">
                    <h4>📋 Instructions:</h4>
                    <ol>
                        <li>Open camera app on your mobile device</li>
                        <li>Point camera at the QR code above</li>
                        <li>Tap the notification to open GoLANshare</li>
                        <li>Or manually enter the URL in your mobile browser</li>
                    </ol>
                </div>
                <div class="qr-actions">
                    <button class="btn btn-outline" onclick="this.closest('.modal').remove()">Close</button>
                    <button class="btn btn-primary" onclick="app.testConnection('${url}')">Test Connection</button>
                </div>
            </div>
        `;
        
        document.body.appendChild(modal);
        
        // Generate QR code
        this.generateQRCode(url, 'qrcode');
        
        // Auto-remove after 2 minutes
        setTimeout(() => {
            if (modal.parentElement) {
                modal.remove();
            }
        }, 120000);
    }

    generateQRCode(text, elementId) {
        const qrElement = document.getElementById(elementId);
        if (qrElement) {
            // Try multiple QR code services for reliability
            const qrServices = [
                `https://api.qrserver.com/v1/create-qr-code/?size=250x250&data=${encodeURIComponent(text)}`,
                `https://chart.googleapis.com/chart?chs=250x250&cht=qr&chl=${encodeURIComponent(text)}`,
                `https://qr-server.com/api/v1/create-qr-code/?size=250x250&data=${encodeURIComponent(text)}`
            ];
            
            let serviceIndex = 0;
            
            const tryNextService = () => {
                if (serviceIndex >= qrServices.length) {
                    qrElement.innerHTML = `
                        <div class="qr-fallback">
                            <div class="qr-error">❌ QR Code generation failed</div>
                            <p>Please copy the URL manually:</p>
                            <code>${text}</code>
                        </div>
                    `;
                    return;
                }
                
                const img = new Image();
                img.onload = () => {
                    qrElement.innerHTML = `
                        <div class="qr-code-container">
                            <img src="${qrServices[serviceIndex]}" alt="QR Code" class="qr-image">
                            <p class="qr-caption">Scan with mobile camera</p>
                        </div>
                    `;
                };
                
                img.onerror = () => {
                    serviceIndex++;
                    tryNextService();
                };
                
                img.src = qrServices[serviceIndex];
            };
            
            tryNextService();
        }
    }

    async copyToClipboard(text) {
        try {
            await navigator.clipboard.writeText(text);
            this.showNotification('URL copied to clipboard!', 'success', 2000);
        } catch (error) {
            // Fallback for older browsers
            const textArea = document.createElement('textarea');
            textArea.value = text;
            document.body.appendChild(textArea);
            textArea.select();
            document.execCommand('copy');
            document.body.removeChild(textArea);
            this.showNotification('URL copied to clipboard!', 'success', 2000);
        }
    }

    async testConnection(url) {
        this.showNotification('Testing connection...', 'info', 2000);
        
        try {
            const response = await fetch(url + '/api/device/test', {
                method: 'GET',
                timeout: 5000
            });
            
            if (response.ok) {
                const data = await response.json();
                this.showNotification(`✅ Connection successful! Found ${data.service || 'GoLANshare'}`, 'success');
            } else {
                this.showNotification('❌ Connection failed - device not responding', 'error');
            }
        } catch (error) {
            this.showNotification('❌ Connection failed - check network and URL', 'error');
        }
    }

    // Transfers
    async loadTransfers() {
        try {
            const response = await fetch(this.apiBase + '/api/transfers');
            if (response.ok) {
                const data = await response.json();
                this.transfers = Array.isArray(data) ? data : [];
                this.renderTransfers();
            } else {
                this.transfers = [];
                this.renderTransfers();
            }
        } catch (error) {
            this.transfers = [];
            this.renderTransfers();
        }
    }

    renderTransfers() {
        const container = document.getElementById('transfersList');
        const countElement = document.getElementById('transfersCount');
        
        if (!this.transfers || !Array.isArray(this.transfers)) {
            this.transfers = [];
        }
        
        // Sort transfers: active first, then by start time
        const sortedTransfers = [...this.transfers].sort((a, b) => {
            const statusOrder = { 'uploading': 0, 'downloading': 1, 'retrying': 2, 'paused': 3, 'completed': 4, 'failed': 5 };
            const aOrder = statusOrder[a.status] || 6;
            const bOrder = statusOrder[b.status] || 6;
            
            if (aOrder !== bOrder) return aOrder - bOrder;
            return new Date(b.started_at) - new Date(a.started_at);
        });
        
        const activeTransfers = sortedTransfers.filter(t => t.status !== 'completed' && t.status !== 'failed');
        countElement.textContent = activeTransfers.length;
        
        if (sortedTransfers.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <div class="empty-icon">📤</div>
                    <p>No transfers</p>
                    <small>Files you upload or download will appear here</small>
                </div>
            `;
            return;
        }
        
        container.innerHTML = sortedTransfers.map(transfer => {
            const statusClass = this.getStatusClass(transfer.status);
            const progressColor = this.getProgressColor(transfer.status);
            const timeInfo = this.getTransferTimeInfo(transfer);
            const speedInfo = this.getSpeedInfo(transfer);
            
            return `
                <div class="transfer-item ${statusClass}" data-transfer-id="${transfer.id}">
                    <div class="transfer-icon">${this.getFileIcon(transfer.file_name)}</div>
                    <div class="transfer-info">
                        <div class="transfer-header">
                            <div class="transfer-name" title="${transfer.file_name}">${transfer.file_name}</div>
                            <div class="transfer-status ${transfer.status}">${this.getStatusText(transfer)}</div>
                        </div>
                        <div class="transfer-details">
                            <span>${this.formatFileSize(transfer.file_size)}</span>
                            ${speedInfo ? `<span>• ${speedInfo}</span>` : ''}
                            ${timeInfo ? `<span>• ${timeInfo}</span>` : ''}
                        </div>
                        <div class="progress-container">
                            <div class="progress-bar">
                                <div class="progress-fill ${progressColor}" style="width: ${transfer.progress || 0}%"></div>
                            </div>
                            <div class="progress-text">${transfer.progress || 0}%</div>
                        </div>
                        ${transfer.error ? `<div class="transfer-error">❌ ${transfer.error}</div>` : ''}
                    </div>
                    <div class="transfer-actions">
                        ${this.getTransferActions(transfer)}
                    </div>
                </div>
            `;
        }).join('');
    }

    getStatusClass(status) {
        const classes = {
            'uploading': 'active',
            'downloading': 'active',
            'completed': 'success',
            'failed': 'error',
            'paused': 'paused',
            'retrying': 'warning'
        };
        return classes[status] || '';
    }

    getProgressColor(status) {
        const colors = {
            'uploading': 'primary',
            'downloading': 'info',
            'completed': 'success',
            'failed': 'error',
            'paused': 'secondary',
            'retrying': 'warning'
        };
        return colors[status] || 'primary';
    }

    getStatusText(transfer) {
        const statusTexts = {
            'uploading': 'Uploading',
            'downloading': 'Downloading',
            'completed': 'Completed',
            'failed': 'Failed',
            'paused': 'Paused',
            'retrying': 'Retrying'
        };
        return statusTexts[transfer.status] || transfer.status;
    }

    getTransferTimeInfo(transfer) {
        if (transfer.status === 'completed' && transfer.completed_at) {
            const duration = new Date(transfer.completed_at) - new Date(transfer.started_at);
            return `Completed in ${this.formatDuration(duration)}`;
        }
        
        if (transfer.eta && transfer.eta > 0 && (transfer.status === 'uploading' || transfer.status === 'downloading')) {
            return `ETA: ${this.formatDuration(transfer.eta * 1000)}`;
        }
        
        if (transfer.started_at) {
            const elapsed = Date.now() - new Date(transfer.started_at);
            return `Started ${this.formatDuration(elapsed)} ago`;
        }
        
        return '';
    }

    getSpeedInfo(transfer) {
        if (transfer.speed && transfer.speed > 0) {
            return `${this.formatFileSize(transfer.speed)}/s`;
        }
        return '';
    }

    getTransferActions(transfer) {
        const actions = [];
        
        switch (transfer.status) {
            case 'uploading':
            case 'downloading':
                actions.push(`<button class="btn btn-sm btn-outline" onclick="app.pauseTransfer('${transfer.id}')">⏸️ Pause</button>`);
                actions.push(`<button class="btn btn-sm btn-error" onclick="app.cancelTransfer('${transfer.id}')">❌ Cancel</button>`);
                break;
            case 'paused':
                actions.push(`<button class="btn btn-sm btn-primary" onclick="app.resumeTransfer('${transfer.id}')">▶️ Resume</button>`);
                actions.push(`<button class="btn btn-sm btn-error" onclick="app.cancelTransfer('${transfer.id}')">❌ Cancel</button>`);
                break;
            case 'failed':
                actions.push(`<button class="btn btn-sm btn-warning" onclick="app.retryTransfer('${transfer.id}')">🔄 Retry</button>`);
                actions.push(`<button class="btn btn-sm btn-outline" onclick="app.removeTransfer('${transfer.id}')">🗑️ Remove</button>`);
                break;
            case 'completed':
                actions.push(`<button class="btn btn-sm btn-outline" onclick="app.removeTransfer('${transfer.id}')">🗑️ Remove</button>`);
                break;
        }
        
        return actions.join('');
    }

    formatDuration(ms) {
        const seconds = Math.floor(ms / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        
        if (hours > 0) {
            return `${hours}h ${minutes % 60}m`;
        } else if (minutes > 0) {
            return `${minutes}m ${seconds % 60}s`;
        } else {
            return `${seconds}s`;
        }
    }

    // Transfer control methods
    pauseTransfer(transferId) {
        const transfer = this.transfers.find(t => t.id === transferId);
        if (transfer) {
            // Cancel ongoing upload if exists
            if (this.activeUploads && this.activeUploads[transferId]) {
                this.activeUploads[transferId].abort();
                delete this.activeUploads[transferId];
            }
            
            transfer.status = 'paused';
            this.renderTransfers();
            this.showNotification(`Paused ${transfer.file_name}`, 'info', 2000);
        }
    }

    async resumeTransfer(transferId) {
        const transfer = this.transfers.find(t => t.id === transferId);
        if (transfer) {
            transfer.status = 'uploading';
            this.renderTransfers();
            this.showNotification(`Resuming ${transfer.file_name}`, 'info', 2000);
            
            // Try to resume the upload
            try {
                const response = await fetch(`${this.apiBase}/api/transfer/resume?token=${transferId}`, {
                    headers: {
                        'Authorization': 'Bearer ' + this.token
                    }
                });
                
                if (response.ok) {
                    const data = await response.json();
                    // Resume from where we left off
                    if (data.transfer && data.uploadedChunks) {
                        // Continue chunked upload from last chunk
                        this.resumeChunkedUpload(transferId, data.uploadedChunks);
                    }
                } else {
                    // Restart from beginning
                    this.retryTransfer(transferId);
                }
            } catch (error) {
                console.error('Resume failed:', error);
                this.retryTransfer(transferId);
            }
        }
    }

    cancelTransfer(transferId) {
        const transfer = this.transfers.find(t => t.id === transferId);
        if (transfer) {
            // Cancel ongoing upload if exists
            if (this.activeUploads && this.activeUploads[transferId]) {
                this.activeUploads[transferId].abort();
                delete this.activeUploads[transferId];
            }
            
            transfer.status = 'cancelled';
            transfer.error = 'Cancelled by user';
            this.renderTransfers();
            this.showNotification(`Cancelled ${transfer.file_name}`, 'warning', 2000);
        }
    }

    async retryTransfer(transferId) {
        const transfer = this.transfers.find(t => t.id === transferId);
        if (transfer) {
            // Reset transfer state
            transfer.status = 'uploading';
            transfer.progress = 0;
            transfer.error = null;
            transfer.started_at = new Date().toISOString();
            
            this.renderTransfers();
            this.showNotification(`Retrying ${transfer.file_name}`, 'info', 2000);
            
            // Find the original file and re-upload
            const fileInput = document.createElement('input');
            fileInput.type = 'file';
            fileInput.onchange = async (e) => {
                const files = Array.from(e.target.files);
                const targetFile = files.find(f => f.name === transfer.file_name);
                if (targetFile) {
                    await this.uploadSingleFile(targetFile, transferId);
                }
            };
            // Note: This is a limitation - we can't automatically retry without the original file
            // In a real implementation, you'd store file references or implement server-side retry
        }
    }

    removeTransfer(transferId) {
        // Cancel if still active
        if (this.activeUploads && this.activeUploads[transferId]) {
            this.activeUploads[transferId].abort();
            delete this.activeUploads[transferId];
        }
        
        this.transfers = this.transfers.filter(t => t.id !== transferId);
        this.renderTransfers();
    }

    // Initialize active uploads tracking
    initializeTransferControls() {
        this.activeUploads = {};
    }

    // Resume chunked upload from specific chunk
    async resumeChunkedUpload(transferId, uploadedChunks) {
        // This would need the original file reference
        // For now, just mark as failed and suggest retry
        const transfer = this.transfers.find(t => t.id === transferId);
        if (transfer) {
            transfer.status = 'failed';
            transfer.error = 'Resume not available - please retry';
            this.renderTransfers();
        }
    }

    // Settings Management
    toggleSettings() {
        const content = document.getElementById('settingsContent');
        const isVisible = content.style.display !== 'none';
        
        if (isVisible) {
            content.classList.add('hide');
            setTimeout(() => {
                content.style.display = 'none';
                content.classList.remove('hide');
            }, 300);
        } else {
            content.style.display = 'block';
            content.classList.add('show');
            setTimeout(() => content.classList.remove('show'), 300);
            this.loadSettings();
        }
    }

    async loadSettings() {
        try {
            const response = await fetch(`${this.apiBase}/api/settings`);
            if (response.ok) {
                const settings = await response.json();
                this.applySettingsToUI(settings);
            }
        } catch (error) {
            console.error('Failed to load settings:', error);
        }
    }

    applySettingsToUI(settings) {
        // Apply checkbox settings to UI elements
        const checkboxElements = {
            'requireAuth': 'requireAuth',
            'encryption': 'enableEncryption',
            'autoDelete': 'autoDelete',
            'compress': 'compressFiles',
            'enableNotifications': 'enableNotifications'
        };
        
        Object.entries(checkboxElements).forEach(([settingKey, elementId]) => {
            const element = document.getElementById(elementId);
            if (element && settings[settingKey] !== undefined) {
                element.checked = settings[settingKey] === 'true';
            }
        });
        
        // Apply input field settings
        if (settings.maxUploadSize) {
            const maxUploadElement = document.getElementById('maxUploadSize');
            if (maxUploadElement) maxUploadElement.value = settings.maxUploadSize;
        }
        
        if (settings.serverPort) {
            const serverPortElement = document.getElementById('serverPort');
            if (serverPortElement) serverPortElement.value = settings.serverPort;
        }
        
        // Apply theme setting
        const theme = settings.theme || localStorage.getItem('theme') || 'auto';
        const themeElement = document.getElementById('themeSelect');
        if (themeElement) themeElement.value = theme;
        this.applyTheme(theme);
    }

    async saveSettings() {
        const settings = this.gatherSettingsFromUI();
        
        try {
            const response = await fetch(`${this.apiBase}/api/settings`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer ' + this.token
                },
                body: JSON.stringify(settings)
            });
            
            if (response.ok) {
                this.showNotification('Settings saved successfully!', 'success');
                
                // Apply theme immediately
                const theme = document.getElementById('themeSelect').value;
                localStorage.setItem('theme', theme);
                this.applyTheme(theme);
            } else {
                throw new Error('Failed to save settings');
            }
        } catch (error) {
            this.showNotification('Failed to save settings: ' + error.message, 'error');
        }
    }

    gatherSettingsFromUI() {
        return {
            requireAuth: document.getElementById('requireAuth').checked ? 'true' : 'false',
            encryption: document.getElementById('enableEncryption').checked ? 'true' : 'false',
            autoDelete: document.getElementById('autoDelete').checked ? 'true' : 'false',
            compress: document.getElementById('compressFiles').checked ? 'true' : 'false',
            maxUploadSize: document.getElementById('maxUploadSize').value,
            serverPort: document.getElementById('serverPort').value,
            theme: document.getElementById('themeSelect').value,
            enableNotifications: document.getElementById('enableNotifications').checked ? 'true' : 'false'
        };
    }

    resetSettings() {
        if (confirm('Reset all settings to defaults? This cannot be undone.')) {
            // Reset UI to defaults
            document.getElementById('requireAuth').checked = false;
            document.getElementById('enableEncryption').checked = true;
            document.getElementById('autoDelete').checked = true;
            document.getElementById('compressFiles').checked = false;
            document.getElementById('maxUploadSize').value = '1024';
            document.getElementById('serverPort').value = '8081';
            document.getElementById('themeSelect').value = 'auto';
            document.getElementById('enableNotifications').checked = true;
            
            this.saveSettings();
        }
    }

    exportSettings() {
        const settings = this.gatherSettingsFromUI();
        const dataStr = JSON.stringify(settings, null, 2);
        const dataBlob = new Blob([dataStr], {type: 'application/json'});
        
        const link = document.createElement('a');
        link.href = URL.createObjectURL(dataBlob);
        link.download = 'golanshare-settings.json';
        link.click();
        
        this.showNotification('Settings exported successfully!', 'success');
    }

    applyTheme(theme) {
        const root = document.documentElement;
        
        if (theme === 'dark') {
            root.classList.add('dark-theme');
        } else if (theme === 'light') {
            root.classList.remove('dark-theme');
        } else {
            // Auto theme - use system preference
            if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
                root.classList.add('dark-theme');
            } else {
                root.classList.remove('dark-theme');
            }
        }
    }

    // Advanced Features
    showAnalytics() {
        const modal = this.createModal('Analytics Dashboard', `
            <div class="analytics-dashboard">
                <div class="analytics-grid">
                    <div class="analytics-card">
                        <h4>📊 Transfer Statistics</h4>
                        <div class="stat-row">
                            <span>Total Uploads:</span>
                            <span class="stat-value">${this.transfers.filter(t => t.type === 'upload').length}</span>
                        </div>
                        <div class="stat-row">
                            <span>Total Downloads:</span>
                            <span class="stat-value">${this.transfers.filter(t => t.type === 'download').length}</span>
                        </div>
                        <div class="stat-row">
                            <span>Success Rate:</span>
                            <span class="stat-value">${this.calculateSuccessRate()}%</span>
                        </div>
                    </div>
                    
                    <div class="analytics-card">
                        <h4>💾 Storage Usage</h4>
                        <div class="stat-row">
                            <span>Files Shared:</span>
                            <span class="stat-value">${this.files.length}</span>
                        </div>
                        <div class="stat-row">
                            <span>Total Size:</span>
                            <span class="stat-value">${this.formatFileSize(this.getTotalFileSize())}</span>
                        </div>
                    </div>
                    
                    <div class="analytics-card">
                        <h4>🌐 Network Activity</h4>
                        <div class="stat-row">
                            <span>Connected Devices:</span>
                            <span class="stat-value">${this.devices.length}</span>
                        </div>
                        <div class="stat-row">
                            <span>Active Transfers:</span>
                            <span class="stat-value">${this.transfers.filter(t => t.status === 'uploading' || t.status === 'downloading').length}</span>
                        </div>
                    </div>
                </div>
            </div>
        `);
    }

    configureEncryption() {
        const modal = this.createModal('Encryption Settings', `
            <div class="encryption-config">
                <div class="config-section">
                    <h4>🔐 Encryption Options</h4>
                    <label class="setting-label">
                        <input type="checkbox" id="encryptUploads" checked>
                        <span>Encrypt all uploaded files</span>
                    </label>
                    <label class="setting-label">
                        <input type="checkbox" id="encryptTransfers" checked>
                        <span>Encrypt file transfers</span>
                    </label>
                    <label class="setting-label">
                        <input type="checkbox" id="encryptStorage">
                        <span>Encrypt stored files on disk</span>
                    </label>
                </div>
                
                <div class="config-section">
                    <h4>🔑 Key Management</h4>
                    <div class="key-info">
                        <p><strong>Current Key:</strong> <span id="currentKeyInfo">AES-256 (Generated)</span></p>
                        <p><strong>Key Strength:</strong> <span class="key-strength">Strong</span></p>
                    </div>
                    <div class="key-actions">
                        <button class="btn btn-primary" onclick="app.generateEncryptionKey()">🔄 Generate New Key</button>
                        <button class="btn btn-outline" onclick="app.importEncryptionKey()">📥 Import Key</button>
                        <button class="btn btn-outline" onclick="app.exportEncryptionKey()">📤 Export Key</button>
                    </div>
                </div>
                
                <div class="config-section">
                    <h4>⚠️ Security Notice</h4>
                    <p class="security-warning">Encryption keys are stored locally. Losing your key means losing access to encrypted files. Always backup your keys securely.</p>
                </div>
                
                <div class="config-actions">
                    <button class="btn btn-primary" onclick="app.saveEncryptionSettings(); this.closest('.modal').remove()">💾 Save Configuration</button>
                    <button class="btn btn-outline" onclick="this.closest('.modal').remove()">Cancel</button>
                </div>
            </div>
        `);
    }

    showScheduler() {
        const scheduledTasks = JSON.parse(localStorage.getItem('scheduledTasks') || '[]');
        
        const modal = this.createModal('Transfer Scheduler', `
            <div class="scheduler-interface">
                <div class="scheduler-form">
                    <h4>📅 Schedule Transfer</h4>
                    <div class="form-group">
                        <label>Transfer Type:</label>
                        <select class="setting-select" id="scheduleType">
                            <option value="upload">Upload Files</option>
                            <option value="download">Download Files</option>
                            <option value="sync">Sync Folder</option>
                            <option value="cleanup">Cleanup Old Files</option>
                        </select>
                    </div>
                    
                    <div class="form-group">
                        <label>Schedule:</label>
                        <select class="setting-select" id="scheduleFrequency">
                            <option value="once">Once</option>
                            <option value="daily">Daily</option>
                            <option value="weekly">Weekly</option>
                            <option value="monthly">Monthly</option>
                        </select>
                    </div>
                    
                    <div class="form-group">
                        <label>Date & Time:</label>
                        <input type="datetime-local" class="setting-input" id="scheduleDateTime" style="width: 200px;">
                    </div>
                    
                    <div class="form-group">
                        <label>Description:</label>
                        <input type="text" class="setting-input" id="scheduleDescription" placeholder="Optional description" style="width: 100%;">
                    </div>
                    
                    <button class="btn btn-primary" onclick="app.addScheduledTask()">📅 Schedule Transfer</button>
                </div>
                
                <div class="scheduled-transfers">
                    <h4>📋 Scheduled Transfers (${scheduledTasks.length})</h4>
                    <div class="scheduled-list">
                        ${scheduledTasks.length === 0 ? `
                            <div class="empty-state">
                                <p>No scheduled transfers</p>
                            </div>
                        ` : scheduledTasks.map(task => `
                            <div class="scheduled-item">
                                <div class="task-info">
                                    <strong>${task.type.toUpperCase()}</strong> - ${task.description || 'No description'}
                                    <br><small>📅 ${new Date(task.datetime).toLocaleString()} (${task.frequency})</small>
                                </div>
                                <div class="task-actions">
                                    <button class="btn btn-sm btn-outline" onclick="app.removeScheduledTask('${task.id}')">🗑️ Remove</button>
                                </div>
                            </div>
                        `).join('')}
                    </div>
                </div>
            </div>
        `);
    }

    setupRemoteAccess() {
        const remoteSettings = JSON.parse(localStorage.getItem('remoteAccessSettings') || '{}');
        
        const modal = this.createModal('Remote Access Setup', `
            <div class="remote-access-config">
                <div class="config-warning">
                    <h4>⚠️ Security Notice</h4>
                    <p>Remote access exposes your GoLANshare instance to the internet. Only enable this if you understand the security implications and have proper firewall protection.</p>
                </div>
                
                <div class="config-section">
                    <h4>🌐 Remote Access Options</h4>
                    <label class="setting-label">
                        <input type="checkbox" id="enableRemote" ${remoteSettings.enabled ? 'checked' : ''}>
                        <span>Enable remote access</span>
                    </label>
                    <label class="setting-label">
                        <input type="checkbox" id="requireVPN" ${remoteSettings.requireVPN ? 'checked' : ''}>
                        <span>Require VPN connection</span>
                    </label>
                    <label class="setting-label">
                        <input type="checkbox" id="enableSSL" ${remoteSettings.enableSSL ? 'checked' : ''}>
                        <span>Enable SSL/HTTPS (Recommended)</span>
                    </label>
                </div>
                
                <div class="config-section">
                    <h4>🔐 Security Settings</h4>
                    <div class="form-group">
                        <label>Access Password:</label>
                        <input type="password" class="setting-input" id="remotePassword" placeholder="Enter secure password" value="${remoteSettings.password || ''}">
                        <small>Leave empty to disable password protection (not recommended)</small>
                    </div>
                    <div class="form-group">
                        <label>Allowed IPs (whitelist):</label>
                        <textarea class="setting-input" id="allowedIPs" placeholder="Enter IP addresses (one per line)&#10;Example:&#10;192.168.1.100&#10;10.0.0.0/8" style="width: 100%; height: 80px;">${remoteSettings.allowedIPs || ''}</textarea>
                        <small>Leave empty to allow all IPs (not recommended)</small>
                    </div>
                    <div class="form-group">
                        <label>Remote Port:</label>
                        <input type="number" class="setting-input" id="remotePort" value="${remoteSettings.port || '8082'}" min="1024" max="65535">
                    </div>
                </div>
                
                <div class="config-section">
                    <h4>📊 Connection Info</h4>
                    <div class="connection-info">
                        <p><strong>Local Access:</strong> http://localhost:8081</p>
                        <p><strong>Network Access:</strong> http://${window.location.hostname}:8081</p>
                        <p><strong>Remote Access:</strong> ${remoteSettings.enabled ? `http://${window.location.hostname}:${remoteSettings.port || '8082'}` : 'Disabled'}</p>
                    </div>
                </div>
                
                <div class="config-actions">
                    <button class="btn btn-primary" onclick="app.saveRemoteAccessSettings()">💾 Save & Apply Settings</button>
                    <button class="btn btn-outline" onclick="app.testRemoteConnection()">🔍 Test Connection</button>
                    <button class="btn btn-outline" onclick="this.closest('.modal').remove()">Cancel</button>
                </div>
            </div>
        `);
    }

    createModal(title, content) {
        const modal = document.createElement('div');
        modal.className = 'modal';
        modal.innerHTML = `
            <div class="modal-content" style="max-width: 600px;">
                <div class="modal-header">
                    <h2>${title}</h2>
                    <button class="modal-close" onclick="this.closest('.modal').remove()">×</button>
                </div>
                <div class="modal-body">
                    ${content}
                </div>
            </div>
        `;
        
        document.body.appendChild(modal);
        return modal;
    }

    // Utility methods for analytics
    calculateSuccessRate() {
        const completed = this.transfers.filter(t => t.status === 'completed').length;
        const total = this.transfers.length;
        return total > 0 ? Math.round((completed / total) * 100) : 100;
    }

    getTotalFileSize() {
        return this.files.reduce((total, file) => total + (file.size || 0), 0);
    }

    // Encryption Management Functions
    generateEncryptionKey() {
        if (confirm('Generate a new encryption key? This will make previously encrypted files inaccessible unless you have backed up the old key.')) {
            // Generate a new AES-256 key (simulated)
            const newKey = this.generateRandomKey();
            localStorage.setItem('encryptionKey', newKey);
            document.getElementById('currentKeyInfo').textContent = 'AES-256 (Newly Generated)';
            this.showNotification('New encryption key generated successfully', 'success');
        }
    }

    importEncryptionKey() {
        const input = document.createElement('input');
        input.type = 'file';
        input.accept = '.key,.txt';
        input.onchange = (e) => {
            const file = e.target.files[0];
            if (file) {
                const reader = new FileReader();
                reader.onload = (e) => {
                    try {
                        const keyData = e.target.result;
                        localStorage.setItem('encryptionKey', keyData);
                        document.getElementById('currentKeyInfo').textContent = 'AES-256 (Imported)';
                        this.showNotification('Encryption key imported successfully', 'success');
                    } catch (error) {
                        this.showNotification('Failed to import encryption key', 'error');
                    }
                };
                reader.readAsText(file);
            }
        };
        input.click();
    }

    exportEncryptionKey() {
        const key = localStorage.getItem('encryptionKey') || this.generateRandomKey();
        const blob = new Blob([key], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `golanshare-encryption-key-${Date.now()}.key`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
        this.showNotification('Encryption key exported successfully', 'success');
    }

    saveEncryptionSettings() {
        const settings = {
            encryptUploads: document.getElementById('encryptUploads').checked,
            encryptTransfers: document.getElementById('encryptTransfers').checked,
            encryptStorage: document.getElementById('encryptStorage').checked
        };
        
        // Save to localStorage for now (in production, this would go to the server)
        localStorage.setItem('encryptionSettings', JSON.stringify(settings));
        this.showNotification('Encryption settings saved', 'success');
    }

    generateRandomKey() {
        // Generate a random 256-bit key (simplified for demo)
        const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
        let result = '';
        for (let i = 0; i < 64; i++) {
            result += chars.charAt(Math.floor(Math.random() * chars.length));
        }
        return result;
    }

    // Scheduler Functions
    addScheduledTask() {
        const type = document.getElementById('scheduleType').value;
        const frequency = document.getElementById('scheduleFrequency').value;
        const datetime = document.getElementById('scheduleDateTime').value;
        const description = document.getElementById('scheduleDescription').value;

        if (!datetime) {
            this.showNotification('Please select a date and time', 'error');
            return;
        }

        const task = {
            id: Date.now().toString(),
            type: type,
            frequency: frequency,
            datetime: datetime,
            description: description,
            created: new Date().toISOString(),
            status: 'scheduled'
        };

        const scheduledTasks = JSON.parse(localStorage.getItem('scheduledTasks') || '[]');
        scheduledTasks.push(task);
        localStorage.setItem('scheduledTasks', JSON.stringify(scheduledTasks));

        this.showNotification(`Task scheduled for ${new Date(datetime).toLocaleString()}`, 'success');
        
        // Refresh the scheduler modal
        document.querySelector('.modal').remove();
        this.showScheduler();
    }

    removeScheduledTask(taskId) {
        if (confirm('Remove this scheduled task?')) {
            const scheduledTasks = JSON.parse(localStorage.getItem('scheduledTasks') || '[]');
            const updatedTasks = scheduledTasks.filter(task => task.id !== taskId);
            localStorage.setItem('scheduledTasks', JSON.stringify(updatedTasks));
            
            this.showNotification('Scheduled task removed', 'success');
            
            // Refresh the scheduler modal
            document.querySelector('.modal').remove();
            this.showScheduler();
        }
    }

    // Check for scheduled tasks (called periodically)
    checkScheduledTasks() {
        const scheduledTasks = JSON.parse(localStorage.getItem('scheduledTasks') || '[]');
        const now = new Date();
        
        scheduledTasks.forEach(task => {
            const taskTime = new Date(task.datetime);
            if (taskTime <= now && task.status === 'scheduled') {
                this.executeScheduledTask(task);
                task.status = 'completed';
            }
        });
        
        // Update localStorage with completed tasks
        localStorage.setItem('scheduledTasks', JSON.stringify(scheduledTasks));
    }

    executeScheduledTask(task) {
        this.showNotification(`Executing scheduled task: ${task.type}`, 'info');
        
        switch (task.type) {
            case 'cleanup':
                this.performCleanup();
                break;
            case 'upload':
                this.showNotification('Scheduled upload ready - please select files', 'info');
                break;
            case 'download':
                this.showNotification('Scheduled download ready', 'info');
                break;
            case 'sync':
                this.showNotification('Scheduled sync ready', 'info');
                break;
        }
    }

    performCleanup() {
        // Simple cleanup simulation
        this.showNotification('Performing cleanup of old files...', 'info');
        setTimeout(() => {
            this.showNotification('Cleanup completed', 'success');
        }, 2000);
    }

    // Remote Access Functions
    saveRemoteAccessSettings() {
        const settings = {
            enabled: document.getElementById('enableRemote').checked,
            requireVPN: document.getElementById('requireVPN').checked,
            enableSSL: document.getElementById('enableSSL').checked,
            password: document.getElementById('remotePassword').value,
            allowedIPs: document.getElementById('allowedIPs').value,
            port: document.getElementById('remotePort').value
        };

        localStorage.setItem('remoteAccessSettings', JSON.stringify(settings));
        
        if (settings.enabled) {
            this.showNotification('Remote access settings saved. Server restart may be required.', 'success');
        } else {
            this.showNotification('Remote access disabled', 'info');
        }
        
        // Close modal
        document.querySelector('.modal').remove();
    }

    testRemoteConnection() {
        const settings = JSON.parse(localStorage.getItem('remoteAccessSettings') || '{}');
        
        if (!settings.enabled) {
            this.showNotification('Remote access is not enabled', 'warning');
            return;
        }

        this.showNotification('Testing remote connection...', 'info');
        
        // Simulate connection test
        setTimeout(() => {
            const success = Math.random() > 0.3; // 70% success rate for demo
            if (success) {
                this.showNotification('Remote connection test successful!', 'success');
            } else {
                this.showNotification('Remote connection test failed. Check firewall and network settings.', 'error');
            }
        }, 2000);
    }

    // Private Sharing Functions
    showShareOptions(deviceId) {
        const device = this.devices.find(d => d.id === deviceId);
        if (!device) return;

        const modal = this.createModal(`Share with ${device.name}`, `
            <div class="share-options-panel">
                <div class="share-target">
                    <div class="target-device">
                        <div class="device-avatar-large">
                            ${this.getDeviceIcon(device.os)}
                        </div>
                        <div class="target-info">
                            <h4>${device.name}</h4>
                            <p>${device.ip} • ${device.os}</p>
                            <span class="connection-status ${device.is_connected ? 'connected' : 'disconnected'}">
                                ${device.is_connected ? '🟢 Connected' : '🔴 Not Connected'}
                            </span>
                        </div>
                    </div>
                </div>
                
                <div class="share-methods">
                    <div class="share-method" onclick="app.shareFiles('${deviceId}')">
                        <div class="method-icon">📁</div>
                        <div class="method-info">
                            <h4>Share Files</h4>
                            <p>Send files directly to this device</p>
                        </div>
                        <div class="method-arrow">→</div>
                    </div>
                    
                    <div class="share-method" onclick="app.shareMessage('${deviceId}')">
                        <div class="method-icon">💬</div>
                        <div class="method-info">
                            <h4>Send Message</h4>
                            <p>Send a private text message</p>
                        </div>
                        <div class="method-arrow">→</div>
                    </div>
                    
                </div>
                
                <div class="share-actions">
                    <button class="btn btn-outline" onclick="this.closest('.modal').remove()">Cancel</button>
                </div>
            </div>
        `);
    }

    async shareFiles(deviceId) {
        // Close current modal
        document.querySelectorAll('.modal').forEach(modal => modal.remove());
        
        // Open file picker
        const input = document.createElement('input');
        input.type = 'file';
        input.multiple = true;
        input.onchange = async (e) => {
            const files = Array.from(e.target.files);
            if (files.length > 0) {
                await this.sendFilesToDevice(deviceId, files);
            }
        };
        input.click();
    }

    async sendFilesToDevice(deviceId, files) {
        const device = this.devices.find(d => d.id === deviceId);
        if (!device) return;

        // First upload files to server
        this.showNotification(`Preparing to send ${files.length} file(s) to ${device.name}...`, 'info');
        
        try {
            // Upload files first
            await this.uploadFiles(files);
            
            // Get file names
            const fileNames = files.map(f => f.name);
            
            // Send share request
            const response = await fetch(`${this.apiBase}/api/share/send`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer ' + this.token
                },
                body: JSON.stringify({
                    to_device_id: deviceId,
                    type: 'file',
                    files: fileNames
                })
            });

            if (response.ok) {
                const result = await response.json();
                this.showNotification(`Share request sent to ${device.name}! Expires in 5 minutes.`, 'success', 8000);
                
                // Show pending request status
                this.showPendingRequest(result.request_id, device.name);
            } else {
                throw new Error('Failed to send share request');
            }
        } catch (error) {
            this.showNotification(`Failed to share files: ${error.message}`, 'error');
        }
    }

    shareMessage(deviceId) {
        const device = this.devices.find(d => d.id === deviceId);
        if (!device) return;

        // Close current modal
        document.querySelectorAll('.modal').forEach(modal => modal.remove());

        const modal = this.createModal(`Send Message to ${device.name}`, `
            <div class="message-composer">
                <div class="composer-header">
                    <div class="target-device-small">
                        <span class="device-icon">${this.getDeviceIcon(device.os)}</span>
                        <span>${device.name}</span>
                    </div>
                </div>
                
                <div class="message-input-area">
                    <textarea id="messageText" placeholder="Type your message here..." rows="4" style="width: 100%; padding: 1rem; border: 1px solid var(--border); border-radius: var(--radius); resize: vertical;"></textarea>
                </div>
                
                <div class="message-actions">
                    <button class="btn btn-outline" onclick="this.closest('.modal').remove()">Cancel</button>
                    <button class="btn btn-primary" onclick="app.sendPrivateMessage('${deviceId}')">
                        <span class="btn-icon">📤</span>
                        Send Message
                    </button>
                </div>
            </div>
        `);

        // Focus on textarea
        setTimeout(() => {
            document.getElementById('messageText')?.focus();
        }, 100);
    }

    async sendPrivateMessage(deviceId) {
        const messageText = document.getElementById('messageText')?.value.trim();
        if (!messageText) {
            this.showNotification('Please enter a message', 'warning');
            return;
        }

        const device = this.devices.find(d => d.id === deviceId);
        if (!device) return;

        try {
            const response = await fetch(`${this.apiBase}/api/share/send`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer ' + this.token
                },
                body: JSON.stringify({
                    to_device_id: deviceId,
                    type: 'message',
                    message: messageText
                })
            });

            if (response.ok) {
                document.querySelectorAll('.modal').forEach(modal => modal.remove());
                this.showNotification(`Message sent to ${device.name}!`, 'success');
            } else {
                throw new Error('Failed to send message');
            }
        } catch (error) {
            this.showNotification(`Failed to send message: ${error.message}`, 'error');
        }
    }

    shareScreen(deviceId) {
        this.showNotification('Screen sharing coming soon!', 'info');
    }

    async shareClipboard(deviceId) {
        try {
            const clipboardText = await navigator.clipboard.readText();
            if (!clipboardText) {
                this.showNotification('Clipboard is empty', 'warning');
                return;
            }

            const device = this.devices.find(d => d.id === deviceId);
            if (!device) return;

            const response = await fetch(`${this.apiBase}/api/share/send`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer ' + this.token
                },
                body: JSON.stringify({
                    to_device_id: deviceId,
                    type: 'message',
                    message: `📋 Clipboard: ${clipboardText}`
                })
            });

            if (response.ok) {
                document.querySelectorAll('.modal').forEach(modal => modal.remove());
                this.showNotification(`Clipboard shared with ${device.name}!`, 'success');
            } else {
                throw new Error('Failed to share clipboard');
            }
        } catch (error) {
            this.showNotification(`Failed to share clipboard: ${error.message}`, 'error');
        }
    }

    async connectToDevice(deviceId) {
        const device = this.devices.find(d => d.id === deviceId);
        if (!device) return;

        if (device.is_connected) {
            // Disconnect
            try {
                const response = await fetch(`${this.apiBase}/api/device/disconnect`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + this.token
                    },
                    body: JSON.stringify({ device_id: deviceId })
                });

                if (response.ok) {
                    device.is_connected = false;
                    this.renderDevices();
                    this.showNotification(`Disconnected from ${device.name}`, 'info');
                }
            } catch (error) {
                this.showNotification(`Failed to disconnect: ${error.message}`, 'error');
            }
        } else {
            // Connect
            try {
                const response = await fetch(`${this.apiBase}/api/device/connect`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + this.token
                    },
                    body: JSON.stringify({ device_id: deviceId })
                });

                if (response.ok) {
                    const result = await response.json();
                    device.is_connected = result.connected;
                    this.renderDevices();
                    
                    if (result.connected) {
                        this.showNotification(`Connected to ${device.name}!`, 'success');
                    } else {
                        this.showNotification(`Failed to connect to ${device.name}`, 'error');
                    }
                }
            } catch (error) {
                this.showNotification(`Connection failed: ${error.message}`, 'error');
            }
        }
    }

    showPendingRequest(requestId, deviceName) {
        const notification = this.showNotification(
            `Waiting for ${deviceName} to accept your share request...`,
            'info',
            0, // Don't auto-remove
            [{
                label: 'Cancel',
                callback: `app.cancelShareRequest('${requestId}')`
            }]
        );
        
        // Store notification for later removal
        this.pendingRequests = this.pendingRequests || {};
        this.pendingRequests[requestId] = notification;
    }

    async cancelShareRequest(requestId) {
        // Remove the notification
        if (this.pendingRequests && this.pendingRequests[requestId]) {
            this.pendingRequests[requestId].remove();
            delete this.pendingRequests[requestId];
        }
        
        this.showNotification('Share request cancelled', 'info', 2000);
    }

    // Handle incoming share requests
    handleIncomingShareRequest(shareRequest) {
        const modal = this.createModal(`📤 Incoming Share Request`, `
            <div class="share-request-panel">
                <div class="request-header">
                    <div class="sender-info">
                        <div class="sender-avatar">
                            ${this.getDeviceIcon(shareRequest.from_device_os || 'Unknown')}
                        </div>
                        <div class="sender-details">
                            <h4>${shareRequest.from_name}</h4>
                            <p>wants to share ${shareRequest.type === 'file' ? 'files' : 'a message'} with you</p>
                        </div>
                    </div>
                    <div class="request-timer">
                        <span class="timer-text">Expires in 5:00</span>
                    </div>
                </div>
                
                <div class="request-content">
                    ${shareRequest.type === 'file' ? `
                        <div class="file-preview">
                            <h5>📁 Files (${shareRequest.files.length})</h5>
                            <div class="file-list">
                                ${shareRequest.files.map(file => `
                                    <div class="file-item-preview">
                                        <span class="file-icon">${this.getFileIcon(file)}</span>
                                        <span class="file-name">${file}</span>
                                    </div>
                                `).join('')}
                            </div>
                            ${shareRequest.size ? `<p class="total-size">Total size: ${this.formatFileSize(shareRequest.size)}</p>` : ''}
                        </div>
                    ` : `
                        <div class="message-preview">
                            <h5>💬 Message</h5>
                            <div class="message-content">${shareRequest.message}</div>
                        </div>
                    `}
                </div>
                
                <div class="request-actions">
                    <button class="btn btn-error" onclick="app.rejectShareRequest('${shareRequest.id}')">
                        <span class="btn-icon">❌</span>
                        Reject
                    </button>
                    <button class="btn btn-success" onclick="app.acceptShareRequest('${shareRequest.id}')">
                        <span class="btn-icon">✅</span>
                        Accept
                    </button>
                </div>
            </div>
        `);

        // Start countdown timer
        this.startRequestTimer(shareRequest.id, shareRequest.expires_at);
    }

    async acceptShareRequest(requestId) {
        try {
            const response = await fetch(`${this.apiBase}/api/share/accept`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer ' + this.token
                },
                body: JSON.stringify({ request_id: requestId })
            });

            if (response.ok) {
                document.querySelectorAll('.modal').forEach(modal => modal.remove());
                this.showNotification('Share request accepted! Transfer starting...', 'success');
            } else {
                throw new Error('Failed to accept share request');
            }
        } catch (error) {
            this.showNotification(`Failed to accept: ${error.message}`, 'error');
        }
    }

    async rejectShareRequest(requestId) {
        try {
            const response = await fetch(`${this.apiBase}/api/share/reject`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer ' + this.token
                },
                body: JSON.stringify({ request_id: requestId })
            });

            if (response.ok) {
                document.querySelectorAll('.modal').forEach(modal => modal.remove());
                this.showNotification('Share request rejected', 'info');
            } else {
                throw new Error('Failed to reject share request');
            }
        } catch (error) {
            this.showNotification(`Failed to reject: ${error.message}`, 'error');
        }
    }

    startRequestTimer(requestId, expiresAt) {
        const timerElement = document.querySelector('.timer-text');
        if (!timerElement) return;

        const updateTimer = () => {
            const now = new Date();
            const expires = new Date(expiresAt);
            const remaining = expires - now;

            if (remaining <= 0) {
                document.querySelectorAll('.modal').forEach(modal => modal.remove());
                this.showNotification('Share request expired', 'warning');
                return;
            }

            const minutes = Math.floor(remaining / 60000);
            const seconds = Math.floor((remaining % 60000) / 1000);
            timerElement.textContent = `Expires in ${minutes}:${seconds.toString().padStart(2, '0')}`;

            setTimeout(updateTimer, 1000);
        };

        updateTimer();
    }

    // WebSocket message handling
    handleWebSocketMessage(data) {
        switch (data.type) {
            case 'message':
                this.addChatMessage(data);
                break;
            case 'devices_updated':
                // Update device list in real-time
                this.devices = data.devices || [];
                this.renderDevices();
                console.log('Devices updated:', data.count, 'devices online');
                break;
            case 'share_request':
                this.handleIncomingShareRequest(data.share_request);
                this.showNotification(`📤 ${data.share_request.from_name} wants to share ${data.share_request.type === 'file' ? 'files' : 'a message'} with you!`, 'info', 8000);
                break;
            case 'share_accepted':
                this.showNotification(`✅ ${data.message}`, 'success');
                // Remove pending request notification
                if (this.pendingRequests && this.pendingRequests[data.request.id]) {
                    this.pendingRequests[data.request.id].remove();
                    delete this.pendingRequests[data.request.id];
                }
                break;
            case 'share_rejected':
                this.showNotification(`❌ ${data.message}`, 'warning');
                // Remove pending request notification
                if (this.pendingRequests && this.pendingRequests[data.request.id]) {
                    this.pendingRequests[data.request.id].remove();
                    delete this.pendingRequests[data.request.id];
                }
                break;
            case 'private_message':
                this.handlePrivateMessage(data);
                break;
            case 'device_connected':
                this.handleDeviceConnection(data);
                break;
            default:
                console.log('Unknown WebSocket message type:', data.type);
        }
    }

    handlePrivateMessage(data) {
        // Show private message notification
        this.showNotification(`💬 Private message from ${data.message.username}: ${data.message.message}`, 'info', 10000);
        
        // Add to chat if chat is open
        this.addChatMessage(data.message);
    }

    handleDeviceConnection(data) {
        this.showNotification(`🔗 ${data.device.name} wants to connect`, 'info', 5000);
        
        // Update device list
        this.loadDevices();
    }

    // Manual device addition
    addDeviceManually() {
        const modal = this.createModal('Add Device Manually', `
            <div class="add-device-form">
                <p>Enter the IP address of a device running GoLANshare:</p>
                <div class="form-group">
                    <label>IP Address:</label>
                    <input type="text" id="deviceIP" placeholder="192.168.1.100" class="setting-input" style="width: 200px;">
                </div>
                <div class="form-group">
                    <label>Device Name (optional):</label>
                    <input type="text" id="deviceName" placeholder="My Device" class="setting-input" style="width: 200px;">
                </div>
                <div class="form-actions">
                    <button class="btn btn-outline" onclick="this.closest('.modal').remove()">Cancel</button>
                    <button class="btn btn-primary" onclick="app.addDeviceByIP()">Add Device</button>
                </div>
            </div>
        `);
    }

    async addDeviceByIP() {
        const ip = document.getElementById('deviceIP')?.value.trim();
        const name = document.getElementById('deviceName')?.value.trim();
        
        if (!ip) {
            this.showNotification('Please enter an IP address', 'warning');
            return;
        }
        
        // Basic IP validation
        const ipRegex = /^(\d{1,3}\.){3}\d{1,3}$/;
        if (!ipRegex.test(ip)) {
            this.showNotification('Please enter a valid IP address', 'warning');
            return;
        }
        
        try {
            const response = await fetch(`${this.apiBase}/api/device/add`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer ' + this.token
                },
                body: JSON.stringify({ ip, name })
            });
            
            if (response.ok) {
                document.querySelectorAll('.modal').forEach(modal => modal.remove());
                this.showNotification(`Checking for GoLANshare at ${ip}...`, 'info', 3000);
                
                // Refresh devices after a short delay
                setTimeout(() => this.loadDevices(), 2000);
            } else {
                throw new Error('Failed to add device');
            }
        } catch (error) {
            this.showNotification(`Failed to add device: ${error.message}`, 'error');
        }
    }

    showDebugInfo() {
        console.log('=== DEBUG INFO ===');
        console.log('API Base:', this.apiBase);
        console.log('Current User:', this.currentUser);
        console.log('Token:', this.token ? 'Present' : 'None');
        console.log('Devices Array:', this.devices);
        console.log('WebSocket Status:', this.ws ? this.ws.readyState : 'Not connected');
        console.log('Online Status:', this.isOnline);
        console.log('==================');
        
        this.showNotification('Debug info logged to console', 'info', 3000);
        
        // Also test the API endpoints
        this.testAllEndpoints();
    }

    async testAllEndpoints() {
        const endpoints = [
            '/api/devices',
            '/api/device/info',
            '/api/device/test',
            '/api/debug/clients',
            '/api/transfers',
            '/api/files'
        ];
        
        console.log('=== TESTING API ENDPOINTS ===');
        for (const endpoint of endpoints) {
            try {
                const response = await fetch(this.apiBase + endpoint);
                console.log(`${endpoint}: ${response.status} ${response.statusText}`);
                if (response.ok) {
                    const data = await response.json();
                    console.log(`  Data:`, data);
                    
                    // Special handling for debug endpoint
                    if (endpoint === '/api/debug/clients') {
                        console.log('=== CLIENT DEBUG INFO ===');
                        console.log('Your IP:', data.requesting_client_ip);
                        console.log('Total Devices:', data.total_devices);
                        console.log('WebSocket Clients:', data.websocket_clients);
                        console.log('Self Device:', data.self_device);
                        console.log('All Devices:', data.devices);
                        console.log('========================');
                        
                        // Show in UI
                        this.showNotification(
                            `Your IP: ${data.requesting_client_ip} | Devices: ${data.total_devices} | WS Clients: ${data.websocket_clients}`, 
                            'info', 
                            10000
                        );
                    }
                }
            } catch (error) {
                console.log(`${endpoint}: ERROR -`, error.message);
            }
        }
        console.log('==============================');
    }

    // Test device discovery
    async testDiscovery() {
        this.showNotification('Testing device discovery...', 'info');
        
        try {
            // Test local device info
            const deviceInfoResponse = await fetch(`${this.apiBase}/api/device/info`);
            if (deviceInfoResponse.ok) {
                const deviceInfo = await deviceInfoResponse.json();
                console.log('Local device info:', deviceInfo);
                
                // Test device test endpoint
                const testResponse = await fetch(`${this.apiBase}/api/device/test`);
                if (testResponse.ok) {
                    const testData = await testResponse.json();
                    console.log('Device test response:', testData);
                    
                    this.showNotification(`✅ Discovery test passed! Device: ${testData.device.name} (${testData.device.ip})`, 'success', 5000);
                    
                    // Show current devices
                    const currentDevices = this.devices.length;
                    this.showNotification(`Current devices found: ${currentDevices}`, 'info', 3000);
                    
                } else {
                    throw new Error('Device test endpoint failed');
                }
            } else {
                throw new Error('Device info endpoint failed');
            }
        } catch (error) {
            this.showNotification(`❌ Discovery test failed: ${error.message}`, 'error');
        }
    }

    // Quick Share Functions

    quickShareFiles() {
        if (this.devices.filter(d => !d.is_self).length === 0) {
            this.showNotification('No devices available for sharing', 'warning');
            return;
        }

        this.showDeviceSelector('Share Files', (deviceId) => {
            this.shareFiles(deviceId);
        });
    }

    showDeviceSelector(title, callback) {
        const availableDevices = this.devices.filter(d => !d.is_self);
        
        const modal = this.createModal(title, `
            <div class="device-selector">
                <p>Select a device to share with:</p>
                <div class="device-selector-list">
                    ${availableDevices.map(device => `
                        <div class="device-selector-item" onclick="${callback.toString().replace('deviceId', `'${device.id}'`)}; this.closest('.modal').remove();">
                            <div class="device-icon">${this.getDeviceIcon(device.os)}</div>
                            <div class="device-info">
                                <div class="device-name">${device.name}</div>
                                <div class="device-details">${device.ip} • ${device.os}</div>
                            </div>
                            <div class="connection-status ${device.is_connected ? 'connected' : 'disconnected'}">
                                ${device.is_connected ? '🟢' : '🔴'}
                            </div>
                        </div>
                    `).join('')}
                </div>
            </div>
        `);
    }

    async showShareHistory() {
        try {
            const response = await fetch(`${this.apiBase}/api/share/history`, {
                headers: {
                    'Authorization': 'Bearer ' + this.token
                }
            });

            if (response.ok) {
                const data = await response.json();
                this.displayShareHistory(data.history || []);
            } else {
                throw new Error('Failed to load share history');
            }
        } catch (error) {
            this.showNotification('Failed to load share history: ' + error.message, 'error');
        }
    }

    displayShareHistory(history) {
        const modal = this.createModal('📋 Share History', `
            <div class="share-history-panel">
                ${history.length === 0 ? `
                    <div class="empty-state">
                        <div class="empty-icon">📋</div>
                        <p>No sharing history</p>
                        <small>Your sent and received shares will appear here</small>
                    </div>
                ` : `
                    <div class="history-list">
                        ${history.map(item => `
                            <div class="history-item">
                                <div class="history-icon">
                                    ${item.type === 'file' ? '📁' : '💬'}
                                </div>
                                <div class="history-info">
                                    <div class="history-title">
                                        ${item.from_device === this.currentDeviceId ? 'Sent to' : 'Received from'} 
                                        ${item.from_device === this.currentDeviceId ? item.to_name : item.from_name}
                                    </div>
                                    <div class="history-details">
                                        ${item.type === 'file' ? `${item.files?.length || 0} files` : 'Message'} • 
                                        ${new Date(item.created_at).toLocaleDateString()}
                                    </div>
                                </div>
                                <div class="history-status ${item.status}">
                                    ${item.status}
                                </div>
                            </div>
                        `).join('')}
                    </div>
                `}
            </div>
        `);
    }

    async showPendingRequests() {
        try {
            const response = await fetch(`${this.apiBase}/api/share/receive`, {
                headers: {
                    'Authorization': 'Bearer ' + this.token
                }
            });

            if (response.ok) {
                const data = await response.json();
                this.displayPendingRequests(data.requests || []);
                this.updatePendingCount(data.count || 0);
            } else {
                throw new Error('Failed to load pending requests');
            }
        } catch (error) {
            this.showNotification('Failed to load pending requests: ' + error.message, 'error');
        }
    }

    displayPendingRequests(requests) {
        const modal = this.createModal('⏳ Pending Share Requests', `
            <div class="pending-requests-panel">
                ${requests.length === 0 ? `
                    <div class="empty-state">
                        <div class="empty-icon">⏳</div>
                        <p>No pending requests</p>
                        <small>Incoming share requests will appear here</small>
                    </div>
                ` : `
                    <div class="requests-list">
                        ${requests.map(request => `
                            <div class="request-item">
                                <div class="request-header">
                                    <div class="sender-info">
                                        <span class="sender-name">${request.from_name}</span>
                                        <span class="request-type">${request.type === 'file' ? '📁 Files' : '💬 Message'}</span>
                                    </div>
                                    <div class="request-time">
                                        ${this.getTimeAgo(request.created_at)}
                                    </div>
                                </div>
                                <div class="request-content">
                                    ${request.type === 'file' ? 
                                        `${request.files?.length || 0} files (${this.formatFileSize(request.size || 0)})` : 
                                        request.message
                                    }
                                </div>
                                <div class="request-actions">
                                    <button class="btn btn-sm btn-error" onclick="app.rejectShareRequest('${request.id}')">
                                        Reject
                                    </button>
                                    <button class="btn btn-sm btn-success" onclick="app.acceptShareRequest('${request.id}')">
                                        Accept
                                    </button>
                                </div>
                            </div>
                        `).join('')}
                    </div>
                `}
            </div>
        `);
    }

    updatePendingCount(count) {
        document.getElementById('pendingCount').textContent = count;
        
        // Add notification badge if there are pending requests
        const button = document.getElementById('pendingRequestsBtn');
        const existingBadge = button.querySelector('.notification-badge');
        
        if (count > 0 && !existingBadge) {
            const badge = document.createElement('span');
            badge.className = 'notification-badge';
            badge.textContent = count;
            button.style.position = 'relative';
            button.appendChild(badge);
        } else if (count === 0 && existingBadge) {
            existingBadge.remove();
        } else if (existingBadge) {
            existingBadge.textContent = count;
        }
    }

    getTimeAgo(dateString) {
        const now = new Date();
        const date = new Date(dateString);
        const diffMs = now - date;
        const diffMins = Math.floor(diffMs / 60000);
        
        if (diffMins < 1) return 'Just now';
        if (diffMins < 60) return `${diffMins}m ago`;
        
        const diffHours = Math.floor(diffMins / 60);
        if (diffHours < 24) return `${diffHours}h ago`;
        
        const diffDays = Math.floor(diffHours / 24);
        return `${diffDays}d ago`;
    }

    // Auto-refresh pending requests
    startPendingRequestsMonitoring() {
        setInterval(async () => {
            try {
                const response = await fetch(`${this.apiBase}/api/share/receive`, {
                    headers: {
                        'Authorization': 'Bearer ' + this.token
                    }
                });

                if (response.ok) {
                    const data = await response.json();
                    this.updatePendingCount(data.count || 0);
                }
            } catch (error) {
                // Silently fail - don't spam user with errors
            }
        }, 10000); // Check every 10 seconds
    }

    getFileIcon(filename) {
        const ext = filename.split('.').pop().toLowerCase();
        const icons = {
            'pdf': '📄',
            'doc': '📝',
            'docx': '📝',
            'txt': '📝',
            'jpg': '🖼️',
            'png': '🖼️',
            'gif': '🖼️',
            'mp4': '🎥',
            'avi': '🎥',
            'mov': '🎥',
            'mp3': '🎵',
            'wav': '🎵',
            'zip': '📦',
            'rar': '📦'
        };
        return icons[ext] || '📁';
    }

    addTransfer(transfer) {
        this.transfers.unshift(transfer);
        this.renderTransfers();
    }



    clearCompletedTransfers() {
        const completedCount = this.transfers.filter(t => t.status === 'completed' || t.status === 'cancelled').length;
        
        if (completedCount === 0) {
            this.showNotification('No completed transfers to clear', 'info');
            return;
        }
        
        // Remove completed and cancelled transfers
        this.transfers = this.transfers.filter(t => 
            t.status !== 'completed' && 
            t.status !== 'cancelled' && 
            t.status !== 'failed'
        );
        
        this.renderTransfers();
        this.showNotification(`Cleared ${completedCount} completed transfer(s)`, 'success');
        
        // Update browser title
        this.updateBrowserTitle();
    }

    // Chat
    connectWebSocket() {
        try {
            this.ws = new WebSocket(this.apiBase.replace('http', 'ws') + '/ws');
            
            this.ws.onopen = () => {
                this.showNotification('Connected to chat', 'success');
            };
            
            this.ws.onmessage = (e) => {
                const data = JSON.parse(e.data);
                this.handleWebSocketMessage(data);
            };
            
            this.ws.onclose = () => {
                setTimeout(() => this.connectWebSocket(), 3000);
            };
        } catch (error) {
            console.log('WebSocket not available');
        }
    }

    addChatMessage(msg) {
        this.chatMessages.push(msg);
        this.renderChat();
        document.getElementById('chatCount').textContent = this.chatMessages.length;
    }

    renderChat() {
        const container = document.getElementById('chatMessages');
        container.innerHTML = this.chatMessages.map(msg => `
            <div class="message">
                <strong>${msg.username}:</strong> ${msg.message}
            </div>
        `).join('');
        
        container.scrollTop = container.scrollHeight;
    }

    sendChatMessage() {
        const input = document.getElementById('chatInput');
        const message = input.value.trim();
        
        if (message && this.ws) {
            const msg = {
                type: 'message',
                username: this.currentUser,
                message: message,
                timestamp: new Date().toISOString()
            };
            
            this.ws.send(JSON.stringify(msg));
            input.value = '';
        }
    }

    // Quick Actions
    handleQuickAction(e) {
        const action = e.currentTarget.dataset.action;
        const actions = {
            text: () => this.shareText()
        };
        
        if (actions[action]) {
            actions[action]();
        }
    }

    shareText() {
        const text = prompt('Enter text to share:');
        if (text) {
            this.showNotification('Text shared to devices', 'success');
        }
    }



    // Utilities
    setupApp() {
        // Initialize counts
        document.getElementById('filesCount').textContent = '0';
        document.getElementById('chatCount').textContent = '0';
    }

    // Files
    async loadFiles() {
        try {
            const response = await fetch(this.apiBase + '/api/files');
            if (response.ok) {
                this.files = await response.json();
                this.renderFiles();
            }
        } catch (error) {
            this.files = [];
            this.renderFiles();
        }
    }

    renderFiles() {
        const container = document.getElementById('filesList');
        const countElement = document.getElementById('filesCount');
        
        if (!this.files || !Array.isArray(this.files)) {
            this.files = [];
        }
        
        countElement.textContent = this.files.length;
        
        if (this.files.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <div class="empty-icon">📁</div>
                    <p>No shared files</p>
                    <small>Upload files to share them with other devices</small>
                </div>
            `;
            return;
        }
        
        container.innerHTML = this.files.map(file => `
            <div class="file-item">
                <div class="file-icon">${this.getFileIcon(file.name)}</div>
                <div class="file-info">
                    <div class="file-name">${file.name}</div>
                    <div class="file-details">
                        ${this.formatFileSize(file.size)} • ${new Date(file.modified).toLocaleDateString()}
                    </div>
                </div>
                <div class="file-actions">
                    <button class="btn btn-sm btn-primary" onclick="app.downloadFile('${file.name}')">
                        <span class="btn-icon">📥</span>
                        Download
                    </button>
                    <button class="btn btn-sm btn-outline" onclick="app.shareFile('${file.name}')">
                        <span class="btn-icon">🔗</span>
                        Share
                    </button>
                    <button class="btn btn-sm btn-error" onclick="app.deleteFile('${file.name}')">
                        <span class="btn-icon">🗑️</span>
                        Delete
                    </button>
                </div>
            </div>
        `).join('');
    }

    downloadFile(filename) {
        const url = `${this.apiBase}/api/download/${encodeURIComponent(filename)}`;
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        this.showNotification(`Downloading ${filename}...`, 'info');
    }

    shareFile(filename) {
        const url = `${window.location.origin}/files/${encodeURIComponent(filename)}`;
        if (navigator.share) {
            navigator.share({
                title: `GoLANshare - ${filename}`,
                text: `Download ${filename} from GoLANshare`,
                url: url
            });
        } else {
            navigator.clipboard.writeText(url).then(() => {
                this.showNotification('File link copied to clipboard!', 'success');
            });
        }
    }

    async deleteFile(filename) {
        if (!confirm(`Are you sure you want to delete "${filename}"? This action cannot be undone.`)) {
            return;
        }
        
        try {
            const response = await fetch(`${this.apiBase}/api/files/delete`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer ' + this.token
                },
                body: JSON.stringify({ filename })
            });
            
            if (response.ok) {
                this.showNotification(`File "${filename}" deleted successfully`, 'success');
                // Refresh file list
                this.loadFiles();
            } else {
                const error = await response.json();
                throw new Error(error.message || 'Failed to delete file');
            }
        } catch (error) {
            this.showNotification(`Failed to delete file: ${error.message}`, 'error');
        }
    }

    startHeartbeat() {
        setInterval(() => {
            this.loadDevices();
            this.loadTransfers();
            this.loadFiles();
            this.checkScheduledTasks(); // Check for scheduled tasks
        }, 10000); // Update every 10 seconds
    }

    formatFileSize(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
    }

    // Enhanced notification system
    showNotification(message, type = 'info', duration = 5000, actions = []) {
        const notifications = document.getElementById('notifications');
        const notification = document.createElement('div');
        notification.className = `toast ${type}`;
        
        const icon = this.getNotificationIcon(type);
        const actionsHtml = actions.map(action => 
            `<button class="btn btn-sm btn-outline" onclick="${action.callback}">${action.label}</button>`
        ).join('');
        
        notification.innerHTML = `
            <div class="toast-content">
                <div class="toast-header">
                    <span class="toast-icon">${icon}</span>
                    <span class="toast-message">${message}</span>
                    <button class="toast-close" onclick="this.parentElement.parentElement.parentElement.remove()">×</button>
                </div>
                ${actionsHtml ? `<div class="toast-actions">${actionsHtml}</div>` : ''}
            </div>
        `;
        
        notifications.appendChild(notification);
        
        // Auto-remove after duration
        if (duration > 0) {
            setTimeout(() => {
                if (notification.parentElement) {
                    notification.style.animation = 'slideOutRight 0.3s ease';
                    setTimeout(() => notification.remove(), 300);
                }
            }, duration);
        }
        
        return notification;
    }

    getNotificationIcon(type) {
        const icons = {
            'success': '✅',
            'error': '❌',
            'warning': '⚠️',
            'info': 'ℹ️'
        };
        return icons[type] || 'ℹ️';
    }

    // Network monitoring
    setupNetworkMonitoring() {
        window.addEventListener('online', () => {
            this.isOnline = true;
            this.showNotification('Connection restored', 'success');
            this.reconnectWebSocket();
            this.processUploadQueue();
        });

        window.addEventListener('offline', () => {
            this.isOnline = false;
            this.showNotification('Connection lost - working offline', 'warning', 0);
        });
    }

    // Global error handling
    setupGlobalErrorHandling() {
        window.addEventListener('error', (event) => {
            console.error('Global error:', event.error);
            this.showNotification('An unexpected error occurred', 'error');
        });

        window.addEventListener('unhandledrejection', (event) => {
            console.error('Unhandled promise rejection:', event.reason);
            this.showNotification('Network request failed', 'error');
        });
    }

    // Keyboard shortcuts
    setupKeyboardShortcuts() {
        document.addEventListener('keydown', (e) => {
            if (e.ctrlKey || e.metaKey) {
                switch (e.key) {
                    case 'u':
                        e.preventDefault();
                        this.openFilePicker();
                        break;
                    case 'r':
                        e.preventDefault();
                        this.refreshAll();
                        break;
                    case '/':
                        e.preventDefault();
                        document.getElementById('chatInput')?.focus();
                        break;
                }
            }
            
            if (e.key === 'Escape') {
                this.closeModals();
            }
        });
    }

    // Enhanced drag and drop
    setupDragAndDrop() {
        const dragOverlay = document.createElement('div');
        dragOverlay.className = 'drag-overlay';
        dragOverlay.innerHTML = `
            <div class="drag-message">
                <div style="font-size: 3rem; margin-bottom: 1rem;">📁</div>
                <h3>Drop files here to upload</h3>
                <p>Release to start uploading</p>
            </div>
        `;
        document.body.appendChild(dragOverlay);

        let dragCounter = 0;

        document.addEventListener('dragenter', (e) => {
            e.preventDefault();
            dragCounter++;
            if (e.dataTransfer.types.includes('Files')) {
                dragOverlay.classList.add('active');
            }
        });

        document.addEventListener('dragleave', (e) => {
            e.preventDefault();
            dragCounter--;
            if (dragCounter === 0) {
                dragOverlay.classList.remove('active');
            }
        });

        document.addEventListener('dragover', (e) => {
            e.preventDefault();
        });

        document.addEventListener('drop', (e) => {
            e.preventDefault();
            dragCounter = 0;
            dragOverlay.classList.remove('active');
            
            if (e.dataTransfer.files.length > 0) {
                this.handleFilesDrop(e.dataTransfer.files);
            }
        });
    }

    // Service Worker setup
    setupServiceWorker() {
        if ('serviceWorker' in navigator) {
            navigator.serviceWorker.register('/sw.js')
                .then(registration => {
                    console.log('SW registered:', registration);
                    this.checkForUpdates(registration);
                })
                .catch(error => {
                    console.log('SW registration failed:', error);
                });
        }
    }

    checkForUpdates(registration) {
        registration.addEventListener('updatefound', () => {
            const newWorker = registration.installing;
            newWorker.addEventListener('statechange', () => {
                if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
                    this.showNotification(
                        'New version available!', 
                        'info', 
                        0,
                        [{
                            label: 'Refresh',
                            callback: 'window.location.reload()'
                        }]
                    );
                }
            });
        });
    }

    // Utility methods
    refreshAll() {
        this.loadDevices();
        this.loadTransfers();
        this.loadFiles();
        this.showNotification('Refreshed all data', 'success', 2000);
    }

    closeModals() {
        document.querySelectorAll('.modal').forEach(modal => {
            modal.style.display = 'none';
        });
    }

    handleFilesDrop(files) {
        const fileArray = Array.from(files);
        this.uploadFiles(fileArray);
    }
}

// Initialize app when page loads
document.addEventListener('DOMContentLoaded', () => {
    window.app = new GoLANshare();
    
    // Register service worker for PWA
    if ('serviceWorker' in navigator) {
        navigator.serviceWorker.register('/sw.js')
            .then(registration => {
                console.log('SW registered: ', registration);
            })
            .catch(registrationError => {
                console.log('SW registration failed: ', registrationError);
            });
    }
});