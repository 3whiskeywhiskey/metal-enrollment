package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/api"
	"github.com/3whiskeywhiskey/metal-enrollment/pkg/database"
	"github.com/3whiskeywhiskey/metal-enrollment/pkg/web"
	"github.com/gorilla/mux"
)

func main() {
	// Parse flags
	dbDriver := flag.String("db-driver", getEnv("DB_DRIVER", "sqlite3"), "Database driver (sqlite3 or postgres)")
	dbDSN := flag.String("db-dsn", getEnv("DB_DSN", "metal-enrollment.db"), "Database connection string")
	listenAddr := flag.String("listen", getEnv("LISTEN_ADDR", ":8080"), "HTTP listen address")
	builderURL := flag.String("builder-url", getEnv("BUILDER_URL", "http://builder:8081"), "Image builder service URL")
	flag.Parse()

	// Initialize database
	db, err := database.New(database.Config{
		Driver: *dbDriver,
		DSN:    *dbDSN,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Printf("Database initialized successfully (%s)", *dbDriver)

	// Create API server
	apiServer := api.New(db, api.Config{
		ListenAddr: *listenAddr,
		BuilderURL: *builderURL,
	})

	// Create web server
	webServer := web.NewServer(db)

	// Combine routers
	router := mux.NewRouter()
	router.PathPrefix("/api/").Handler(apiServer.router)
	router.PathPrefix("/").Handler(webServer.Router())

	// Start server
	log.Printf("Starting Metal Enrollment server on %s", *listenAddr)
	if err := http.ListenAndServe(*listenAddr, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
