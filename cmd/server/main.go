package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/api"
	"github.com/3whiskeywhiskey/metal-enrollment/pkg/auth"
	"github.com/3whiskeywhiskey/metal-enrollment/pkg/database"
	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/3whiskeywhiskey/metal-enrollment/pkg/web"
	"github.com/gorilla/mux"
)

func main() {
	// Parse flags
	dbDriver := flag.String("db-driver", getEnv("DB_DRIVER", "sqlite3"), "Database driver (sqlite3 or postgres)")
	dbDSN := flag.String("db-dsn", getEnv("DB_DSN", "metal-enrollment.db"), "Database connection string")
	listenAddr := flag.String("listen", getEnv("LISTEN_ADDR", ":8080"), "HTTP listen address")
	builderURL := flag.String("builder-url", getEnv("BUILDER_URL", "http://builder:8081"), "Image builder service URL")
	enableAuth := flag.Bool("enable-auth", getEnv("ENABLE_AUTH", "true") == "true", "Enable authentication")
	jwtSecret := flag.String("jwt-secret", getEnv("JWT_SECRET", "change-me-in-production"), "JWT signing secret")
	createAdmin := flag.Bool("create-admin", false, "Create default admin user")
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

	// Create default admin user if requested
	if *createAdmin {
		if err := createDefaultAdmin(db); err != nil {
			log.Fatalf("Failed to create admin user: %v", err)
		}
	}

	// Create API server
	apiServer := api.New(db, api.Config{
		ListenAddr: *listenAddr,
		BuilderURL: *builderURL,
		JWTSecret:  *jwtSecret,
		JWTExpiry:  24 * time.Hour,
		EnableAuth: *enableAuth,
	})

	// Create web server
	webServer := web.NewServer(db)

	// Combine routers
	router := mux.NewRouter()
	router.PathPrefix("/api/").Handler(apiServer.Router)
	router.PathPrefix("/").Handler(webServer.Router())

	// Start server
	log.Printf("Starting Metal Enrollment server on %s (auth: %v)", *listenAddr, *enableAuth)
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

func createDefaultAdmin(db *database.DB) error {
	// Check if admin already exists
	admin, err := db.GetUserByUsername("admin")
	if err != nil {
		return err
	}

	if admin != nil {
		log.Println("Admin user already exists")
		return nil
	}

	// Create admin user
	passwordHash, err := auth.HashPassword("admin")
	if err != nil {
		return err
	}

	admin, err = db.CreateUser("admin", "admin@localhost", passwordHash, models.RoleAdmin)
	if err != nil {
		return err
	}

	log.Printf("Created default admin user (username: admin, password: admin)")
	log.Printf("IMPORTANT: Change the default password immediately!")
	return nil
}
