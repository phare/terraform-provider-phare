package datasources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

func TestMapMonitorToModel_DisplayName(t *testing.T) {
	resp := &datasource.ReadResponse{}

	displayName := "Website"
	monitor := &client.MonitorResponse{
		ID:          1,
		ProjectID:   1,
		Name:        "mon",
		DisplayName: &displayName,
		Protocol:    "http",
		Regions:     []string{"eu-fra-cdg"},
	}

	model := mapMonitorToModel(context.Background(), monitor, resp)
	require.False(t, resp.Diagnostics.HasError())
	require.Equal(t, types.StringValue("Website"), model.DisplayName)

	monitor.DisplayName = nil
	model = mapMonitorToModel(context.Background(), monitor, resp)
	require.False(t, resp.Diagnostics.HasError())
	require.True(t, model.DisplayName.IsNull())
}

func TestMapMonitorToModel_Secrets(t *testing.T) {
	ctx := context.Background()
	resp := &datasource.ReadResponse{}

	monitor := &client.MonitorResponse{
		ID:       1,
		Name:     "mon",
		Protocol: "http",
		Request: client.MonitorRequestConfig{
			UserAgentSecretEnabled: new(true),
			Headers: []client.MonitorRequestHeader{
				{Name: "Authorization", Secret: new(true)},
				{Name: "X-Public", Value: new("hello"), Secret: new(false)},
			},
		},
	}

	model := mapMonitorToModel(ctx, monitor, resp)
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)

	request := model.Request.Attributes()
	require.True(t, request["user_agent_secret"].IsNull())
	require.Equal(t, types.BoolValue(true), request["user_agent_secret_enabled"])

	headers := request["headers"].(types.List).Elements()
	require.Len(t, headers, 2)

	secret := headers[0].(types.Object).Attributes()
	require.Equal(t, types.StringValue("Authorization"), secret["name"])
	require.True(t, secret["value"].IsNull())
	require.Equal(t, types.BoolValue(true), secret["secret"])

	public := headers[1].(types.Object).Attributes()
	require.Equal(t, types.StringValue("hello"), public["value"])
	require.Equal(t, types.BoolValue(false), public["secret"])

	t.Run("monitors of other protocols have no user agent secret flag", func(t *testing.T) {
		monitor := &client.MonitorResponse{ID: 2, Name: "tcp", Protocol: "tcp", Request: client.MonitorRequestConfig{Host: new("example.com")}}
		model := mapMonitorToModel(ctx, monitor, resp)
		require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
		require.True(t, model.Request.Attributes()["user_agent_secret_enabled"].IsNull())
		require.True(t, model.Request.Attributes()["headers"].IsNull())
	})
}

func TestUptimeMonitorDataSource_Schema(t *testing.T) {
	d := NewUptimeMonitorDataSource()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.Background(), req, resp)

	// Verify schema is not nil
	require.NotNil(t, resp.Schema)
	require.NotNil(t, resp.Schema.Attributes)

	idAttr, ok := resp.Schema.Attributes["id"].(schema.Int64Attribute)
	require.True(t, ok)
	require.True(t, idAttr.Required)
	require.False(t, idAttr.Computed)

	displayNameAttr, ok := resp.Schema.Attributes["display_name"].(schema.StringAttribute)
	require.True(t, ok)
	require.True(t, displayNameAttr.Computed)
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
