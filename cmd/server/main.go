package main

import (
	"log"

	"expensio/internal/config"
	"expensio/internal/db"
	"expensio/internal/routes"
)

func main() {
	log.Println("🚀 Starting Expensio API Server...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	// Connect to database
	if err := db.Connect(cfg); err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := db.Migrate(); err != nil {
		log.Fatalf("❌ Failed to run migrations: %v", err)
	}

	// Setup router
	router := routes.SetupRouter(cfg)

	// Start server
	addr := ":" + cfg.Port
	log.Printf("✅ Server is running on http://localhost%s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}
