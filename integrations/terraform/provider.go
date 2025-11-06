package main

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Provider returns the Metal Enrollment terraform provider
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_url": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("METAL_ENROLLMENT_URL", ""),
				Description: "URL of the Metal Enrollment API",
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("METAL_ENROLLMENT_TOKEN", ""),
				Description: "JWT authentication token",
			},
			"insecure": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Skip TLS certificate verification",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"metal-enrollment_machine":           resourceMachine(),
			"metal-enrollment_group":             resourceGroup(),
			"metal-enrollment_group_membership":  resourceGroupMembership(),
			"metal-enrollment_power_operation":   resourcePowerOperation(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"metal-enrollment_machine":  dataSourceMachine(),
			"metal-enrollment_machines": dataSourceMachines(),
			"metal-enrollment_group":    dataSourceGroup(),
			"metal-enrollment_groups":   dataSourceGroups(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

type apiClient struct {
	BaseURL   string
	Token     string
	Insecure  bool
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	apiURL := d.Get("api_url").(string)
	token := d.Get("token").(string)
	insecure := d.Get("insecure").(bool)

	var diags diag.Diagnostics

	if apiURL == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to create Metal Enrollment API client",
			Detail:   "api_url is required",
		})
		return nil, diags
	}

	client := &apiClient{
		BaseURL:  apiURL,
		Token:    token,
		Insecure: insecure,
	}

	return client, diags
}

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: Provider,
	})
}
