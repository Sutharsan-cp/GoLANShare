

import socket
import threading

clients = []

def handle_client(client_socket, addr):
    print(f"[+] New connection from {addr}")
    clients.append(client_socket)
    
    while True:
        try:
            message = client_socket.recv(1024).decode()
            if not message:
                break
            
            print(f"Message from {addr}: {message}")
            broadcast(message, client_socket)
        except:
            break
    
    print(f"[-] {addr} disconnected")
    clients.remove(client_socket)
    client_socket.close()

def broadcast(message, sender_socket):
    for client in clients:
        if client != sender_socket:
            try:
                client.send(message.encode())
            except:
                clients.remove(client)
                client.close()

def start_server():
    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.bind(("0.0.0.0", 3000))
    server.listen(5)
    print("[*] Server listening on port 3000")

    while True:
        client_socket, addr = server.accept()
        thread = threading.Thread(target=handle_client, args=(client_socket, addr))
        thread.start()

if __name__ == "__main__":
    start_server()
