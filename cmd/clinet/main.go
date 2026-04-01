// cmd/clinet/main.go
package main

import (
	"log"
	"os"
	"strconv"
	"strings"

	"clinet/internal/api"
	"clinet/internal/db"
	"clinet/internal/pubsub"
	"clinet/internal/ssh"
)

func main() {
	dbPath := getenv("CLINET_DB_PATH", "clinet.db")
	port := getenvInt("CLINET_PORT", 2222)
	apiPort := getenvInt("CLINET_API_PORT", 8080)
	hostKeyPath := getenv("CLINET_HOST_KEY_PATH", ".ssh/term_info_ed25519")

	database, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	broker := pubsub.NewBroker()
	api.Start(database, api.Config{Port: apiPort})
	ssh.Start(database, broker, ssh.Config{
		Port:        port,
		HostKeyPath: hostKeyPath,
	})
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
