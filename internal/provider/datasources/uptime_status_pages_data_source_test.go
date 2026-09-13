package datasources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stretchr/testify/require"

	"terraform-provider-phare/internal/client"
)

func TestUptimeStatusPagesDataSource_Metadata(t *testing.T) {
	d := NewUptimeStatusPagesDataSource()
	req := datasource.MetadataRequest{
		ProviderTypeName: "phare",
	}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.Background(), req, resp)

	require.Equal(t, "phare_uptime_status_pages", resp.TypeName)
}

func TestUptimeStatusPagesDataSource_Schema(t *testing.T) {
	d := NewUptimeStatusPagesDataSource()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.Background(), req, resp)

	// Verify schema is not nil
	require.NotNil(t, resp.Schema)
	require.NotNil(t, resp.Schema.Attributes)

	// Verify status_pages attribute exists and contains show_response_times and display_name
	pagesAttr, ok := resp.Schema.Attributes["status_pages"].(schema.ListNestedAttribute)
	require.True(t, ok)

	showAttr, ok := pagesAttr.NestedObject.Attributes["show_response_times"].(schema.BoolAttribute)
	require.True(t, ok)
	require.True(t, showAttr.Computed)

	compAttr, ok := pagesAttr.NestedObject.Attributes["components"].(schema.ListNestedAttribute)
	require.True(t, ok)
	topDisplayAttr, ok := compAttr.NestedObject.Attributes["display_name"].(schema.StringAttribute)
	require.True(t, ok)
	require.True(t, topDisplayAttr.Computed)

	nestedCompAttr, ok := compAttr.NestedObject.Attributes["components"].(schema.ListNestedAttribute)
	require.True(t, ok)
	childDisplayAttr, ok := nestedCompAttr.NestedObject.Attributes["display_name"].(schema.StringAttribute)
	require.True(t, ok)
	require.True(t, childDisplayAttr.Computed)
}

func TestUptimeStatusPagesDataSource_Configure(t *testing.T) {
	// Create a real client for testing
	realClient := &client.Client{}

	d := &uptimeStatusPagesDataSource{}
	d.Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: realClient,
	}, &datasource.ConfigureResponse{})

	// Verify data source is properly configured
	require.NotNil(t, d.GetClient())
}
