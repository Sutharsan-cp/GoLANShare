import socket
import threading

def receive_messages(client_socket):
    while True:
        try:
            message = client_socket.recv(1024).decode()
            print(message)
        except:
            print("Disconnected from server")
            client_socket.close()
            break

def start_client():
    server_ip = "10.1.66.250"  # Replace with server's actual IP
    port = 12345

    client = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    client.connect((server_ip, port))

    receive_thread = threading.Thread(target=receive_messages, args=(client,))
    receive_thread.start()

    while True:
        message = input("")
        if message.lower() == 'exit':
            client.close()
            break
        client.send(message.encode())

if __name__ == "__main__":
    print("Connected to server. Type 'exit' to quit.")
    start_client()
