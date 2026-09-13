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

func TestUptimeStatusPageDataSource_Metadata(t *testing.T) {
	d := NewUptimeStatusPageDataSource()
	req := datasource.MetadataRequest{
		ProviderTypeName: "phare",
	}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.Background(), req, resp)

	require.Equal(t, "phare_uptime_status_page", resp.TypeName)
}

func TestUptimeStatusPageDataSource_Schema(t *testing.T) {
	d := NewUptimeStatusPageDataSource()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.Background(), req, resp)

	// Verify schema is not nil
	require.NotNil(t, resp.Schema)
	require.NotNil(t, resp.Schema.Attributes)

	// Verify show_response_times exists
	showAttr, ok := resp.Schema.Attributes["show_response_times"].(schema.BoolAttribute)
	require.True(t, ok)
	require.True(t, showAttr.Computed)

	// Verify display_name exists on top-level and nested components
	compAttr, ok := resp.Schema.Attributes["components"].(schema.ListNestedAttribute)
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

func TestUptimeStatusPageDataSource_Configure(t *testing.T) {
	// Create a real client for testing
	realClient := &client.Client{}

	d := &uptimeStatusPageDataSource{}
	d.Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: realClient,
	}, &datasource.ConfigureResponse{})

	// Verify data source is properly configured
	require.NotNil(t, d.GetClient())
}

func TestUptimeStatusPageDataSource_MapStatusPageToModel(t *testing.T) {
	ctx := context.Background()

	t.Run("default show_response_times when nil", func(t *testing.T) {
		page := &client.StatusPageResponse{
			ID:                1,
			ProjectID:         10,
			Name:              "Test Page",
			Title:             "Test Title",
			Description:       "Test Description",
			WebsiteURL:        "https://example.com",
			ShowResponseTimes: nil,
		}

		resp := &datasource.ReadResponse{}
		model := mapStatusPageToModel(ctx, page, resp)
		require.False(t, resp.Diagnostics.HasError())
		require.True(t, model.ShowResponseTimes.ValueBool())
	})

	t.Run("explicit show_response_times false", func(t *testing.T) {
		showRT := false
		page := &client.StatusPageResponse{
			ID:                1,
			ProjectID:         10,
			Name:              "Test Page",
			Title:             "Test Title",
			Description:       "Test Description",
			WebsiteURL:        "https://example.com",
			ShowResponseTimes: &showRT,
		}

		resp := &datasource.ReadResponse{}
		model := mapStatusPageToModel(ctx, page, resp)
		require.False(t, resp.Diagnostics.HasError())
		require.False(t, model.ShowResponseTimes.ValueBool())
	})

	t.Run("component display_name mapping", func(t *testing.T) {
		topDN := "Top Monitor"
		childDN := "Child Monitor"
		id1 := int64(101)
		id2 := int64(102)
		grpName := "Group 1"

		page := &client.StatusPageResponse{
			ID:          1,
			ProjectID:   10,
			Name:        "Test Page",
			Title:       "Test Title",
			Description: "Test Description",
			WebsiteURL:  "https://example.com",
			Components: []client.StatusPageComponent{
				{
					ComponentableType: "uptime/monitor",
					ComponentableID:   &id1,
					DisplayName:       &topDN,
				},
				{
					ComponentableType: "uptime/group",
					Name:              &grpName,
					Components: []client.StatusPageComponent{
						{
							ComponentableType: "uptime/monitor",
							ComponentableID:   &id2,
							DisplayName:       &childDN,
						},
					},
				},
			},
		}

		resp := &datasource.ReadResponse{}
		model := mapStatusPageToModel(ctx, page, resp)
		require.False(t, resp.Diagnostics.HasError())
		require.False(t, model.Components.IsNull())

		var compElements []types.Object
		diags := model.Components.ElementsAs(ctx, &compElements, false)
		require.False(t, diags.HasError())
		require.Len(t, compElements, 2)

		// Top monitor check
		topAttrs := compElements[0].Attributes()
		require.Equal(t, types.StringValue("Top Monitor"), topAttrs["display_name"])

		// Group child check
		groupAttrs := compElements[1].Attributes()
		require.True(t, groupAttrs["display_name"].(types.String).IsNull())

		childList := groupAttrs["components"].(types.List)
		var childElements []types.Object
		diags = childList.ElementsAs(ctx, &childElements, false)
		require.False(t, diags.HasError())
		require.Len(t, childElements, 1)
		childAttrs := childElements[0].Attributes()
		require.Equal(t, types.StringValue("Child Monitor"), childAttrs["display_name"])
	})
}
