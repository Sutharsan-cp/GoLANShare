"use client"

import type React from "react"

import { useState } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  Upload,
  Users,
  Wifi,
  Zap,
  FileText,
  ImageIcon,
  Video,
  Music,
  Archive,
  Smartphone,
  Monitor,
  Laptop,
  Send,
  QrCode,
  History,
  Search,
  Settings,
  Lock,
  Unlock,
  Share2,
  MessageCircle,
  CheckCircle,
  AlertCircle,
  Clock,
} from "lucide-react"
import { useToast } from "@/hooks/use-toast"

interface Device {
  id: string
  name: string
  ip: string
  type: "desktop" | "laptop" | "mobile"
  status: "online" | "offline"
  lastSeen: string
}

interface FileTransfer {
  id: string
  fileName: string
  fileSize: number
  progress: number
  status: "pending" | "transferring" | "completed" | "failed"
  from: string
  to: string
  timestamp: string
  type: string
}

interface ChatMessage {
  id: string
  from: string
  message: string
  timestamp: string
  type: "text" | "file" | "system"
}

export default function LANFileSharePro() {
  const [devices, setDevices] = useState<Device[]>([
    { id: "1", name: "Prince's Desktop", ip: "192.168.1.100", type: "desktop", status: "online", lastSeen: "now" },
    { id: "2", name: "Loki's Laptop", ip: "192.168.1.101", type: "laptop", status: "online", lastSeen: "2 min ago" },
    { id: "3", name: "Barath's Phone", ip: "192.168.1.102", type: "mobile", status: "offline", lastSeen: "5 min ago" },
  ])

  const [transfers, setTransfers] = useState<FileTransfer[]>([
    {
      id: "1",
      fileName: "project-docs.pdf",
      fileSize: 2048000,
      progress: 100,
      status: "completed",
      from: "Prince's Desktop",
      to: "Loki's Laptop",
      timestamp: "2 min ago",
      type: "pdf",
    },
    {
      id: "2",
      fileName: "vacation-photos.zip",
      fileSize: 15728640,
      progress: 65,
      status: "transferring",
      from: "Loki's Laptop",
      to: "Prince's Desktop",
      timestamp: "now",
      type: "archive",
    },
  ])

  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([
    { id: "1", from: "Prince", message: "Hey, I'm sharing the project files", timestamp: "10:30 AM", type: "text" },
    {
      id: "2",
      from: "Loki",
      message: "Thanks! Sending you the photos from yesterday",
      timestamp: "10:32 AM",
      type: "text",
    },
    { id: "3", from: "System", message: "Barath joined the network", timestamp: "10:35 AM", type: "system" },
  ])

  const [selectedFiles, setSelectedFiles] = useState<File[]>([])
  const [isScanning, setIsScanning] = useState(false)
  const [encryptionEnabled, setEncryptionEnabled] = useState(true)
  const [newMessage, setNewMessage] = useState("")
  const [searchQuery, setSearchQuery] = useState("")

  const { toast } = useToast()

  const getDeviceIcon = (type: string) => {
    switch (type) {
      case "desktop":
        return <Monitor className="h-4 w-4" />
      case "laptop":
        return <Laptop className="h-4 w-4" />
      case "mobile":
        return <Smartphone className="h-4 w-4" />
      default:
        return <Monitor className="h-4 w-4" />
    }
  }

  const getFileIcon = (type: string) => {
    if (type.includes("image")) return <ImageIcon className="h-4 w-4" />
    if (type.includes("video")) return <Video className="h-4 w-4" />
    if (type.includes("audio")) return <Music className="h-4 w-4" />
    if (type.includes("archive") || type.includes("zip")) return <Archive className="h-4 w-4" />
    return <FileText className="h-4 w-4" />
  }

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return "0 Bytes"
    const k = 1024
    const sizes = ["Bytes", "KB", "MB", "GB"]
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Number.parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i]
  }

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files || [])
    setSelectedFiles(files)
    if (files.length > 0) {
      toast({
        title: "Files Selected",
        description: `${files.length} file(s) ready to share`,
      })
    }
  }

  const handleScanNetwork = () => {
    setIsScanning(true)
    setTimeout(() => {
      setIsScanning(false)
      toast({
        title: "Network Scan Complete",
        description: `Found ${devices.length} devices on the network`,
      })
    }, 3000)
  }

  const handleSendFile = (deviceId: string) => {
    if (selectedFiles.length === 0) {
      toast({
        title: "No Files Selected",
        description: "Please select files to share first",
        variant: "destructive",
      })
      return
    }

    const device = devices.find((d) => d.id === deviceId)
    selectedFiles.forEach((file) => {
      const newTransfer: FileTransfer = {
        id: Date.now().toString() + Math.random(),
        fileName: file.name,
        fileSize: file.size,
        progress: 0,
        status: "pending",
        from: "Your Device",
        to: device?.name || "Unknown",
        timestamp: "now",
        type: file.type,
      }

      setTransfers((prev) => [newTransfer, ...prev])

      // Simulate file transfer progress
      let progress = 0
      const interval = setInterval(() => {
        progress += Math.random() * 15
        if (progress >= 100) {
          progress = 100
          clearInterval(interval)
          setTransfers((prev) =>
            prev.map((t) => (t.id === newTransfer.id ? { ...t, progress: 100, status: "completed" } : t)),
          )
          toast({
            title: "Transfer Complete",
            description: `${file.name} sent successfully`,
          })
        } else {
          setTransfers((prev) =>
            prev.map((t) => (t.id === newTransfer.id ? { ...t, progress, status: "transferring" } : t)),
          )
        }
      }, 500)
    })

    setSelectedFiles([])
  }

  const handleSendMessage = () => {
    if (!newMessage.trim()) return

    const message: ChatMessage = {
      id: Date.now().toString(),
      from: "You",
      message: newMessage,
      timestamp: new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
      type: "text",
    }

    setChatMessages((prev) => [...prev, message])
    setNewMessage("")
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 p-4">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-4xl font-bold text-gray-900 mb-2">LAN File Share Pro</h1>
              <p className="text-gray-600">Secure, fast, and intelligent file sharing across your local network</p>
            </div>
            <div className="flex items-center space-x-4">
              <Badge variant={encryptionEnabled ? "default" : "secondary"} className="flex items-center space-x-1">
                {encryptionEnabled ? <Lock className="h-3 w-3" /> : <Unlock className="h-3 w-3" />}
                <span>{encryptionEnabled ? "Encrypted" : "Unencrypted"}</span>
              </Badge>
              <Button variant="outline" size="sm">
                <Settings className="h-4 w-4 mr-2" />
                Settings
              </Button>
            </div>
          </div>
        </div>

        {/* Stats Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
          <Card>
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-600">Active Devices</p>
                  <p className="text-2xl font-bold text-gray-900">
                    {devices.filter((d) => d.status === "online").length}
                  </p>
                </div>
                <Users className="h-8 w-8 text-blue-600" />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-600">Files Shared</p>
                  <p className="text-2xl font-bold text-gray-900">
                    {transfers.filter((t) => t.status === "completed").length}
                  </p>
                </div>
                <Share2 className="h-8 w-8 text-green-600" />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-600">Data Transferred</p>
                  <p className="text-2xl font-bold text-gray-900">127 MB</p>
                </div>
                <Zap className="h-8 w-8 text-yellow-600" />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-600">Network Status</p>
                  <p className="text-2xl font-bold text-green-600">Online</p>
                </div>
                <Wifi className="h-8 w-8 text-blue-600" />
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Main Content */}
        <Tabs defaultValue="share" className="space-y-6">
          <TabsList className="grid w-full grid-cols-4">
            <TabsTrigger value="share">File Sharing</TabsTrigger>
            <TabsTrigger value="devices">Network Devices</TabsTrigger>
            <TabsTrigger value="transfers">Transfer History</TabsTrigger>
            <TabsTrigger value="chat">Live Chat</TabsTrigger>
          </TabsList>

          {/* File Sharing Tab */}
          <TabsContent value="share" className="space-y-6">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* File Upload */}
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center space-x-2">
                    <Upload className="h-5 w-5" />
                    <span>Share Files</span>
                  </CardTitle>
                  <CardDescription>Select files to share with devices on your network</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="border-2 border-dashed border-gray-300 rounded-lg p-8 text-center">
                    <Upload className="h-12 w-12 text-gray-400 mx-auto mb-4" />
                    <p className="text-lg font-medium text-gray-900 mb-2">Drop files here or click to browse</p>
                    <p className="text-sm text-gray-500 mb-4">Supports all file types up to 1GB each</p>
                    <input type="file" multiple onChange={handleFileSelect} className="hidden" id="file-upload" />
                    <Button asChild>
                      <label htmlFor="file-upload" className="cursor-pointer">
                        Select Files
                      </label>
                    </Button>
                  </div>

                  {selectedFiles.length > 0 && (
                    <div className="space-y-2">
                      <h4 className="font-medium">Selected Files:</h4>
                      {selectedFiles.map((file, index) => (
                        <div key={index} className="flex items-center justify-between p-2 bg-gray-50 rounded">
                          <div className="flex items-center space-x-2">
                            {getFileIcon(file.type)}
                            <span className="text-sm">{file.name}</span>
                          </div>
                          <span className="text-xs text-gray-500">{formatFileSize(file.size)}</span>
                        </div>
                      ))}
                    </div>
                  )}
                </CardContent>
              </Card>

              {/* Quick Actions */}
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center space-x-2">
                    <Zap className="h-5 w-5" />
                    <span>Quick Actions</span>
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <Button
                    onClick={handleScanNetwork}
                    disabled={isScanning}
                    className="w-full bg-transparent"
                    variant="outline"
                  >
                    <Search className="h-4 w-4 mr-2" />
                    {isScanning ? "Scanning Network..." : "Scan for Devices"}
                  </Button>

                  <Button className="w-full bg-transparent" variant="outline">
                    <QrCode className="h-4 w-4 mr-2" />
                    Generate QR Code
                  </Button>

                  <Button onClick={() => setEncryptionEnabled(!encryptionEnabled)} className="w-full" variant="outline">
                    {encryptionEnabled ? <Lock className="h-4 w-4 mr-2" /> : <Unlock className="h-4 w-4 mr-2" />}
                    {encryptionEnabled ? "Disable" : "Enable"} Encryption
                  </Button>

                  <div className="pt-4 border-t">
                    <h4 className="font-medium mb-2">Network Info</h4>
                    <div className="text-sm text-gray-600 space-y-1">
                      <p>IP Address: 192.168.1.100</p>
                      <p>Port: 8080</p>
                      <p>Protocol: TCP/WebSocket</p>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          {/* Network Devices Tab */}
          <TabsContent value="devices" className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center justify-between">
                  <div className="flex items-center space-x-2">
                    <Users className="h-5 w-5" />
                    <span>Network Devices</span>
                  </div>
                  <Button onClick={handleScanNetwork} disabled={isScanning} size="sm">
                    <Search className="h-4 w-4 mr-2" />
                    {isScanning ? "Scanning..." : "Refresh"}
                  </Button>
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {devices.map((device) => (
                    <Card key={device.id} className="relative">
                      <CardContent className="p-4">
                        <div className="flex items-center justify-between mb-3">
                          <div className="flex items-center space-x-2">
                            {getDeviceIcon(device.type)}
                            <span className="font-medium">{device.name}</span>
                          </div>
                          <Badge variant={device.status === "online" ? "default" : "secondary"}>{device.status}</Badge>
                        </div>

                        <div className="text-sm text-gray-600 mb-3">
                          <p>IP: {device.ip}</p>
                          <p>Last seen: {device.lastSeen}</p>
                        </div>

                        {device.status === "online" && (
                          <div className="flex space-x-2">
                            <Button
                              size="sm"
                              onClick={() => handleSendFile(device.id)}
                              disabled={selectedFiles.length === 0}
                            >
                              <Send className="h-3 w-3 mr-1" />
                              Send
                            </Button>
                            <Button size="sm" variant="outline">
                              <MessageCircle className="h-3 w-3 mr-1" />
                              Chat
                            </Button>
                          </div>
                        )}
                      </CardContent>
                    </Card>
                  ))}
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          {/* Transfer History Tab */}
          <TabsContent value="transfers" className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center space-x-2">
                  <History className="h-5 w-5" />
                  <span>Transfer History</span>
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {transfers.map((transfer) => (
                    <div key={transfer.id} className="border rounded-lg p-4">
                      <div className="flex items-center justify-between mb-2">
                        <div className="flex items-center space-x-3">
                          {getFileIcon(transfer.type)}
                          <div>
                            <p className="font-medium">{transfer.fileName}</p>
                            <p className="text-sm text-gray-600">
                              {transfer.from} → {transfer.to}
                            </p>
                          </div>
                        </div>
                        <div className="text-right">
                          <Badge
                            variant={
                              transfer.status === "completed"
                                ? "default"
                                : transfer.status === "transferring"
                                  ? "secondary"
                                  : transfer.status === "failed"
                                    ? "destructive"
                                    : "outline"
                            }
                          >
                            {transfer.status === "completed" && <CheckCircle className="h-3 w-3 mr-1" />}
                            {transfer.status === "transferring" && <Clock className="h-3 w-3 mr-1" />}
                            {transfer.status === "failed" && <AlertCircle className="h-3 w-3 mr-1" />}
                            {transfer.status}
                          </Badge>
                          <p className="text-xs text-gray-500 mt-1">{transfer.timestamp}</p>
                        </div>
                      </div>

                      <div className="flex items-center justify-between">
                        <div className="flex-1 mr-4">
                          <Progress value={transfer.progress} className="h-2" />
                        </div>
                        <div className="text-sm text-gray-600">
                          {formatFileSize(transfer.fileSize)} • {transfer.progress}%
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          {/* Live Chat Tab */}
          <TabsContent value="chat" className="space-y-6">
            <Card className="h-96">
              <CardHeader>
                <CardTitle className="flex items-center space-x-2">
                  <MessageCircle className="h-5 w-5" />
                  <span>Live Chat</span>
                </CardTitle>
              </CardHeader>
              <CardContent className="flex flex-col h-full">
                <ScrollArea className="flex-1 mb-4">
                  <div className="space-y-3">
                    {chatMessages.map((message) => (
                      <div
                        key={message.id}
                        className={`flex ${message.from === "You" ? "justify-end" : "justify-start"}`}
                      >
                        <div
                          className={`max-w-xs lg:max-w-md px-3 py-2 rounded-lg ${
                            message.from === "You"
                              ? "bg-blue-600 text-white"
                              : message.type === "system"
                                ? "bg-gray-100 text-gray-600 text-center"
                                : "bg-gray-100 text-gray-900"
                          }`}
                        >
                          {message.type !== "system" && <p className="text-xs opacity-75 mb-1">{message.from}</p>}
                          <p className="text-sm">{message.message}</p>
                          <p className="text-xs opacity-75 mt-1">{message.timestamp}</p>
                        </div>
                      </div>
                    ))}
                  </div>
                </ScrollArea>

                <div className="flex space-x-2">
                  <Input
                    placeholder="Type a message..."
                    value={newMessage}
                    onChange={(e) => setNewMessage(e.target.value)}
                    onKeyPress={(e) => e.key === "Enter" && handleSendMessage()}
                  />
                  <Button onClick={handleSendMessage}>
                    <Send className="h-4 w-4" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  )
}
