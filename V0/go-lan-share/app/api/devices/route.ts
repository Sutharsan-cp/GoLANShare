import { NextResponse } from "next/server"
import { exec } from "child_process"
import { promisify } from "util"

const execAsync = promisify(exec)

interface NetworkDevice {
  ip: string
  mac?: string
  hostname?: string
  vendor?: string
}

export async function GET() {
  try {
    // Scan local network for devices
    const devices = await scanNetwork()

    return NextResponse.json({
      success: true,
      devices,
      timestamp: new Date().toISOString(),
    })
  } catch (error) {
    console.error("Network scan error:", error)
    return NextResponse.json({ error: "Network scan failed" }, { status: 500 })
  }
}

async function scanNetwork(): Promise<NetworkDevice[]> {
  try {
    // Get local IP range
    const { stdout: ipOutput } = await execAsync("hostname -I")
    const localIP = ipOutput.trim().split(" ")[0]
    const networkBase = localIP.substring(0, localIP.lastIndexOf("."))

    // Ping sweep to find active devices
    const devices: NetworkDevice[] = []
    const promises = []

    for (let i = 1; i <= 254; i++) {
      const ip = `${networkBase}.${i}`
      promises.push(
        execAsync(`ping -c 1 -W 1 ${ip}`)
          .then(() => {
            devices.push({ ip })
            return getDeviceInfo(ip)
          })
          .then((info) => {
            if (info) {
              const deviceIndex = devices.findIndex((d) => d.ip === ip)
              if (deviceIndex !== -1) {
                devices[deviceIndex] = { ...devices[deviceIndex], ...info }
              }
            }
          })
          .catch(() => {
            // Device not reachable, ignore
          }),
      )
    }

    await Promise.allSettled(promises)
    return devices
  } catch (error) {
    console.error("Network scan error:", error)
    return []
  }
}

async function getDeviceInfo(ip: string): Promise<Partial<NetworkDevice> | null> {
  try {
    // Try to get hostname
    const { stdout: hostnameOutput } = await execAsync(`nslookup ${ip}`)
    const hostnameMatch = hostnameOutput.match(/name = (.+)\./)
    const hostname = hostnameMatch ? hostnameMatch[1] : undefined

    // Try to get MAC address
    const { stdout: arpOutput } = await execAsync(`arp -n ${ip}`)
    const macMatch = arpOutput.match(/([0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2})/i)
    const mac = macMatch ? macMatch[1] : undefined

    return { hostname, mac }
  } catch (error) {
    return null
  }
}
