package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/mdns"
)

type Device struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	IP        string    `json:"ip"`
	LastSeen  time.Time `json:"last_seen"`
	OS        string    `json:"os"`
	Status    string    `json:"status"`
	IsSelf    bool      `json:"is_self"`
}

var devices []Device
var selfDevice Device

func startDiscovery() {
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
	
	selfDevice = Device{
		ID:       generateDeviceID(hostname, ip),
		Name:     hostname,
		IP:       ip,
		LastSeen: time.Now(),
		OS:       getOS(),
		Status:   "online",
		IsSelf:   true,
	}
	
	devices = []Device{selfDevice}
	if err := insertDevice(selfDevice); err != nil {
		log.Printf("Failed to insert self device: %v", err)
	}

	log.Printf("Self device registered: %s (%s) - %s", selfDevice.Name, selfDevice.IP, selfDevice.OS)

	// Start mDNS discovery
	go startMDNSDiscovery()
	
	// Start network scanning as fallback
	go startNetworkScan()
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

	// Start mDNS client for discovery
	entriesCh := make(chan *mdns.ServiceEntry, 8)
	
	go func() {
		for entry := range entriesCh {
			if entry.AddrV4 == nil {
				continue
			}
			
			// Don't add our own device
			if entry.AddrV4.String() == selfDevice.IP {
				continue
			}
			
			device := Device{
				ID:       generateDeviceID(entry.Name, entry.AddrV4.String()),
				Name:     entry.Name,
				IP:       entry.AddrV4.String(),
				LastSeen: time.Now(),
				OS:       guessOSFromName(entry.Name),
				Status:   "online",
				IsSelf:   false,
			}
			
			updateOrAddDevice(device)
		}
	}()

	// Continuous mDNS lookup
	for {
		if err := mdns.Lookup("_golanshare._tcp", entriesCh); err != nil {
			log.Printf("mDNS lookup error: %v", err)
		}
		time.Sleep(15 * time.Second) // Check every 15 seconds
	}
}

func startNetworkScan() {
	// Simple network scanning as fallback
	for {
		time.Sleep(30 * time.Second)
		
		// Get local network range
		_, err := getLocalIP()
		if err != nil {
			continue
		}
		
		// Simple network scan - you can expand this
		log.Printf("Scanning network for other GoLANshare instances...")
		
		// For now, just log that we're scanning
		// In a real implementation, you'd try to connect to common ports
		// on other devices in the same subnet
	}
}

func updateOrAddDevice(device Device) {
	for i, d := range devices {
		if d.IP == device.IP {
			// Update existing device
			devices[i].LastSeen = time.Now()
			devices[i].Status = "online"
			devices[i].Name = device.Name // Update name if changed
			if err := updateDevice(devices[i]); err != nil {
				log.Printf("Failed to update device %s: %v", d.ID, err)
			}
			log.Printf("Updated device: %s (%s)", device.Name, device.IP)
			return
		}
	}
	
	// Add new device
	devices = append(devices, device)
	if err := insertDevice(device); err != nil {
		log.Printf("Failed to insert device %s: %v", device.ID, err)
	}
	log.Printf("Discovered new device: %s (%s) - %s", device.Name, device.IP, device.OS)
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
	return fmt.Sprintf("%s-%s", name, ip)
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
	for {
		time.Sleep(2 * time.Minute) // Check every 2 minutes
		now := time.Now()
		
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
			}
		}
	}
}

// Export devices for API
func getDevices() []Device {
	// Return a copy to avoid concurrent modification
	result := make([]Device, len(devices))
	copy(result, devices)
	return result
}

func init() {
	go cleanupOldDevices()
}