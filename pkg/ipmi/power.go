package ipmi

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
)

// PowerOperation represents supported power operations
type PowerOperation string

const (
	PowerOn     PowerOperation = "on"
	PowerOff    PowerOperation = "off"
	PowerCycle  PowerOperation = "cycle"
	PowerReset  PowerOperation = "reset"
	PowerStatus PowerOperation = "status"
)

// PowerController handles IPMI power operations
type PowerController struct {
	timeout time.Duration
}

// NewPowerController creates a new IPMI power controller
func NewPowerController() *PowerController {
	return &PowerController{
		timeout: 30 * time.Second,
	}
}

// ExecutePowerOperation executes a power operation on a machine
func (pc *PowerController) ExecutePowerOperation(bmc *models.BMCInfo, operation PowerOperation) (string, error) {
	if bmc == nil {
		return "", fmt.Errorf("BMC info is required")
	}

	if !bmc.Enabled {
		return "", fmt.Errorf("BMC is not enabled for this machine")
	}

	if bmc.IPAddress == "" {
		return "", fmt.Errorf("BMC IP address is required")
	}

	// Build ipmitool command
	args := []string{
		"-I", "lanplus",
		"-H", bmc.IPAddress,
		"-U", bmc.Username,
	}

	// Add password if provided
	if bmc.Password != "" {
		args = append(args, "-P", bmc.Password)
	}

	// Add port if specified
	if bmc.Port > 0 {
		args = append(args, "-p", fmt.Sprintf("%d", bmc.Port))
	}

	// Add the power command
	args = append(args, "power", string(operation))

	// Execute the command
	cmd := exec.Command("ipmitool", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("ipmitool error: %w, stderr: %s", err, stderr.String())
		}
		return strings.TrimSpace(stdout.String()), nil
	case <-time.After(pc.timeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return "", fmt.Errorf("ipmitool command timed out after %s", pc.timeout)
	}
}

// GetPowerStatus gets the current power status of a machine
func (pc *PowerController) GetPowerStatus(bmc *models.BMCInfo) (string, error) {
	result, err := pc.ExecutePowerOperation(bmc, PowerStatus)
	if err != nil {
		return "unknown", err
	}

	// Parse the result
	// ipmitool returns "Chassis Power is on" or "Chassis Power is off"
	result = strings.ToLower(result)
	if strings.Contains(result, "on") {
		return "on", nil
	} else if strings.Contains(result, "off") {
		return "off", nil
	}

	return "unknown", nil
}

// PowerOn turns on a machine
func (pc *PowerController) PowerOn(bmc *models.BMCInfo) (string, error) {
	return pc.ExecutePowerOperation(bmc, PowerOn)
}

// PowerOff turns off a machine (graceful if supported)
func (pc *PowerController) PowerOff(bmc *models.BMCInfo) (string, error) {
	return pc.ExecutePowerOperation(bmc, PowerOff)
}

// PowerReset performs a hard reset of a machine
func (pc *PowerController) PowerReset(bmc *models.BMCInfo) (string, error) {
	return pc.ExecutePowerOperation(bmc, PowerReset)
}

// PowerCycle performs a power cycle (off then on)
func (pc *PowerController) PowerCycle(bmc *models.BMCInfo) (string, error) {
	return pc.ExecutePowerOperation(bmc, PowerCycle)
}

// TestConnection tests the connection to the BMC
func (pc *PowerController) TestConnection(bmc *models.BMCInfo) error {
	_, err := pc.GetPowerStatus(bmc)
	return err
}

// GetBMCInfo retrieves BMC information
func (pc *PowerController) GetBMCInfo(bmc *models.BMCInfo) (map[string]string, error) {
	if bmc == nil {
		return nil, fmt.Errorf("BMC info is required")
	}

	args := []string{
		"-I", "lanplus",
		"-H", bmc.IPAddress,
		"-U", bmc.Username,
	}

	if bmc.Password != "" {
		args = append(args, "-P", bmc.Password)
	}

	if bmc.Port > 0 {
		args = append(args, "-p", fmt.Sprintf("%d", bmc.Port))
	}

	args = append(args, "mc", "info")

	cmd := exec.Command("ipmitool", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("ipmitool error: %w, stderr: %s", err, stderr.String())
		}

		// Parse the output
		info := make(map[string]string)
		lines := strings.Split(stdout.String(), "\n")
		for _, line := range lines {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				info[key] = value
			}
		}

		return info, nil
	case <-time.After(pc.timeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return nil, fmt.Errorf("ipmitool command timed out after %s", pc.timeout)
	}
}

// GetSensorReadings retrieves sensor readings from the BMC
func (pc *PowerController) GetSensorReadings(bmc *models.BMCInfo) ([]SensorReading, error) {
	if bmc == nil {
		return nil, fmt.Errorf("BMC info is required")
	}

	args := []string{
		"-I", "lanplus",
		"-H", bmc.IPAddress,
		"-U", bmc.Username,
	}

	if bmc.Password != "" {
		args = append(args, "-P", bmc.Password)
	}

	if bmc.Port > 0 {
		args = append(args, "-p", fmt.Sprintf("%d", bmc.Port))
	}

	args = append(args, "sdr", "list")

	cmd := exec.Command("ipmitool", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("ipmitool error: %w, stderr: %s", err, stderr.String())
		}

		// Parse sensor readings
		var readings []SensorReading
		lines := strings.Split(stdout.String(), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}

			parts := strings.Split(line, "|")
			if len(parts) >= 3 {
				reading := SensorReading{
					Name:   strings.TrimSpace(parts[0]),
					Value:  strings.TrimSpace(parts[1]),
					Status: strings.TrimSpace(parts[2]),
				}
				readings = append(readings, reading)
			}
		}

		return readings, nil
	case <-time.After(pc.timeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return nil, fmt.Errorf("ipmitool command timed out after %s", pc.timeout)
	}
}

// SensorReading represents a sensor reading from IPMI
type SensorReading struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Status string `json:"status"`
}
