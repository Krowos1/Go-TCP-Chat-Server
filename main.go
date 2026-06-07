package main

import (
	"bufio"
	"log"
	"net"
	"strings"
	"sync"
)

var clients = make(map[net.Conn]string)
var mu sync.Mutex

func main() {
	address := "localhost:8080"
	listener, err := net.Listen("tcp", address)

	if err != nil {
		log.Fatal(err)
	}

	defer listener.Close()
	log.Println("Server listening on", address)

	for {
		connc, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handleClient(connc)
	}

}

func broadcast(message string) {
	mu.Lock()
	defer mu.Unlock()

	for client := range clients {
		_, err := client.Write([]byte(message))
		if err != nil {
			log.Println("Can't send message", err)
		}
	}

}

func handleClient(connc net.Conn) {

	defer connc.Close()
	log.Println("Client connected")

	reader := bufio.NewReader(connc)
	_, err := connc.Write([]byte("Enter your name: "))
	if err != nil {
		log.Println("Can't ask name", err)
		return
	}

	name, err := reader.ReadString('\n')
	if err != nil {
		log.Println("Can't read name", err)
		return
	}
	clientName := strings.TrimSpace(name)
	if clientName == "" {
		clientName = connc.RemoteAddr().String()
	}

	mu.Lock()
	clients[connc] = clientName
	mu.Unlock()

	defer func() {
		mu.Lock()
		delete(clients, connc)
		mu.Unlock()
		broadcast(clientName + " left the chat\n")
		log.Println("Client disconnected")
	}()

	broadcast(clientName + " joined the chat\n")

	for {
		message, err := reader.ReadString('\n')

		if err != nil {
			log.Println("Can't read", err)
			return
		}
		log.Println("Received:", message)

		cleanMessage := strings.TrimSpace(message)

		if cleanMessage == "/quit" {
			connc.Write([]byte("Bye\n"))
			return
		}

		if cleanMessage == "/users" {
			sendUsers(connc)
			continue
		}

		if cleanMessage == "/help" {
			sendHelp(connc)
			continue
		}

		if strings.HasPrefix(cleanMessage, "/whisper ") {
			parts := strings.SplitN(cleanMessage, " ", 3)

			if len(parts) < 3 {
				connc.Write([]byte("Usage: /whisper <name> <message>\n"))
				continue
			}

			targetName := parts[1]
			privateMessage := parts[2]

			ok := sendWhisper(clientName, targetName, privateMessage)
			if !ok {
				connc.Write([]byte("User not found\n"))
			}
			continue
		}

		broadcast(clientName + ": " + message)

	}

}

func sendUsers(connc net.Conn) {
	mu.Lock()
	defer mu.Unlock()
	connc.Write([]byte("Online users:\n"))

	for _, name := range clients {
		connc.Write([]byte("- " + name + "\n"))
	}
}

func sendHelp(connc net.Conn) {
	connc.Write([]byte("Available commands: \n"))
	connc.Write([]byte("/users - show online users\n"))
	connc.Write([]byte("/quit - leave the chat\n"))
	connc.Write([]byte("/help - show commands\n"))
	connc.Write([]byte("/whisper <name> <message> - send private message\n"))
}

func sendWhisper(senderName string, targetName string, message string) bool {
	mu.Lock()
	defer mu.Unlock()

	for client, name := range clients {
		if name == targetName {
			client.Write([]byte("private from " + senderName + ": " + message + "\n"))
			return true
		}
	}

	return false
}
