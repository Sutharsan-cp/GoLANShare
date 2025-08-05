import type { NextRequest } from "next/server"
import { WebSocketServer } from "ws"

const wss = new WebSocketServer({ port: 8080 })

interface ConnectedClient {
  ws: any
  id: string
  deviceName: string
  ip: string
}

const clients = new Map<string, ConnectedClient>()

wss.on("connection", (ws, req) => {
  const clientId = generateClientId()
  const clientIP = req.socket.remoteAddress || "unknown"

  console.log(`New client connected: ${clientId} from ${clientIP}`)

  clients.set(clientId, {
    ws,
    id: clientId,
    deviceName: `Device-${clientId.substring(0, 8)}`,
    ip: clientIP,
  })

  // Send welcome message
  ws.send(
    JSON.stringify({
      type: "welcome",
      clientId,
      message: "Connected to LAN File Share Pro",
    }),
  )

  // Broadcast device list update
  broadcastDeviceList()

  ws.on("message", (data: Buffer) => {
    try {
      const message = JSON.parse(data.toString())
      handleMessage(clientId, message)
    } catch (error) {
      console.error("Invalid message format:", error)
    }
  })

  ws.on("close", () => {
    console.log(`Client disconnected: ${clientId}`)
    clients.delete(clientId)
    broadcastDeviceList()
  })

  ws.on("error", (error: Error) => {
    console.error(`WebSocket error for client ${clientId}:`, error)
  })
})

function generateClientId(): string {
  return Math.random().toString(36).substring(2, 15) + Math.random().toString(36).substring(2, 15)
}

function handleMessage(clientId: string, message: any) {
  const client = clients.get(clientId)
  if (!client) return

  switch (message.type) {
    case "chat":
      broadcastMessage(
        {
          type: "chat",
          from: client.deviceName,
          message: message.content,
          timestamp: new Date().toISOString(),
        },
        clientId,
      )
      break

    case "file_transfer_request":
      const targetClient = clients.get(message.targetId)
      if (targetClient) {
        targetClient.ws.send(
          JSON.stringify({
            type: "file_transfer_request",
            from: client.deviceName,
            fromId: clientId,
            fileName: message.fileName,
            fileSize: message.fileSize,
            fileId: message.fileId,
          }),
        )
      }
      break

    case "file_transfer_response":
      const requesterClient = clients.get(message.requesterId)
      if (requesterClient) {
        requesterClient.ws.send(
          JSON.stringify({
            type: "file_transfer_response",
            accepted: message.accepted,
            fileId: message.fileId,
          }),
        )
      }
      break

    case "device_name_update":
      client.deviceName = message.deviceName
      broadcastDeviceList()
      break
  }
}

function broadcastMessage(message: any, excludeClientId?: string) {
  const messageStr = JSON.stringify(message)
  clients.forEach((client, clientId) => {
    if (clientId !== excludeClientId && client.ws.readyState === 1) {
      client.ws.send(messageStr)
    }
  })
}

function broadcastDeviceList() {
  const deviceList = Array.from(clients.values()).map((client) => ({
    id: client.id,
    name: client.deviceName,
    ip: client.ip,
    status: "online",
  }))

  broadcastMessage({
    type: "device_list_update",
    devices: deviceList,
  })
}

export async function GET(request: NextRequest) {
  return new Response("WebSocket server is running on port 8080", {
    status: 200,
    headers: {
      "Content-Type": "text/plain",
    },
  })
}
