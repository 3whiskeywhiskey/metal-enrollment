package api

import (
	"fmt"
	"net/http"
	"strings"
)

// handlePrometheusMetrics exports metrics in Prometheus format
func (s *Server) handlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	// Get all machines
	machines, err := s.db.ListMachines()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get machines: %v", err), http.StatusInternalServerError)
		return
	}

	var output strings.Builder

	// Write Prometheus format metrics
	output.WriteString("# HELP metal_enrollment_machines_total Total number of enrolled machines\n")
	output.WriteString("# TYPE metal_enrollment_machines_total gauge\n")
	output.WriteString(fmt.Sprintf("metal_enrollment_machines_total %d\n", len(machines)))
	output.WriteString("\n")

	// Machine status counts
	statusCounts := make(map[string]int)
	for _, machine := range machines {
		statusCounts[string(machine.Status)]++
	}

	output.WriteString("# HELP metal_enrollment_machines_by_status Number of machines by status\n")
	output.WriteString("# TYPE metal_enrollment_machines_by_status gauge\n")
	for status, count := range statusCounts {
		output.WriteString(fmt.Sprintf("metal_enrollment_machines_by_status{status=\"%s\"} %d\n", status, count))
	}
	output.WriteString("\n")

	// Metrics for each machine
	output.WriteString("# HELP metal_machine_cpu_usage_percent CPU usage percentage\n")
	output.WriteString("# TYPE metal_machine_cpu_usage_percent gauge\n")

	output.WriteString("# HELP metal_machine_memory_used_bytes Memory used in bytes\n")
	output.WriteString("# TYPE metal_machine_memory_used_bytes gauge\n")

	output.WriteString("# HELP metal_machine_memory_total_bytes Total memory in bytes\n")
	output.WriteString("# TYPE metal_machine_memory_total_bytes gauge\n")

	output.WriteString("# HELP metal_machine_disk_used_bytes Disk used in bytes\n")
	output.WriteString("# TYPE metal_machine_disk_used_bytes gauge\n")

	output.WriteString("# HELP metal_machine_disk_total_bytes Total disk space in bytes\n")
	output.WriteString("# TYPE metal_machine_disk_total_bytes gauge\n")

	output.WriteString("# HELP metal_machine_network_rx_bytes Network received bytes\n")
	output.WriteString("# TYPE metal_machine_network_rx_bytes counter\n")

	output.WriteString("# HELP metal_machine_network_tx_bytes Network transmitted bytes\n")
	output.WriteString("# TYPE metal_machine_network_tx_bytes counter\n")

	output.WriteString("# HELP metal_machine_load_average Load average\n")
	output.WriteString("# TYPE metal_machine_load_average gauge\n")

	output.WriteString("# HELP metal_machine_temperature_celsius Machine temperature in Celsius\n")
	output.WriteString("# TYPE metal_machine_temperature_celsius gauge\n")

	output.WriteString("# HELP metal_machine_uptime_seconds Machine uptime in seconds\n")
	output.WriteString("# TYPE metal_machine_uptime_seconds counter\n")

	// Get metrics for each machine
	for _, machine := range machines {
		metrics, err := s.db.GetLatestMetrics(machine.ID)
		if err != nil || metrics == nil {
			continue
		}

		labels := fmt.Sprintf("machine_id=\"%s\",hostname=\"%s\",service_tag=\"%s\"",
			machine.ID, machine.Hostname, machine.ServiceTag)

		output.WriteString(fmt.Sprintf("metal_machine_cpu_usage_percent{%s} %.2f\n", labels, metrics.CPUUsagePercent))
		output.WriteString(fmt.Sprintf("metal_machine_memory_used_bytes{%s} %d\n", labels, metrics.MemoryUsedBytes))
		output.WriteString(fmt.Sprintf("metal_machine_memory_total_bytes{%s} %d\n", labels, metrics.MemoryTotalBytes))
		output.WriteString(fmt.Sprintf("metal_machine_disk_used_bytes{%s} %d\n", labels, metrics.DiskUsedBytes))
		output.WriteString(fmt.Sprintf("metal_machine_disk_total_bytes{%s} %d\n", labels, metrics.DiskTotalBytes))
		output.WriteString(fmt.Sprintf("metal_machine_network_rx_bytes{%s} %d\n", labels, metrics.NetworkRxBytes))
		output.WriteString(fmt.Sprintf("metal_machine_network_tx_bytes{%s} %d\n", labels, metrics.NetworkTxBytes))
		output.WriteString(fmt.Sprintf("metal_machine_load_average{%s,period=\"1m\"} %.2f\n", labels, metrics.LoadAverage1))
		output.WriteString(fmt.Sprintf("metal_machine_load_average{%s,period=\"5m\"} %.2f\n", labels, metrics.LoadAverage5))
		output.WriteString(fmt.Sprintf("metal_machine_load_average{%s,period=\"15m\"} %.2f\n", labels, metrics.LoadAverage15))

		if metrics.Temperature != nil {
			output.WriteString(fmt.Sprintf("metal_machine_temperature_celsius{%s} %.2f\n", labels, *metrics.Temperature))
		}

		output.WriteString(fmt.Sprintf("metal_machine_uptime_seconds{%s} %d\n", labels, metrics.Uptime))

		// Power state as a boolean
		powerOn := 0
		if metrics.PowerState == "on" {
			powerOn = 1
		}
		output.WriteString(fmt.Sprintf("metal_machine_power_on{%s} %d\n", labels, powerOn))
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.Write([]byte(output.String()))
}
