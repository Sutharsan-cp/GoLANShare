#!/usr/bin/env python3
"""
Advanced Network Scanner for LAN File Share Pro
Discovers devices on the local network with detailed information
"""

import socket
import subprocess
import threading
import json
import time
from concurrent.futures import ThreadPoolExecutor
import netifaces
import nmap

class NetworkScanner:
    def __init__(self):
        self.devices = []
        self.local_ip = self.get_local_ip()
        self.network_range = self.get_network_range()
        
    def get_local_ip(self):
        """Get the local IP address"""
        try:
            # Connect to a remote address to determine local IP
            s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            s.connect(("8.8.8.8", 80))
            local_ip = s.getsockname()[0]
            s.close()
            return local_ip
        except Exception:
            return "127.0.0.1"
    
    def get_network_range(self):
        """Calculate network range from local IP"""
        ip_parts = self.local_ip.split('.')
        network_base = '.'.join(ip_parts[:3])
        return f"{network_base}.0/24"
    
    def ping_host(self, ip):
        """Ping a single host to check if it's alive"""
        try:
            # Use ping command based on OS
            import platform
            param = "-n" if platform.system().lower() == "windows" else "-c"
            command = ["ping", param, "1", "-W", "1000", ip]
            
            result = subprocess.run(command, capture_output=True, text=True, timeout=2)
            return result.returncode == 0
        except Exception:
            return False
    
    def get_hostname(self, ip):
        """Get hostname for an IP address"""
        try:
            hostname = socket.gethostbyaddr(ip)[0]
            return hostname
        except Exception:
            return None
    
    def get_mac_address(self, ip):
        """Get MAC address for an IP address"""
        try:
            # Try ARP table lookup
            result = subprocess.run(["arp", "-n", ip], capture_output=True, text=True)
            if result.returncode == 0:
                lines = result.stdout.split('\n')
                for line in lines:
                    if ip in line:
                        parts = line.split()
                        for part in parts:
                            if ':' in part and len(part) == 17:
                                return part
        except Exception:
            pass
        return None
    
    def detect_os(self, ip):
        """Detect operating system using nmap"""
        try:
            nm = nmap.PortScanner()
            result = nm.scan(ip, arguments='-O')
            
            if ip in result['scan']:
                if 'osmatch' in result['scan'][ip]:
                    matches = result['scan'][ip]['osmatch']
                    if matches:
                        return matches[0]['name']
        except Exception:
            pass
        return "Unknown"
    
    def scan_ports(self, ip, ports=[22, 80, 443, 8080, 3000]):
        """Scan common ports on a host"""
        open_ports = []
        for port in ports:
            try:
                sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                sock.settimeout(1)
                result = sock.connect_ex((ip, port))
                if result == 0:
                    open_ports.append(port)
                sock.close()
            except Exception:
                pass
        return open_ports
    
    def scan_single_host(self, ip):
        """Comprehensive scan of a single host"""
        if not self.ping_host(ip):
            return None
        
        device_info = {
            'ip': ip,
            'hostname': self.get_hostname(ip),
            'mac_address': self.get_mac_address(ip),
            'os': self.detect_os(ip),
            'open_ports': self.scan_ports(ip),
            'last_seen': time.strftime('%Y-%m-%d %H:%M:%S'),
            'status': 'online'
        }
        
        # Determine device type based on open ports and hostname
        device_info['device_type'] = self.determine_device_type(device_info)
        
        return device_info
    
    def determine_device_type(self, device_info):
        """Determine device type based on available information"""
        hostname = device_info.get('hostname', '').lower()
        open_ports = device_info.get('open_ports', [])
        
        if any(keyword in hostname for keyword in ['phone', 'android', 'iphone']):
            return 'mobile'
        elif any(keyword in hostname for keyword in ['laptop', 'macbook']):
            return 'laptop'
        elif 80 in open_ports or 443 in open_ports:
            return 'server'
        elif 22 in open_ports:
            return 'server'
        else:
            return 'desktop'
    
    def scan_network(self, max_threads=50):
        """Scan the entire network range"""
        print(f"Scanning network range: {self.network_range}")
        print(f"Local IP: {self.local_ip}")
        
        # Generate IP range
        network_base = '.'.join(self.local_ip.split('.')[:-1])
        ip_range = [f"{network_base}.{i}" for i in range(1, 255)]
        
        # Use ThreadPoolExecutor for concurrent scanning
        with ThreadPoolExecutor(max_workers=max_threads) as executor:
            results = list(executor.map(self.scan_single_host, ip_range))
        
        # Filter out None results
        self.devices = [device for device in results if device is not None]
        
        return self.devices
    
    def save_results(self, filename='network_scan_results.json'):
        """Save scan results to JSON file"""
        scan_data = {
            'scan_time': time.strftime('%Y-%m-%d %H:%M:%S'),
            'local_ip': self.local_ip,
            'network_range': self.network_range,
            'devices_found': len(self.devices),
            'devices': self.devices
        }
        
        with open(filename, 'w') as f:
            json.dump(scan_data, f, indent=2)
        
        print(f"Results saved to {filename}")
    
    def print_results(self):
        """Print scan results in a formatted way"""
        print(f"\n{'='*60}")
        print(f"NETWORK SCAN RESULTS")
        print(f"{'='*60}")
        print(f"Devices found: {len(self.devices)}")
        print(f"Network range: {self.network_range}")
        print(f"{'='*60}")
        
        for device in self.devices:
            print(f"\nIP Address: {device['ip']}")
            print(f"Hostname: {device['hostname'] or 'Unknown'}")
            print(f"MAC Address: {device['mac_address'] or 'Unknown'}")
            print(f"Device Type: {device['device_type']}")
            print(f"OS: {device['os']}")
            print(f"Open Ports: {', '.join(map(str, device['open_ports'])) or 'None detected'}")
            print(f"Last Seen: {device['last_seen']}")
            print("-" * 40)

def main():
    """Main function to run the network scanner"""
    print("LAN File Share Pro - Network Scanner")
    print("Discovering devices on your local network...")
    
    scanner = NetworkScanner()
    devices = scanner.scan_network()
    
    scanner.print_results()
    scanner.save_results()
    
    return devices

if __name__ == "__main__":
    main()
