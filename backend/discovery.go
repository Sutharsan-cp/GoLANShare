package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/mdns"
)

type Device struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	IP          string    `json:"ip"`
	LastSeen    time.Time `json:"last_seen"`
	OS          string    `json:"os"`
	Status      string    `json:"status"`
	IsSelf      bool      `json:"is_self"`
	Port        int       `json:"port"`
	Version     string    `json:"version"`
	Avatar      string    `json:"avatar,omitempty"`
	IsConnected bool      `json:"is_connected"`
	Distance    string    `json:"distance,omitempty"`
	DeviceType  string    `json:"device_type"`
}

var (
	devices      []Device
	devicesMutex sync.RWMutex
	selfDevice   Device
	scanSem      chan struct{} // Semaphore for limiting concurrent scans
)

const (
	maxConcurrentScans = 20
	scanTimeout        = 3 * time.Second
)

func startDiscovery() {
	// Initialize scan semaphore
	scanSem = make(chan struct{}, maxConcurrentScans)
	
	// Initialize self device
	ip, err := getLocalIP()
	if err != nil {
		log.Printf("Failed to get local IP: %v, using fallback", err)
		ip = "127.0.0.1"
	}
	
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("Failed to get hostname: %v, using fallback", err)
		hostname = "Unknown-Device"
	}
	
	port, _ := strconv.Atoi(config.Port)
	
	selfDevice = Device{
		ID:          generateDeviceID(hostname, ip),
		Name:        hostname,
		IP:          ip,
		LastSeen:    time.Now(),
		OS:          getOS(),
		Status:      "online",
		IsSelf:      true,
		Port:        port,
		Version:     "2.0.0",
		IsConnected: true,
		DeviceType:  getOS(),
	}
	
	devicesMutex.Lock()
	devices = []Device{selfDevice}
	devicesMutex.Unlock()
	
	if err := insertDevice(selfDevice); err != nil {
		log.Printf("Failed to insert self device: %v", err)
	}

	log.Printf("Self device registered: %s (%s) - %s", selfDevice.Name, selfDevice.IP, selfDevice.OS)

	// Start mDNS discovery
	go startMDNSDiscovery()
	
	// Start network scanning as fallback
	go startNetworkScan()
	
	// Start device cleanup
	go cleanupOldDevices()
}

func startMDNSDiscovery() {
	// Create mDNS service
	port, err := strconv.Atoi(config.Port)
	if err != nil {
		port = 8081
		log.Printf("Invalid port, using default: 8081")
	}

	service, err := mdns.NewMDNSService(
		selfDevice.Name,
		"_golanshare._tcp",
		"",
		"",
		port,
		[]net.IP{net.ParseIP(selfDevice.IP)},
		[]string{"GoLANshare File Sharing Service"},
	)
	
	if err != nil {
		log.Printf("Failed to create mDNS service: %v", err)
		return
	}

	// Start mDNS server
	server, err := mdns.NewServer(&mdns.Config{
		Zone: service,
	})
	
	if err != nil {
		log.Printf("Failed to start mDNS server: %v", err)
	} else {
		defer server.Shutdown()
		log.Printf("mDNS server started on port %d", port)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start mDNS client for discovery with controlled lifecycle
	entriesCh := make(chan *mdns.ServiceEntry, 50)
	
	// Start entry processor
	go func() {
		for {
			select {
			case entry, ok := <-entriesCh:
				if !ok {
					return // Channel closed
				}
				processMDNSEntry(entry)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Continuous mDNS lookup with context
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Use a separate goroutine for lookup to avoid blocking
			go func() {
				if err := mdns.Lookup("_golanshare._tcp", entriesCh); err != nil {
					log.Printf("mDNS lookup error: %v", err)
				}
			}()
		case <-ctx.Done():
			close(entriesCh)
			return
		}
	}
}

func processMDNSEntry(entry *mdns.ServiceEntry) {
	if entry.AddrV4 == nil {
		return
	}
	
	// Don't add our own device
	if entry.AddrV4.String() == selfDevice.IP {
		return
	}
	
	device := Device{
		ID:          generateDeviceID(entry.Name, entry.AddrV4.String()),
		Name:        entry.Name,
		IP:          entry.AddrV4.String(),
		LastSeen:    time.Now(),
		OS:          guessOSFromName(entry.Name),
		Status:      "online",
		IsSelf:      false,
		Port:        entry.Port,
		Version:     "2.0.0",
		IsConnected: true,
		DeviceType:  guessOSFromName(entry.Name),
	}
	
	updateOrAddDevice(device)
}

func startNetworkScan() {
	// Network scanning with better intervals
	ticker := time.NewTicker(60 * time.Second) // Reduced frequency
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Get local network range
			localIP, err := getLocalIP()
			if err != nil {
				log.Printf("Failed to get local IP for scanning: %v", err)
				continue
			}
			
			log.Printf("Scanning network for other GoLANshare instances...")
			
			// Scan local subnet for GoLANshare instances
			go scanSubnet(localIP)
		}
	}
}

func scanSubnet(localIP string) {
	// Parse IP to get network range
	ip := net.ParseIP(localIP)
	if ip == nil {
		return
	}
	
	// Get network interfaces to find subnet
	interfaces, err := net.Interfaces()
	if err != nil {
		return
	}
	
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		
		for _, addr := range addrs {
			var ipNet *net.IPNet
			switch v := addr.(type) {
			case *net.IPNet:
				ipNet = v
			}
			
			if ipNet == nil || ipNet.IP.To4() == nil {
				continue
			}
			
			// Check if our local IP is in this network
			if ipNet.Contains(ip) {
				// Scan this subnet with controlled concurrency
				go scanNetworkRange(ipNet)
				return
			}
		}
	}
}

func scanNetworkRange(ipNet *net.IPNet) {
	// Convert to IPv4
	ip := ipNet.IP.To4()
	if ip == nil {
		return
	}
	
	// Get network and broadcast addresses
	mask := ipNet.Mask
	network := ip.Mask(mask)
	
	// Use worker pool pattern for scanning
	var wg sync.WaitGroup
	scanned := 0
	
	log.Printf("Scanning subnet %s with mask %s", network.String(), mask.String())
	
	// Scan common IP ranges with controlled concurrency
	for i := 1; i < 255; i++ {
		targetIP := make(net.IP, 4)
		copy(targetIP, network)
		targetIP[3] = byte(i)
		
		// Skip our own IP
		if targetIP.String() == selfDevice.IP {
			continue
		}
		
		// Acquire semaphore
		scanSem <- struct{}{}
		wg.Add(1)
		scanned++
		
		go func(ip string) {
			defer wg.Done()
			defer func() { <-scanSem }()
			
			checkGoLANshareInstance(ip)
		}(targetIP.String())
	}
	
	wg.Wait()
	log.Printf("Completed scanning %d IP addresses", scanned)
}

func checkGoLANshareInstance(ip string) {
	ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
	defer cancel()
	
	// Try to connect to potential GoLANshare instance
	port, _ := strconv.Atoi(config.Port)
	address := fmt.Sprintf("%s:%d", ip, port)
	
	// Use context for dial timeout
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		return // No service on this IP
	}
	conn.Close()
	
	log.Printf("Found service at %s, checking if it's GoLANshare...", address)
	
	// Create HTTP client with timeout
	client := &http.Client{Timeout: 2 * time.Second}
	
	// Try device info endpoint first
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://%s/api/device/info", address), nil)
	if err != nil {
		return
	}
	
	resp, err := client.Do(req)
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()
		
		var deviceInfo map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&deviceInfo); err == nil {
			if deviceData, ok := deviceInfo["device"].(map[string]interface{}); ok {
				// Extract device information
				name := fmt.Sprintf("%v", deviceData["name"])
				os := fmt.Sprintf("%v", deviceData["os"])
				
				device := Device{
					ID:          generateDeviceID(name, ip),
					Name:        name,
					IP:          ip,
					LastSeen:    time.Now(),
					OS:          os,
					Status:      "online",
					IsSelf:      false,
					Port:        port,
					Version:     "2.0.0",
					IsConnected: true,
					DeviceType:  os,
				}
				
				updateOrAddDevice(device)
				log.Printf("Found GoLANshare instance via device/info: %s (%s)", name, ip)
				return
			}
		}
	}
	
	// Try devices endpoint
	req, err = http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://%s/api/devices", address), nil)
	if err == nil {
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			
			hostname := fmt.Sprintf("GoLANshare-%s", strings.Replace(ip, ".", "-", -1))
			
			device := Device{
				ID:          generateDeviceID(hostname, ip),
				Name:        hostname,
				IP:          ip,
				LastSeen:    time.Now(),
				OS:          "Unknown",
				Status:      "online",
				IsSelf:      false,
				Port:        port,
				Version:     "2.0.0",
				IsConnected: true,
				DeviceType:  "Unknown",
			}
			
			updateOrAddDevice(device)
			log.Printf("Found GoLANshare instance via devices endpoint: %s", ip)
			return
		}
	}
	
	// Try root endpoint to see if it's a web server
	req, err = http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://%s/", address), nil)
	if err == nil {
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			
			// Check if response contains GoLANshare indicators
			bodyReq, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://%s/", address), nil)
			if err == nil {
				bodyResp, err := client.Do(bodyReq)
				if err == nil {
					defer bodyResp.Body.Close()
					body := make([]byte, 1024)
					n, _ := bodyResp.Body.Read(body)
					bodyStr := string(body[:n])
					
					if strings.Contains(bodyStr, "GoLANshare") || strings.Contains(bodyStr, "golanshare") {
						hostname := fmt.Sprintf("GoLANshare-%s", strings.Replace(ip, ".", "-", -1))
						
						device := Device{
							ID:          generateDeviceID(hostname, ip),
							Name:        hostname,
							IP:          ip,
							LastSeen:    time.Now(),
							OS:          "Unknown",
							Status:      "online",
							IsSelf:      false,
							Port:        port,
							Version:     "2.0.0",
							IsConnected: true,
							DeviceType:  "Unknown",
						}
						
						updateOrAddDevice(device)
						log.Printf("Found GoLANshare instance via root page: %s", ip)
						return
					}
				}
			}
		}
	}
	
	log.Printf("Service at %s is not GoLANshare", address)
}

func updateOrAddDevice(device Device) {
	// Set default values
	if device.Port == 0 {
		port, _ := strconv.Atoi(config.Port)
		device.Port = port
	}
	if device.Version == "" {
		device.Version = "2.0.0"
	}
	if device.DeviceType == "" {
		device.DeviceType = device.OS
	}
	
	devicesMutex.Lock()
	defer devicesMutex.Unlock()
	
	for i, d := range devices {
		if d.IP == device.IP || d.ID == device.ID {
			// Update existing device
			devices[i].LastSeen = time.Now()
			devices[i].Status = "online"
			devices[i].Name = device.Name
			devices[i].OS = device.OS
			devices[i].Port = device.Port
			devices[i].Version = device.Version
			devices[i].DeviceType = device.DeviceType
			devices[i].IsConnected = true
			
			if err := updateDevice(devices[i]); err != nil {
				log.Printf("Failed to update device %s: %v", d.ID, err)
			}
			log.Printf("Updated device: %s (%s)", device.Name, device.IP)
			
			// Broadcast device update to all clients
			go broadcastDeviceUpdate()
			return
		}
	}
	
	// Add new device
	device.IsConnected = true
	devices = append(devices, device)
	if err := insertDevice(device); err != nil {
		log.Printf("Failed to insert device %s: %v", device.ID, err)
	}
	log.Printf("Discovered new device: %s (%s) - %s", device.Name, device.IP, device.OS)
	
	// Broadcast device update to all clients
	go broadcastDeviceUpdate()
}

func broadcastDeviceUpdate() {
	// Send device list update to all connected clients
	deviceList := getDevices()
	message := map[string]interface{}{
		"type":    "devices_updated",
		"devices": deviceList,
		"count":   len(deviceList) - 1, // Exclude self
	}
	
	messageData, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling device update: %v", err)
		return
	}
	
	// Use non-blocking send with timeout protection
	select {
	case hub.broadcast <- messageData:
		// Message sent successfully
	case <-time.After(100 * time.Millisecond):
		log.Printf("Timeout broadcasting device update")
	}
}

func getLocalIP() (string, error) {
	// Try multiple methods to get local IP
	
	// Method 1: Dial to external IP to get local IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		return localAddr.IP.String(), nil
	}
	
	// Method 2: Get from interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	
	for _, iface := range interfaces {
		// Skip loopback and inactive interfaces
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			
			// Skip IPv6 and loopback
			if ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}
			
			return ip.String(), nil
		}
	}
	
	return "", fmt.Errorf("no suitable IP address found")
}

func generateDeviceID(name, ip string) string {
	// Create a consistent ID based on name and IP
	return fmt.Sprintf("%s-%s", strings.ReplaceAll(name, " ", "_"), ip)
}

func getOS() string {
	// Detect operating system
	switch {
	case os.Getenv("OS") == "Windows_NT":
		return "Windows"
	case runtime.GOOS == "darwin":
		return "macOS"
	case runtime.GOOS == "linux":
		return "Linux"
	case runtime.GOOS == "android":
		return "Android"
	case runtime.GOOS == "ios":
		return "iOS"
	default:
		return "Unknown"
	}
}

func guessOSFromName(name string) string {
	name = strings.ToLower(name)
	
	switch {
	case strings.Contains(name, "windows") || strings.Contains(name, "win"):
		return "Windows"
	case strings.Contains(name, "mac") || strings.Contains(name, "apple"):
		return "macOS"
	case strings.Contains(name, "linux"):
		return "Linux"
	case strings.Contains(name, "android"):
		return "Android"
	case strings.Contains(name, "iphone") || strings.Contains(name, "ipad") || strings.Contains(name, "ios"):
		return "iOS"
	default:
		return "Unknown"
	}
}

func cleanupOldDevices() {
	ticker := time.NewTicker(2 * time.Minute) // Check every 2 minutes
	defer ticker.Stop()
	
	for {
		<-ticker.C
		now := time.Now()
		
		devicesMutex.Lock()
		removedCount := 0
		for i := len(devices) - 1; i >= 0; i-- {
			// Never remove self device
			if devices[i].IsSelf {
				continue
			}
			
			// Remove devices not seen for 5 minutes
			if now.Sub(devices[i].LastSeen) > 5*time.Minute {
				log.Printf("Removing offline device: %s (%s)", devices[i].Name, devices[i].IP)
				if err := deleteDevice(devices[i].ID); err != nil {
					log.Printf("Failed to delete device %s: %v", devices[i].ID, err)
				}
				devices = append(devices[:i], devices[i+1:]...)
				removedCount++
			}
		}
		devicesMutex.Unlock()
		
		if removedCount > 0 {
			go broadcastDeviceUpdate()
		}
	}
}

// Export devices for API
func getDevices() []Device {
	devicesMutex.RLock()
	defer devicesMutex.RUnlock()
	
	// Return a copy to avoid concurrent modification
	result := make([]Device, len(devices))
	copy(result, devices)
	return result
}