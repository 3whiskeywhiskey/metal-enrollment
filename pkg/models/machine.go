package models

import (
	"encoding/json"
	"time"
)

// MachineStatus represents the current state of a machine
type MachineStatus string

const (
	StatusUnknown     MachineStatus = "unknown"
	StatusEnrolled    MachineStatus = "enrolled"
	StatusConfigured  MachineStatus = "configured"
	StatusBuilding    MachineStatus = "building"
	StatusReady       MachineStatus = "ready"
	StatusProvisioned MachineStatus = "provisioned"
	StatusFailed      MachineStatus = "failed"
)

// Machine represents a bare metal machine in the system
type Machine struct {
	ID          string        `json:"id" db:"id"`
	ServiceTag  string        `json:"service_tag" db:"service_tag"`
	MACAddress  string        `json:"mac_address" db:"mac_address"`
	Status      MachineStatus `json:"status" db:"status"`
	Hostname    string        `json:"hostname" db:"hostname"`
	Description string        `json:"description" db:"description"`

	// Hardware information
	Hardware HardwareInfo `json:"hardware" db:"hardware"`

	// NixOS configuration
	NixOSConfig string `json:"nixos_config" db:"nixos_config"`

	// Build information
	LastBuildID   *string    `json:"last_build_id,omitempty" db:"last_build_id"`
	LastBuildTime *time.Time `json:"last_build_time,omitempty" db:"last_build_time"`

	// IPMI/BMC configuration
	BMCInfo *BMCInfo `json:"bmc_info,omitempty" db:"bmc_info"`

	// Timestamps
	EnrolledAt time.Time  `json:"enrolled_at" db:"enrolled_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty" db:"last_seen_at"`
}

// BMCInfo contains BMC/IPMI configuration and credentials
type BMCInfo struct {
	IPAddress string `json:"ip_address"`
	Username  string `json:"username"`
	Password  string `json:"password,omitempty"` // Encrypted in storage
	Type      string `json:"type"`               // IPMI, Redfish, etc.
	Port      int    `json:"port,omitempty"`
	Enabled   bool   `json:"enabled"`
}

// Scan implements the sql.Scanner interface for BMCInfo
func (b *BMCInfo) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, b)
}

// Value implements the driver.Valuer interface for BMCInfo
func (b BMCInfo) Value() (interface{}, error) {
	return json.Marshal(b)
}

// HardwareInfo contains detailed hardware information about a machine
type HardwareInfo struct {
	Manufacturer string          `json:"manufacturer"`
	Model        string          `json:"model"`
	SerialNumber string          `json:"serial_number"`
	BIOSVersion  string          `json:"bios_version"`

	CPU     CPUInfo     `json:"cpu"`
	Memory  MemoryInfo  `json:"memory"`
	Disks   []DiskInfo  `json:"disks"`
	NICs    []NICInfo   `json:"nics"`
	GPUs    []GPUInfo   `json:"gpus,omitempty"`

	// Raw data from dmidecode, lshw, etc.
	RawData map[string]interface{} `json:"raw_data,omitempty"`
}

// CPUInfo contains CPU details
type CPUInfo struct {
	Model       string `json:"model"`
	Cores       int    `json:"cores"`
	Threads     int    `json:"threads"`
	Sockets     int    `json:"sockets"`
	MaxFreqMHz  int    `json:"max_freq_mhz"`
	Architecture string `json:"architecture"`
}

// MemoryInfo contains memory details
type MemoryInfo struct {
	TotalBytes int64        `json:"total_bytes"`
	TotalGB    float64      `json:"total_gb"`
	Modules    []MemorySlot `json:"modules"`
}

// MemorySlot represents a single memory module
type MemorySlot struct {
	Slot      string `json:"slot"`
	SizeBytes int64  `json:"size_bytes"`
	Type      string `json:"type"` // DDR4, DDR5, etc.
	Speed     int    `json:"speed"` // MHz
}

// DiskInfo contains disk details
type DiskInfo struct {
	Device     string  `json:"device"`
	Model      string  `json:"model"`
	SizeBytes  int64   `json:"size_bytes"`
	SizeGB     float64 `json:"size_gb"`
	Type       string  `json:"type"` // SSD, HDD, NVMe
	Serial     string  `json:"serial"`
	WWN        string  `json:"wwn,omitempty"`
	Rotational bool    `json:"rotational"`
}

// NICInfo contains network interface details
type NICInfo struct {
	Name       string `json:"name"`
	MACAddress string `json:"mac_address"`
	Driver     string `json:"driver"`
	Speed      string `json:"speed"` // 1Gbps, 10Gbps, etc.
	PCIAddress string `json:"pci_address"`
	LinkStatus string `json:"link_status"` // up, down
}

// GPUInfo contains GPU details
type GPUInfo struct {
	Model      string `json:"model"`
	Vendor     string `json:"vendor"`
	PCIAddress string `json:"pci_address"`
	Memory     int64  `json:"memory_bytes,omitempty"`
}

// Scan implements the sql.Scanner interface for HardwareInfo
func (h *HardwareInfo) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, h)
}

// Value implements the driver.Valuer interface for HardwareInfo
func (h HardwareInfo) Value() (interface{}, error) {
	return json.Marshal(h)
}

// EnrollmentRequest is the payload sent by the registration image
type EnrollmentRequest struct {
	ServiceTag  string       `json:"service_tag"`
	MACAddress  string       `json:"mac_address"`
	Hardware    HardwareInfo `json:"hardware"`
}

// BuildRequest represents a request to build a custom NixOS image
type BuildRequest struct {
	ID          string    `json:"id" db:"id"`
	MachineID   string    `json:"machine_id" db:"machine_id"`
	Status      string    `json:"status" db:"status"` // pending, building, success, failed
	Config      string    `json:"config" db:"config"`
	LogOutput   string    `json:"log_output" db:"log_output"`
	Error       string    `json:"error,omitempty" db:"error"`
	ArtifactURL string    `json:"artifact_url,omitempty" db:"artifact_url"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
}

// PowerOperation represents a power control operation
type PowerOperation struct {
	ID         string    `json:"id" db:"id"`
	MachineID  string    `json:"machine_id" db:"machine_id"`
	Operation  string    `json:"operation" db:"operation"` // on, off, reset, status
	Status     string    `json:"status" db:"status"`       // pending, success, failed
	Result     string    `json:"result,omitempty" db:"result"`
	Error      string    `json:"error,omitempty" db:"error"`
	InitiatedBy string   `json:"initiated_by" db:"initiated_by"` // User ID
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
}

// MachineMetrics represents collected metrics from a machine
type MachineMetrics struct {
	ID              string    `json:"id" db:"id"`
	MachineID       string    `json:"machine_id" db:"machine_id"`
	Timestamp       time.Time `json:"timestamp" db:"timestamp"`
	CPUUsagePercent float64   `json:"cpu_usage_percent" db:"cpu_usage_percent"`
	MemoryUsedBytes int64     `json:"memory_used_bytes" db:"memory_used_bytes"`
	MemoryTotalBytes int64    `json:"memory_total_bytes" db:"memory_total_bytes"`
	DiskUsedBytes   int64     `json:"disk_used_bytes" db:"disk_used_bytes"`
	DiskTotalBytes  int64     `json:"disk_total_bytes" db:"disk_total_bytes"`
	NetworkRxBytes  int64     `json:"network_rx_bytes" db:"network_rx_bytes"`
	NetworkTxBytes  int64     `json:"network_tx_bytes" db:"network_tx_bytes"`
	LoadAverage1    float64   `json:"load_average_1" db:"load_average_1"`
	LoadAverage5    float64   `json:"load_average_5" db:"load_average_5"`
	LoadAverage15   float64   `json:"load_average_15" db:"load_average_15"`
	Temperature     *float64  `json:"temperature,omitempty" db:"temperature"`
	PowerState      string    `json:"power_state" db:"power_state"` // on, off, unknown
	Uptime          int64     `json:"uptime" db:"uptime"` // seconds
}

// ImageTest represents a test result for a boot image
type ImageTest struct {
	ID          string    `json:"id" db:"id"`
	ImagePath   string    `json:"image_path" db:"image_path"`
	ImageType   string    `json:"image_type" db:"image_type"` // registration, custom
	TestType    string    `json:"test_type" db:"test_type"`   // boot, integrity, validation
	Status      string    `json:"status" db:"status"`         // pending, running, passed, failed
	Result      string    `json:"result,omitempty" db:"result"`
	Error       string    `json:"error,omitempty" db:"error"`
	MachineID   *string   `json:"machine_id,omitempty" db:"machine_id"` // Optional: machine used for testing
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
}

// Webhook represents a webhook endpoint for event notifications
type Webhook struct {
	ID          string          `json:"id" db:"id"`
	Name        string          `json:"name" db:"name"`
	URL         string          `json:"url" db:"url"`
	Events      []string        `json:"events" db:"events"` // machine.enrolled, machine.status_changed, etc.
	Secret      string          `json:"secret,omitempty" db:"secret"` // For HMAC signature
	Active      bool            `json:"active" db:"active"`
	Headers     json.RawMessage `json:"headers,omitempty" db:"headers"` // Custom headers as JSON
	Timeout     int             `json:"timeout" db:"timeout"` // Request timeout in seconds
	MaxRetries  int             `json:"max_retries" db:"max_retries"`
	LastSuccess *time.Time      `json:"last_success,omitempty" db:"last_success"`
	LastFailure *time.Time      `json:"last_failure,omitempty" db:"last_failure"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

// WebhookDelivery represents a webhook delivery attempt
type WebhookDelivery struct {
	ID          string    `json:"id" db:"id"`
	WebhookID   string    `json:"webhook_id" db:"webhook_id"`
	Event       string    `json:"event" db:"event"`
	Payload     string    `json:"payload" db:"payload"`
	StatusCode  int       `json:"status_code" db:"status_code"`
	Response    string    `json:"response,omitempty" db:"response"`
	Error       string    `json:"error,omitempty" db:"error"`
	Attempts    int       `json:"attempts" db:"attempts"`
	Success     bool      `json:"success" db:"success"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
}

// MachineTemplate represents a configuration template for machines
type MachineTemplate struct {
	ID          string          `json:"id" db:"id"`
	Name        string          `json:"name" db:"name"`
	Description string          `json:"description" db:"description"`
	NixOSConfig string          `json:"nixos_config" db:"nixos_config"`
	BMCConfig   *BMCInfo        `json:"bmc_config,omitempty" db:"bmc_config"`
	Tags        json.RawMessage `json:"tags,omitempty" db:"tags"` // Array of tags as JSON
	Variables   json.RawMessage `json:"variables,omitempty" db:"variables"` // Template variables as JSON
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
	CreatedBy   string          `json:"created_by" db:"created_by"` // User ID
}

// MachineEvent represents an event that occurred for a machine
type MachineEvent struct {
	ID          string          `json:"id" db:"id"`
	MachineID   string          `json:"machine_id" db:"machine_id"`
	Event       string          `json:"event" db:"event"` // enrolled, status_changed, build_started, etc.
	Data        json.RawMessage `json:"data" db:"data"` // Event-specific data
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	CreatedBy   *string         `json:"created_by,omitempty" db:"created_by"` // User ID if applicable
}
