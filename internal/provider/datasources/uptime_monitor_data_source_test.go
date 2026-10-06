package datasources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/stretchr/testify/require"

	"terraform-provider-phare/internal/client"
)

func TestUptimeMonitorDataSource_Metadata(t *testing.T) {
	d := NewUptimeMonitorDataSource()
	req := datasource.MetadataRequest{
		ProviderTypeName: "phare",
	}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.Background(), req, resp)

	require.Equal(t, "phare_uptime_monitor", resp.TypeName)
}

func TestMapMonitorToModel_Tags(t *testing.T) {
	resp := &datasource.ReadResponse{}

	monitor := &client.MonitorResponse{
		ID:        1,
		ProjectID: 1,
		Name:      "mon",
		Protocol:  "http",
		Regions:   []string{"eu-fra-cdg"},
		Tags:      []string{"environment:production", "team:backend"},
	}

	model := mapMonitorToModel(context.Background(), monitor, resp)
	require.False(t, resp.Diagnostics.HasError())

	var tags []string
	require.False(t, model.Tags.ElementsAs(context.Background(), &tags, false).HasError())
	require.Equal(t, []string{"environment:production", "team:backend"}, tags)

	monitor.Tags = nil
	model = mapMonitorToModel(context.Background(), monitor, resp)
	require.False(t, resp.Diagnostics.HasError())
	require.True(t, model.Tags.IsNull())
}

func TestUptimeMonitorDataSource_Schema(t *testing.T) {
	d := NewUptimeMonitorDataSource()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.Background(), req, resp)

	// Verify schema is not nil
	require.NotNil(t, resp.Schema)
	require.NotNil(t, resp.Schema.Attributes)
}

func TestUptimeMonitorDataSource_Configure(t *testing.T) {
	// Create a real client for testing
	realClient := &client.Client{}

	d := &uptimeMonitorDataSource{}
	d.Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: realClient,
	}, &datasource.ConfigureResponse{})

	// Verify data source is properly configured
	require.NotNil(t, d.GetClient())
}
