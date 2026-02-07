package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/require"

	"terraform-provider-phare/internal/client"
)

func TestUptimeMonitorHttpResource_Metadata(t *testing.T) {
	r := NewUptimeMonitorHttpResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "phare",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	require.Equal(t, "phare_uptime_monitor_http", resp.TypeName)
}

func TestUptimeMonitorHttpResource_Schema(t *testing.T) {
	r := NewUptimeMonitorHttpResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	// Verify schema is not nil
	require.NotNil(t, resp.Schema)
	require.NotNil(t, resp.Schema.Attributes)
}

func TestUptimeMonitorHttpResource_Configure(t *testing.T) {
	// Create a real client for testing
	realClient := &client.Client{}

	r := &uptimeMonitorHttpResource{}
	r.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: realClient,
	}, &resource.ConfigureResponse{})

	// Verify resource is properly configured
	require.NotNil(t, r.GetClient())
}
