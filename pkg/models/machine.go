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

	// Timestamps
	EnrolledAt time.Time  `json:"enrolled_at" db:"enrolled_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty" db:"last_seen_at"`
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
