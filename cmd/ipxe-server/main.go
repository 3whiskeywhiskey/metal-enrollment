package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gorilla/mux"
)

const defaultIPXEScript = `#!ipxe
# Registration image for {{.ServiceTag}}
# Unknown machine - serving registration image

echo Metal Enrollment - Registration Mode
echo Service Tag: {{.ServiceTag}}
echo ========================================

kernel {{.BaseURL}}/images/registration/bzImage init=/nix/store/HASH-nixos-system-registration/init console=ttyS0,115200 console=tty0 enrollment_url={{.EnrollmentURL}}
initrd {{.BaseURL}}/images/registration/initrd
boot
`

const machineIPXEScript = `#!ipxe
# Custom image for {{.ServiceTag}}

echo Metal Enrollment - Custom Image
echo Service Tag: {{.ServiceTag}}
echo Hostname: {{.Hostname}}
echo ========================================

kernel {{.BaseURL}}/images/machines/{{.ServiceTag}}/bzImage init=/nix/store/HASH-nixos-system-{{.Hostname}}/init console=ttyS0,115200 console=tty0
initrd {{.BaseURL}}/images/machines/{{.ServiceTag}}/initrd
boot
`

type iPXEConfig struct {
	ServiceTag    string
	Hostname      string
	BaseURL       string
	EnrollmentURL string
}

type Server struct {
	baseURL       string
	enrollmentURL string
	apiURL        string
	imagesDir     string
	templates     struct {
		registration *template.Template
		machine      *template.Template
	}
}

func main() {
	baseURL := flag.String("base-url", getEnv("BASE_URL", "http://192.168.1.100"), "Base URL for iPXE scripts")
	enrollmentURL := flag.String("enrollment-url", getEnv("ENROLLMENT_URL", "http://enrollment.local:8080/api/v1/enroll"), "Enrollment API URL")
	apiURL := flag.String("api-url", getEnv("API_URL", "http://enrollment.local:8080/api/v1"), "API base URL")
	imagesDir := flag.String("images-dir", getEnv("IMAGES_DIR", "/var/lib/metal-enrollment/images"), "Directory for serving images")
	listenAddr := flag.String("listen", getEnv("LISTEN_ADDR", ":8080"), "HTTP listen address")
	flag.Parse()

	server := &Server{
		baseURL:       *baseURL,
		enrollmentURL: *enrollmentURL,
		apiURL:        *apiURL,
		imagesDir:     *imagesDir,
	}

	// Parse templates
	var err error
	server.templates.registration, err = template.New("registration").Parse(defaultIPXEScript)
	if err != nil {
		log.Fatalf("Failed to parse registration template: %v", err)
	}

	server.templates.machine, err = template.New("machine").Parse(machineIPXEScript)
	if err != nil {
		log.Fatalf("Failed to parse machine template: %v", err)
	}

	// Ensure images directory exists
	if err := os.MkdirAll(*imagesDir, 0755); err != nil {
		log.Fatalf("Failed to create images directory: %v", err)
	}

	router := mux.NewRouter()

	// iPXE script routes
	router.HandleFunc("/nixos/machines/{servicetag}.ipxe", server.handleMachineIPXE).Methods("GET")

	// Serve kernel and initrd images
	router.PathPrefix("/images/").Handler(http.StripPrefix("/images/",
		http.FileServer(http.Dir(*imagesDir))))

	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	}).Methods("GET")

	log.Printf("Starting iPXE server on %s", *listenAddr)
	log.Printf("Base URL: %s", *baseURL)
	log.Printf("Enrollment URL: %s", *enrollmentURL)
	log.Printf("Images directory: %s", *imagesDir)

	if err := http.ListenAndServe(*listenAddr, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func (s *Server) handleMachineIPXE(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceTag := vars["servicetag"]

	log.Printf("iPXE request for service tag: %s", serviceTag)

	// Check if machine exists and has a custom image
	machineExists, hostname := s.checkMachine(serviceTag)

	w.Header().Set("Content-Type", "text/plain")

	config := iPXEConfig{
		ServiceTag:    serviceTag,
		Hostname:      hostname,
		BaseURL:       s.baseURL,
		EnrollmentURL: s.enrollmentURL,
	}

	if machineExists && hostname != "" {
		// Check if custom image exists
		imagePath := filepath.Join(s.imagesDir, "machines", serviceTag, "bzImage")
		if _, err := os.Stat(imagePath); err == nil {
			log.Printf("Serving custom image for %s (hostname: %s)", serviceTag, hostname)
			if err := s.templates.machine.Execute(w, config); err != nil {
				log.Printf("Error executing template: %v", err)
			}
			return
		}
	}

	// Serve registration image
	log.Printf("Serving registration image for %s", serviceTag)
	if err := s.templates.registration.Execute(w, config); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

func (s *Server) checkMachine(serviceTag string) (bool, string) {
	// Make API call to check if machine exists
	url := fmt.Sprintf("%s/machines/by-servicetag/%s", s.apiURL, serviceTag)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error checking machine: %v", err)
		return false, ""
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, ""
	}

	if resp.StatusCode != http.StatusOK {
		return false, ""
	}

	// Parse response to get hostname
	// For now, just return true - we'll implement full parsing later
	return true, ""
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
