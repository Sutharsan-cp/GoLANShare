import { type NextRequest, NextResponse } from "next/server"
import { writeFile, mkdir } from "fs/promises"
import { join } from "path"
import crypto from "crypto"

export async function POST(request: NextRequest) {
  try {
    const formData = await request.formData()
    const file = formData.get("file") as File
    const deviceId = formData.get("deviceId") as string
    const encrypt = formData.get("encrypt") === "true"

    if (!file) {
      return NextResponse.json({ error: "No file provided" }, { status: 400 })
    }

    const bytes = await file.arrayBuffer()
    let buffer = Buffer.from(bytes)

    // Encrypt file if requested
    if (encrypt) {
      const algorithm = "aes-256-cbc"
      const key = crypto.randomBytes(32)
      const iv = crypto.randomBytes(16)
      const cipher = crypto.createCipher(algorithm, key)

      buffer = Buffer.concat([cipher.update(buffer), cipher.final()])
    }

    // Create uploads directory if it doesn't exist
    const uploadsDir = join(process.cwd(), "uploads")
    try {
      await mkdir(uploadsDir, { recursive: true })
    } catch (error) {
      // Directory might already exist
    }

    // Generate unique filename
    const fileId = crypto.randomUUID()
    const filename = `${fileId}-${file.name}`
    const filepath = join(uploadsDir, filename)

    await writeFile(filepath, buffer)

    return NextResponse.json({
      success: true,
      fileId,
      filename: file.name,
      size: file.size,
      encrypted: encrypt,
      uploadPath: `/uploads/${filename}`,
    })
  } catch (error) {
    console.error("Upload error:", error)
    return NextResponse.json({ error: "Upload failed" }, { status: 500 })
  }
}
