package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/database"
	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/gorilla/mux"
)

type Builder struct {
	db          *database.DB
	buildDir    string
	outputDir   string
	nixosDir    string
}

type BuildJobRequest struct {
	BuildID   string `json:"build_id"`
	MachineID string `json:"machine_id"`
	Config    string `json:"config"`
}

func main() {
	dbDriver := flag.String("db-driver", getEnv("DB_DRIVER", "sqlite3"), "Database driver")
	dbDSN := flag.String("db-dsn", getEnv("DB_DSN", "metal-enrollment.db"), "Database connection string")
	listenAddr := flag.String("listen", getEnv("LISTEN_ADDR", ":8081"), "HTTP listen address")
	buildDir := flag.String("build-dir", getEnv("BUILD_DIR", "/tmp/metal-builds"), "Build working directory")
	outputDir := flag.String("output-dir", getEnv("OUTPUT_DIR", "/var/lib/metal-enrollment/images"), "Output directory for built images")
	nixosDir := flag.String("nixos-dir", getEnv("NIXOS_DIR", "/etc/metal-enrollment/nixos"), "NixOS configurations directory")
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

	builder := &Builder{
		db:          db,
		buildDir:    *buildDir,
		outputDir:   *outputDir,
		nixosDir:    *nixosDir,
	}

	// Ensure directories exist
	for _, dir := range []string{*buildDir, *outputDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Start build worker
	go builder.worker()

	// Start HTTP server
	router := mux.NewRouter()
	router.HandleFunc("/health", handleHealth).Methods("GET")
	router.HandleFunc("/build", builder.handleBuild).Methods("POST")

	log.Printf("Starting builder service on %s", *listenAddr)
	if err := http.ListenAndServe(*listenAddr, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func (b *Builder) handleBuild(w http.ResponseWriter, r *http.Request) {
	var req BuildJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Build will be picked up by worker
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "accepted",
		"build_id": req.BuildID,
	})
}

func (b *Builder) worker() {
	log.Println("Build worker started")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Get pending builds
		builds, err := b.getPendingBuilds()
		if err != nil {
			log.Printf("Error getting pending builds: %v", err)
			continue
		}

		for _, build := range builds {
			log.Printf("Processing build %s for machine %s", build.ID, build.MachineID)
			b.processBuild(build)
		}
	}
}

func (b *Builder) getPendingBuilds() ([]*models.BuildRequest, error) {
	// Query database for pending builds
	// This is a simplified version - in production you'd want proper querying
	query := `SELECT id, machine_id, status, config, created_at FROM builds WHERE status = 'pending' ORDER BY created_at ASC LIMIT 1`

	rows, err := b.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var builds []*models.BuildRequest
	for rows.Next() {
		build := &models.BuildRequest{}
		err := rows.Scan(&build.ID, &build.MachineID, &build.Status, &build.Config, &build.CreatedAt)
		if err != nil {
			return nil, err
		}
		builds = append(builds, build)
	}

	return builds, nil
}

func (b *Builder) processBuild(build *models.BuildRequest) {
	// Update status to building
	build.Status = "building"
	if err := b.db.UpdateBuild(build); err != nil {
		log.Printf("Failed to update build status: %v", err)
		return
	}

	// Get machine details
	machine, err := b.db.GetMachine(build.MachineID)
	if err != nil {
		log.Printf("Failed to get machine: %v", err)
		b.failBuild(build, fmt.Sprintf("Failed to get machine: %v", err))
		return
	}

	// Create build directory
	buildPath := filepath.Join(b.buildDir, build.ID)
	if err := os.MkdirAll(buildPath, 0755); err != nil {
		b.failBuild(build, fmt.Sprintf("Failed to create build directory: %v", err))
		return
	}
	defer os.RemoveAll(buildPath)

	// Write configuration file
	configPath := filepath.Join(buildPath, "configuration.nix")
	if err := os.WriteFile(configPath, []byte(build.Config), 0644); err != nil {
		b.failBuild(build, fmt.Sprintf("Failed to write config: %v", err))
		return
	}

	// Build NixOS system
	log.Printf("Building NixOS system for %s", machine.ServiceTag)
	output, err := b.buildNixOS(buildPath, machine)
	build.LogOutput = output

	if err != nil {
		b.failBuild(build, fmt.Sprintf("Build failed: %v", err))
		return
	}

	// Copy artifacts to output directory
	outputPath := filepath.Join(b.outputDir, "machines", machine.ServiceTag)
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		b.failBuild(build, fmt.Sprintf("Failed to create output directory: %v", err))
		return
	}

	// Extract kernel and initrd from result
	resultPath := filepath.Join(buildPath, "result")
	kernelSrc := filepath.Join(resultPath, "kernel")
	initrdSrc := filepath.Join(resultPath, "initrd")

	if err := copyFile(kernelSrc, filepath.Join(outputPath, "bzImage")); err != nil {
		b.failBuild(build, fmt.Sprintf("Failed to copy kernel: %v", err))
		return
	}

	if err := copyFile(initrdSrc, filepath.Join(outputPath, "initrd")); err != nil {
		b.failBuild(build, fmt.Sprintf("Failed to copy initrd: %v", err))
		return
	}

	// Mark build as success
	build.Status = "success"
	build.ArtifactURL = fmt.Sprintf("/images/machines/%s", machine.ServiceTag)
	now := time.Now()
	build.CompletedAt = &now

	if err := b.db.UpdateBuild(build); err != nil {
		log.Printf("Failed to update build: %v", err)
		return
	}

	// Update machine status
	machine.Status = models.StatusReady
	machine.LastBuildID = &build.ID
	machine.LastBuildTime = &now
	if err := b.db.UpdateMachine(machine); err != nil {
		log.Printf("Failed to update machine: %v", err)
	}

	log.Printf("Build %s completed successfully", build.ID)
}

func (b *Builder) buildNixOS(buildPath string, machine *models.Machine) (string, error) {
	// Build the netboot system
	// nix-build '<nixpkgs/nixos>' -A config.system.build.netbootRamdisk -I nixos-config=./configuration.nix

	cmd := exec.Command("nix-build",
		"<nixpkgs/nixos>",
		"-A", "config.system.build.netbootRamdisk",
		"-I", fmt.Sprintf("nixos-config=%s/configuration.nix", buildPath),
		"-o", filepath.Join(buildPath, "result"),
	)

	cmd.Dir = buildPath
	output, err := cmd.CombinedOutput()

	return string(output), err
}

func (b *Builder) failBuild(build *models.BuildRequest, errorMsg string) {
	log.Printf("Build %s failed: %s", build.ID, errorMsg)

	build.Status = "failed"
	build.Error = errorMsg
	now := time.Now()
	build.CompletedAt = &now

	if err := b.db.UpdateBuild(build); err != nil {
		log.Printf("Failed to update build status: %v", err)
	}

	// Update machine status
	machine, err := b.db.GetMachine(build.MachineID)
	if err == nil {
		machine.Status = models.StatusFailed
		b.db.UpdateMachine(machine)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
