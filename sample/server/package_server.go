// server.go
package main

import (
    "fmt"
    "io"
    "net"
    "os"
)

func main() {
    listener, err := net.Listen("tcp", ":8080")
    if err != nil {
        panic(err)
    }
    fmt.Println("Server is listening on port 8080...")

    for {
        conn, err := listener.Accept()
        if err != nil {
            continue
        }
        go handleConnection(conn)
    }
}

func handleConnection(conn net.Conn) {
    defer conn.Close()

    file, err := os.Open("sample.txt") // Replace with your file
    if err != nil {
        fmt.Println("Error opening file:", err)
        return
    }
    defer file.Close()

    io.Copy(conn, file)
    fmt.Println("File sent successfully!")
}
