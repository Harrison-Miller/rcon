package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Harrison-Miller/rcon"
)

func main() {
	client, err := rcon.DialRcon("127.0.0.1:50301", "admin", 2*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	fmt.Printf("Connected to rcon server\n")
	client.Message("Gold Party Enabled!")
	for {
		time.Sleep(10 * time.Second)
		_, err = client.Write("server_DropCoins(Vec2f(0, -50) + getPlayer(XORRandom(getPlayerCount())).getBlob().getPosition(), 10)")
		if err != nil {
			log.Fatal(err)
		}
	}
}
