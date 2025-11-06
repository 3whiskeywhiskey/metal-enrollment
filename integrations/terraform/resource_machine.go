package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceMachine() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMachineUpdate,  // Machines are auto-enrolled, we only update
		ReadContext:   resourceMachineRead,
		UpdateContext: resourceMachineUpdate,
		DeleteContext: resourceMachineDelete,

		Schema: map[string]*schema.Schema{
			"service_tag": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Machine service tag (unique identifier)",
			},
			"hostname": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Machine hostname",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Machine description",
			},
			"nixos_config": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "NixOS configuration",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Machine status",
			},
			"mac_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Machine MAC address",
			},
			"enrolled_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Enrollment timestamp",
			},
			"bmc": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "BMC/IPMI configuration",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ip_address": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "BMC IP address",
						},
						"username": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "BMC username",
						},
						"password": {
							Type:        schema.TypeString,
							Required:    true,
							Sensitive:   true,
							Description: "BMC password",
						},
						"type": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "IPMI",
							Description: "BMC type (IPMI, Redfish, etc.)",
						},
						"port": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     623,
							Description: "BMC port",
						},
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable BMC access",
						},
					},
				},
			},
		},
	}
}

func resourceMachineRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*apiClient)
	var diags diag.Diagnostics

	machineID := d.Id()
	url := fmt.Sprintf("%s/api/v1/machines/%s", client.BaseURL, machineID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	if client.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.Token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return diag.FromErr(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		d.SetId("")
		return diags
	}

	if resp.StatusCode != 200 {
		return diag.Errorf("API returned status %d", resp.StatusCode)
	}

	var machine map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&machine); err != nil {
		return diag.FromErr(err)
	}

	d.Set("service_tag", machine["service_tag"])
	d.Set("hostname", machine["hostname"])
	d.Set("description", machine["description"])
	d.Set("nixos_config", machine["nixos_config"])
	d.Set("status", machine["status"])
	d.Set("mac_address", machine["mac_address"])
	d.Set("enrolled_at", machine["enrolled_at"])

	// Set BMC info if present
	if bmcInfo, ok := machine["bmc_info"].(map[string]interface{}); ok && bmcInfo != nil {
		bmcList := []map[string]interface{}{
			{
				"ip_address": bmcInfo["ip_address"],
				"username":   bmcInfo["username"],
				"type":       bmcInfo["type"],
				"port":       bmcInfo["port"],
				"enabled":    bmcInfo["enabled"],
				// Password is not returned from API for security
			},
		}
		d.Set("bmc", bmcList)
	}

	return diags
}

func resourceMachineUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*apiClient)

	// First, try to find the machine by service tag
	serviceTag := d.Get("service_tag").(string)
	url := fmt.Sprintf("%s/api/v1/machines", client.BaseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	if client.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.Token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return diag.FromErr(err)
	}
	defer resp.Body.Close()

	var machines []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&machines); err != nil {
		return diag.FromErr(err)
	}

	var machineID string
	for _, m := range machines {
		if m["service_tag"] == serviceTag {
			machineID = m["id"].(string)
			break
		}
	}

	if machineID == "" {
		return diag.Errorf("Machine with service tag %s not found. Ensure it has been enrolled first.", serviceTag)
	}

	d.SetId(machineID)

	// Build update payload
	update := map[string]interface{}{
		"hostname":     d.Get("hostname"),
		"description":  d.Get("description"),
		"nixos_config": d.Get("nixos_config"),
	}

	// Add BMC info if configured
	if bmcList, ok := d.GetOk("bmc"); ok && len(bmcList.([]interface{})) > 0 {
		bmcData := bmcList.([]interface{})[0].(map[string]interface{})
		update["bmc_info"] = map[string]interface{}{
			"ip_address": bmcData["ip_address"],
			"username":   bmcData["username"],
			"password":   bmcData["password"],
			"type":       bmcData["type"],
			"port":       bmcData["port"],
			"enabled":    bmcData["enabled"],
		}
	}

	body, err := json.Marshal(update)
	if err != nil {
		return diag.FromErr(err)
	}

	updateURL := fmt.Sprintf("%s/api/v1/machines/%s", client.BaseURL, machineID)
	req, err = http.NewRequestWithContext(ctx, "PUT", updateURL, bytes.NewReader(body))
	if err != nil {
		return diag.FromErr(err)
	}

	req.Header.Set("Content-Type", "application/json")
	if client.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.Token))
	}

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return diag.FromErr(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return diag.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return resourceMachineRead(ctx, d, meta)
}

func resourceMachineDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*apiClient)
	var diags diag.Diagnostics

	machineID := d.Id()
	url := fmt.Sprintf("%s/api/v1/machines/%s", client.BaseURL, machineID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	if client.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.Token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return diag.FromErr(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return diag.Errorf("API returned status %d", resp.StatusCode)
	}

	d.SetId("")
	return diags
}
