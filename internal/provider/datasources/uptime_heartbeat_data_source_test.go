package datasources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"terraform-provider-phare/internal/client"
)

func TestUptimeHeartbeatDataSources_Metadata(t *testing.T) {
	for typeName, d := range map[string]datasource.DataSource{
		"phare_uptime_heartbeat":  NewUptimeHeartbeatDataSource(),
		"phare_uptime_heartbeats": NewUptimeHeartbeatsDataSource(),
	} {
		resp := &datasource.MetadataResponse{}
		d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "phare"}, resp)
		require.Equal(t, typeName, resp.TypeName)
	}
}

func TestUptimeHeartbeatDataSource_Schema(t *testing.T) {
	resp := &datasource.SchemaResponse{}
	NewUptimeHeartbeatDataSource().Schema(context.Background(), datasource.SchemaRequest{}, resp)

	require.True(t, resp.Schema.Attributes["id"].IsRequired())
	require.True(t, resp.Schema.Attributes["project_scope"].IsOptional())
	for key := range heartbeatSchemaAttributes() {
		require.Contains(t, resp.Schema.Attributes, key)
	}

	pingURL, ok := resp.Schema.Attributes["ping_url"].(schema.StringAttribute)
	require.True(t, ok)
	require.True(t, pingURL.Sensitive)
}

func TestUptimeHeartbeatsDataSource_Schema(t *testing.T) {
	resp := &datasource.SchemaResponse{}
	NewUptimeHeartbeatsDataSource().Schema(context.Background(), datasource.SchemaRequest{}, resp)

	require.True(t, resp.Schema.Attributes["tags"].IsOptional())
	heartbeats, ok := resp.Schema.Attributes["heartbeats"].(schema.ListNestedAttribute)
	require.True(t, ok)
	require.True(t, heartbeats.Computed)
	require.Len(t, heartbeats.NestedObject.Attributes, len(heartbeatSchemaAttributes()))
}

func TestUptimeHeartbeatDataSources_Configure(t *testing.T) {
	single := &uptimeHeartbeatDataSource{}
	single.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: &client.Client{}}, &datasource.ConfigureResponse{})
	require.NotNil(t, single.GetClient())

	list := &uptimeHeartbeatsDataSource{}
	list.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: &client.Client{}}, &datasource.ConfigureResponse{})
	require.NotNil(t, list.GetClient())
}

func TestMapHeartbeatToModel(t *testing.T) {
	t.Run("never pinged heartbeat", func(t *testing.T) {
		var diags diag.Diagnostics
		model := mapHeartbeatToModel(&client.HeartbeatResponse{
			ID:        1,
			ProjectID: 2,
			Name:      "Nightly backup",
			Interval:  86400,
			Grace:     300,
			Status:    "waiting",
			PingURL:   "https://beacon.phare.io/heartbeats/token",
		}, &diags)

		require.False(t, diags.HasError())
		require.Equal(t, int64(1), model.ID.ValueInt64())
		require.Equal(t, int64(300), model.Grace.ValueInt64())
		require.Equal(t, "waiting", model.Status.ValueString())
		require.Equal(t, "https://beacon.phare.io/heartbeats/token", model.PingURL.ValueString())
		require.True(t, model.DisplayName.IsNull())
		require.True(t, model.LastPingedAt.IsNull())
		require.True(t, model.Tags.IsNull())
	})

	t.Run("pinged heartbeat with display name and tags", func(t *testing.T) {
		var diags diag.Diagnostics
		model := mapHeartbeatToModel(&client.HeartbeatResponse{
			DisplayName:  new("Backups"),
			Tags:         []string{"environment:production"},
			Status:       "online",
			LastPingedAt: new("2026-11-16T10:00:00.000000Z"),
		}, &diags)

		require.False(t, diags.HasError())
		require.Equal(t, "Backups", model.DisplayName.ValueString())
		require.Equal(t, "2026-11-16T10:00:00.000000Z", model.LastPingedAt.ValueString())
		require.Equal(t, types.SetValueMust(types.StringType, []attr.Value{types.StringValue("environment:production")}), model.Tags)
	})
}
