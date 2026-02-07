package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/require"

	"terraform-provider-phare/internal/client"
)

func TestAlertRuleResource_Metadata(t *testing.T) {
	r := NewAlertRuleResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "phare",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	require.Equal(t, "phare_alert_rule", resp.TypeName)
}

func TestAlertRuleResource_Schema(t *testing.T) {
	r := NewAlertRuleResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	// Verify schema is not nil
	require.NotNil(t, resp.Schema)
	require.NotNil(t, resp.Schema.Attributes)
}

func TestAlertRuleResource_Configure(t *testing.T) {
	// Create a real client for testing
	realClient := &client.Client{}

	r := &alertRuleResource{}
	r.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: realClient,
	}, &resource.ConfigureResponse{})

	// Verify resource is properly configured
	require.NotNil(t, r.GetClient())
}
