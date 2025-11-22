package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"webring/internal/config"
	"webring/internal/database"
	"webring/internal/handlers"
	"webring/internal/services"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found, using system environment variables")
	}

	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	fmt.Println("Connected to PostgreSQL database")

	if err := database.InitSchema(db); err != nil {
		log.Fatal("Failed to initialize schema:", err)
	}

	fmt.Println("Database schema initialized")

	if cfg.FrontendRepo != "" {
		fmt.Printf("Pulling frontend from %s...\n", cfg.FrontendRepo)
		if err := services.PullFrontend(cfg.FrontendRepo); err != nil {
			fmt.Printf("Warning: Failed to pull frontend: %v\n", err)
			fmt.Println("Falling back to default frontend")
		} else {
			fmt.Println("Frontend pulled successfully")
		}
	}

	fmt.Println("Starting initial site checks...")
	services.RecheckAllSites(db)
	fmt.Println("Initial checks completed")

	go services.StartPeriodicChecker(db, cfg.CheckCooldown)

	handlers.SetupRoutes(db, cfg)

	fmt.Printf("Webring server starting on :%s\n", cfg.Port)
	fmt.Printf("Site Title: %s\n", cfg.SiteTitle)
	if cfg.SiteFavicon != "" {
		fmt.Printf("Favicon: %s\n", cfg.SiteFavicon)
	}
	fmt.Printf("Webring URL: %s\n", cfg.WebringURL)
	fmt.Printf("API Docs: %s/api/docs\n", cfg.WebringURL)

	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatal("Server error:", err)
	}
}
