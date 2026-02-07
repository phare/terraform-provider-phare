package datasources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/stretchr/testify/require"

	"terraform-provider-phare/internal/client"
)

func TestIntegrationDataSource_Metadata(t *testing.T) {
	d := NewIntegrationDataSource()
	req := datasource.MetadataRequest{
		ProviderTypeName: "phare",
	}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.Background(), req, resp)

	require.Equal(t, "phare_integration", resp.TypeName)
}

func TestIntegrationDataSource_Schema(t *testing.T) {
	d := NewIntegrationDataSource()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.Background(), req, resp)

	// Verify schema is not nil
	require.NotNil(t, resp.Schema)
	require.NotNil(t, resp.Schema.Attributes)
}

func TestIntegrationDataSource_Configure(t *testing.T) {
	// Create a real client for testing
	realClient := &client.Client{}

	d := &integrationDataSource{}
	d.Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: realClient,
	}, &datasource.ConfigureResponse{})

	// Verify data source is properly configured
	require.NotNil(t, d.GetClient())
}
