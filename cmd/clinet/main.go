// cmd/clinet/main.go
package main

import (
	"log"

	"clinet/internal/db"
	"clinet/internal/pubsub"
	"clinet/internal/ssh"
)

func main() {
	// Initializes Sqlite DB, migrations, and active WAL mode.
	database, err := db.InitDB("clinet.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize the Real-Time PubSub Event Broker map memory cache.
	broker := pubsub.NewBroker()

	// Expose via Wish on port 2222 
	ssh.Start(database, broker, 2222)
}
