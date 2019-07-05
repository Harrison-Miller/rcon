package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Harrison-Miller/rcon"
)

func read(client rcon.RconClient) {
	for {
		message, _ := client.Read()
		if rcon.RemoveTimestamp(message) != "" {
			fmt.Println(rcon.RemoveTimestamp(message))
		}
	}
}

func main() {
	client, err := rcon.DialRcon("127.0.0.1:50301", "admin", 2*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	fmt.Printf("Connected to rcon server\n")
	go read(client)
	for {
		reader := bufio.NewReader(os.Stdin)
		command, _ := reader.ReadString('\n')
		if command != "\n" {
			_, err = client.Write(command)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
