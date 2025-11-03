class GoLANshare {
    constructor() {
        this.apiBase = window.location.origin;
        this.ws = null;
        this.token = null;
        this.devices = [];
        this.transfers = [];
        this.chatMessages = [];
        this.currentUser = 'User';
        
        this.init();
    }

    init() {
        this.bindEvents();
        this.checkAuth();
        this.setupApp();
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
        
        // Chat
        document.getElementById('sendChatBtn').addEventListener('click', () => this.sendChatMessage());
        document.getElementById('chatInput').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.sendChatMessage();
        });
        
        // Transfers
        document.getElementById('pauseAllBtn').addEventListener('click', () => this.pauseAllTransfers());
        document.getElementById('resumeAllBtn').addEventListener('click', () => this.resumeAllTransfers());
        document.getElementById('clearCompletedBtn').addEventListener('click', () => this.clearCompletedTransfers());
        
        // Quick actions
        document.querySelectorAll('.quick-action-btn').forEach(btn => {
            btn.addEventListener('click', (e) => this.handleQuickAction(e));
        });
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
        this.connectWebSocket();
        this.startHeartbeat();
    }

    updateUserGreeting() {
        document.getElementById('userGreeting').textContent = `Hello, ${this.currentUser}!`;
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
        this.showNotification(`Uploading ${files.length} file(s)...`, 'info');
        
        const formData = new FormData();
        for (let file of files) {
            formData.append('files', file);
            this.addTransfer({
                id: 'transfer-' + Date.now(),
                file_name: file.name,
                file_size: file.size,
                progress: 0,
                status: 'uploading',
                started_at: new Date().toISOString()
            });
        }

        try {
            const response = await fetch(this.apiBase + '/api/upload', {
                method: 'POST',
                headers: {
                    'Authorization': 'Bearer ' + this.token
                },
                body: formData
            });

            if (response.ok) {
                this.showNotification('Files uploaded successfully!', 'success');
                this.loadTransfers();
            } else {
                throw new Error('Upload failed');
            }
        } catch (error) {
            this.showNotification('Upload failed: ' + error.message, 'error');
        }
    }

    // Devices
    async loadDevices() {
        try {
            const response = await fetch(this.apiBase + '/api/devices');
            if (response.ok) {
                this.devices = await response.json();
                this.renderDevices();
            }
        } catch (error) {
            // Fallback demo data
            this.devices = [
                { id: '1', name: 'My Computer', ip: '127.0.0.1', os: 'Windows', status: 'online' },
                { id: '2', name: 'Phone', ip: '192.168.1.101', os: 'Android', status: 'online' },
                { id: '3', name: 'Tablet', ip: '192.168.1.102', os: 'iOS', status: 'online' }
            ];
            this.renderDevices();
        }
    }

    // In your app.js, update the renderDevices function:

renderDevices() {
    const container = document.getElementById('devicesList');
    const countElement = document.getElementById('devicesCount');
    
    // Filter out self device if you want, or show it differently
    const otherDevices = this.devices.filter(device => !device.is_self);
    countElement.textContent = otherDevices.length;
    
    if (otherDevices.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <div class="empty-icon">🔍</div>
                <p>No other devices found</p>
                <small>Make sure other devices are on the same network and running GoLANshare</small>
            </div>
        `;
        return;
    }
    
    container.innerHTML = otherDevices.map(device => `
        <div class="device-item">
            <div class="device-icon">${this.getDeviceIcon(device.os)}</div>
            <div class="device-info">
                <div class="device-name">${device.name}</div>
                <div class="device-details">${device.ip} • ${device.os}</div>
                <div class="device-status ${device.status}">${device.status}</div>
            </div>
            <button class="btn btn-sm btn-primary" onclick="app.sendToDevice('${device.id}')">
                Send File
            </button>
        </div>
    `).join('');
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
        this.showNotification('Preparing to send files...', 'info');
    }

    // Transfers
    async loadTransfers() {
        try {
            const response = await fetch(this.apiBase + '/api/transfers');
            if (response.ok) {
                this.transfers = await response.json();
                this.renderTransfers();
            }
        } catch (error) {
            this.renderTransfers(); // Will show empty state
        }
    }

    renderTransfers() {
        const container = document.getElementById('transfersList');
        const countElement = document.getElementById('transfersCount');
        
        if (!this.transfers || !Array.isArray(this.transfers)) {
            this.transfers = [];
        }
        
        const activeTransfers = this.transfers.filter(t => t.status !== 'completed');
        countElement.textContent = activeTransfers.length;
        
        if (this.transfers.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <div class="empty-icon">📤</div>
                    <p>No active transfers</p>
                    <small>Files you send or receive will appear here</small>
                </div>
            `;
            return;
        }
        
        container.innerHTML = this.transfers.map(transfer => `
            <div class="transfer-item">
                <div class="transfer-icon">${this.getFileIcon(transfer.file_name)}</div>
                <div class="transfer-info">
                    <div class="transfer-name">${transfer.file_name}</div>
                    <div class="transfer-details">
                        ${this.formatFileSize(transfer.file_size)} • ${transfer.status}
                    </div>
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: ${transfer.progress}%"></div>
                    </div>
                </div>
                <div class="transfer-actions">
                    <button class="btn btn-sm btn-outline">${transfer.status === 'paused' ? 'Resume' : 'Pause'}</button>
                </div>
            </div>
        `).join('');
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

    pauseAllTransfers() {
        this.showNotification('All transfers paused', 'info');
    }

    resumeAllTransfers() {
        this.showNotification('All transfers resumed', 'info');
    }

    clearCompletedTransfers() {
        this.transfers = this.transfers.filter(t => t.status !== 'completed');
        this.renderTransfers();
        this.showNotification('Completed transfers cleared', 'success');
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
                if (data.type === 'message') {
                    this.addChatMessage(data);
                }
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
            screenshot: () => this.takeScreenshot(),
            text: () => this.shareText(),
            folder: () => this.shareFolder()
        };
        
        if (actions[action]) {
            actions[action]();
        }
    }

    takeScreenshot() {
        this.showNotification('Screenshot feature coming soon!', 'info');
    }

    shareText() {
        const text = prompt('Enter text to share:');
        if (text) {
            this.showNotification('Text shared to devices', 'success');
        }
    }

    shareFolder() {
        this.showNotification('Folder sharing coming soon!', 'info');
    }

    // Utilities
    setupApp() {
        // Initialize counts
        document.getElementById('filesCount').textContent = '0';
        document.getElementById('chatCount').textContent = '0';
    }

    startHeartbeat() {
        setInterval(() => {
            this.loadDevices();
            this.loadTransfers();
        }, 10000); // Update every 10 seconds
    }

    formatFileSize(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
    }

    showNotification(message, type = 'info') {
        const notifications = document.getElementById('notifications');
        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.innerHTML = message;
        
        notifications.appendChild(notification);
        
        setTimeout(() => {
            notification.remove();
        }, 3000);
    }
}

// Initialize app when page loads
document.addEventListener('DOMContentLoaded', () => {
    window.app = new GoLANshare();
});