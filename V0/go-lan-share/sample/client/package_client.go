// client.go
package main

import (
    "fmt"
    "io"
    "net"
    "os"
)

func main() {
    conn, err := net.Dial("tcp", "10.1.66.251:8080") // Replace with server IP
    if err != nil {
        panic(err)
    }
    defer conn.Close()

    outFile, err := os.Create("received.txt")
    if err != nil {
        panic(err)
    }
    defer outFile.Close()

    io.Copy(outFile, conn)
    fmt.Println("File received and saved as received.txt")
}
