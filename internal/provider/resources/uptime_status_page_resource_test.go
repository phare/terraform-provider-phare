package resources

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"

	"terraform-provider-phare/internal/client"
)

func int64Ptr(v int64) *int64 {
	return &v
}

func stringPtr(v string) *string {
	return &v
}

func createComponent(t *testing.T, compType string, compID *int64, groupName *string, children []NestedComponentModel) ComponentModel {
	var idVal types.Int64
	if compID != nil {
		idVal = types.Int64Value(*compID)
	} else {
		idVal = types.Int64Null()
	}

	var nameVal types.String
	if groupName != nil {
		nameVal = types.StringValue(*groupName)
	} else {
		nameVal = types.StringNull()
	}

	var childrenVal types.List
	if children != nil {
		var diags diag.Diagnostics
		childrenVal, diags = types.ListValueFrom(context.Background(), types.ObjectType{AttrTypes: NestedComponentModelAttrTypes}, children)
		require.False(t, diags.HasError())
	} else {
		childrenVal = types.ListNull(types.ObjectType{AttrTypes: NestedComponentModelAttrTypes})
	}

	return ComponentModel{
		ComponentableType: types.StringValue(compType),
		ComponentableID:   idVal,
		Name:              nameVal,
		IsExpanded:        types.BoolNull(),
		Components:        childrenVal,
	}
}

func createNestedComponent(compType string, compID *int64) NestedComponentModel {
	var idVal types.Int64
	if compID != nil {
		idVal = types.Int64Value(*compID)
	} else {
		idVal = types.Int64Null()
	}
	return NestedComponentModel{
		ComponentableType: types.StringValue(compType),
		ComponentableID:   idVal,
	}
}

func createComponentsList(t *testing.T, comps []ComponentModel) types.List {
	l, diags := types.ListValueFrom(context.Background(), types.ObjectType{AttrTypes: ComponentModelAttrTypes}, comps)
	require.False(t, diags.HasError())
	return l
}

func validStatusPageModel(t *testing.T) UptimeStatusPageModel {
	compList := createComponentsList(t, []ComponentModel{
		createComponent(t, "uptime/monitor", int64Ptr(101), nil, nil),
	})

	return UptimeStatusPageModel{
		AccessIPs:             types.ListNull(types.StringType),
		AccessPassword:        types.StringNull(),
		AccessPasswordEnabled: types.BoolNull(),
		AccessToken:           types.StringNull(),
		AccessTokenEnabled:    types.BoolNull(),
		ColorScheme:           types.StringValue("all"),
		Components:            compList,
		CreatedAt:             types.StringNull(),
		Description:           types.StringValue("Status page description"),
		Domain:                types.StringNull(),
		FaviconDark:           types.StringNull(),
		FaviconLight:          types.StringNull(),
		Id:                    types.Int64Value(1),
		LogoDark:              types.StringNull(),
		LogoLight:             types.StringNull(),
		Name:                  types.StringValue("Status Page"),
		ProjectId:             types.Int64Value(1),
		SearchEngineIndexed:   types.BoolValue(true),
		Subdomain:             types.StringValue("status-page"),
		SubscriptionChannels:  types.ListNull(types.StringType),
		Theme:                 types.ObjectNull(ThemeModelAttrTypes),
		Timeframe:             types.Int64Value(30),
		Title:                 types.StringValue("Status Page Title"),
		UpdatedAt:             types.StringNull(),
		WebsiteUrl:            types.StringValue("https://example.com"),
		ProjectScope:          types.DynamicNull(),
	}
}

func TestUptimeStatusPageResource_Metadata(t *testing.T) {
	r := NewUptimeStatusPageResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "phare",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	require.Equal(t, "phare_uptime_status_page", resp.TypeName)
}

func TestUptimeStatusPageResource_Schema(t *testing.T) {
	r := NewUptimeStatusPageResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	require.NotNil(t, resp.Schema)
	require.NotNil(t, resp.Schema.Attributes)
}

func TestUptimeStatusPageResource_Configure(t *testing.T) {
	realClient := &client.Client{}

	r := &uptimeStatusPageResource{}
	r.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: realClient,
	}, &resource.ConfigureResponse{})

	require.NotNil(t, r.GetClient())
}

func TestUptimeStatusPageResource_NameValidation(t *testing.T) {
	r := NewUptimeStatusPageResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	nameAttr, ok := resp.Schema.Attributes["name"].(schema.StringAttribute)
	require.True(t, ok)
	require.NotEmpty(t, nameAttr.Validators)

	testCases := []struct {
		name        string
		val         types.String
		expectError bool
	}{
		{"valid 2 chars", types.StringValue("ab"), false},
		{"valid 30 chars", types.StringValue(strings.Repeat("a", 30)), false},
		{"valid 30 chars with surrounding whitespace", types.StringValue("   " + strings.Repeat("a", 30) + "   "), false},
		{"invalid whitespace only", types.StringValue("   "), true},
		{"invalid empty string", types.StringValue(""), true},
		{"invalid 1 char", types.StringValue("a"), true},
		{"invalid 31 chars", types.StringValue(strings.Repeat("a", 31)), true},
		{"null value skipped", types.StringNull(), false},
		{"unknown value skipped", types.StringUnknown(), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valReq := validator.StringRequest{
				Path:        path.Root("name"),
				ConfigValue: tc.val,
			}
			valResp := &validator.StringResponse{}

			for _, v := range nameAttr.Validators {
				v.ValidateString(context.Background(), valReq, valResp)
			}

			if tc.expectError {
				require.True(t, valResp.Diagnostics.HasError())
			} else {
				require.False(t, valResp.Diagnostics.HasError())
			}
		})
	}
}

func TestUptimeStatusPageResource_SubdomainValidation(t *testing.T) {
	r := NewUptimeStatusPageResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	subdomainAttr, ok := resp.Schema.Attributes["subdomain"].(schema.StringAttribute)
	require.True(t, ok)
	require.NotEmpty(t, subdomainAttr.Validators)

	testCases := []struct {
		name        string
		val         types.String
		expectError bool
	}{
		{"valid lowercase and numbers", types.StringValue("status-page-123"), false},
		{"valid 2 chars", types.StringValue("ab"), false},
		{"valid 30 chars", types.StringValue(strings.Repeat("a", 30)), false},
		{"invalid 1 char", types.StringValue("a"), true},
		{"invalid 31 chars", types.StringValue(strings.Repeat("a", 31)), true},
		{"invalid uppercase", types.StringValue("Status-Page"), true},
		{"invalid underscore", types.StringValue("status_page"), true},
		{"invalid dot", types.StringValue("status.page"), true},
		{"invalid special char", types.StringValue("status!page"), true},
		{"null value skipped", types.StringNull(), false},
		{"unknown value skipped", types.StringUnknown(), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valReq := validator.StringRequest{
				Path:        path.Root("subdomain"),
				ConfigValue: tc.val,
			}
			valResp := &validator.StringResponse{}

			for _, v := range subdomainAttr.Validators {
				v.ValidateString(context.Background(), valReq, valResp)
			}

			if tc.expectError {
				require.True(t, valResp.Diagnostics.HasError())
			} else {
				require.False(t, valResp.Diagnostics.HasError())
			}
		})
	}
}

func TestUptimeStatusPageResource_DomainValidation(t *testing.T) {
	r := NewUptimeStatusPageResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	domainAttr, ok := resp.Schema.Attributes["domain"].(schema.StringAttribute)
	require.True(t, ok)
	require.NotEmpty(t, domainAttr.Validators)

	testCases := []struct {
		name        string
		val         types.String
		expectError bool
	}{
		{"valid 4 chars", types.StringValue("a.co"), false},
		{"valid 60 chars", types.StringValue(strings.Repeat("a", 60)), false},
		{"invalid 3 chars", types.StringValue("a.c"), true},
		{"invalid 61 chars", types.StringValue(strings.Repeat("a", 61)), true},
		{"null value skipped", types.StringNull(), false},
		{"unknown value skipped", types.StringUnknown(), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valReq := validator.StringRequest{
				Path:        path.Root("domain"),
				ConfigValue: tc.val,
			}
			valResp := &validator.StringResponse{}

			for _, v := range domainAttr.Validators {
				v.ValidateString(context.Background(), valReq, valResp)
			}

			if tc.expectError {
				require.True(t, valResp.Diagnostics.HasError())
			} else {
				require.False(t, valResp.Diagnostics.HasError())
			}
		})
	}
}

func TestUptimeStatusPageResource_TitleValidation(t *testing.T) {
	r := NewUptimeStatusPageResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	titleAttr, ok := resp.Schema.Attributes["title"].(schema.StringAttribute)
	require.True(t, ok)
	require.NotEmpty(t, titleAttr.Validators)

	testCases := []struct {
		name        string
		val         types.String
		expectError bool
	}{
		{"valid 2 chars", types.StringValue("ab"), false},
		{"valid 150 chars", types.StringValue(strings.Repeat("a", 150)), false},
		{"invalid 1 char", types.StringValue("a"), true},
		{"invalid 151 chars", types.StringValue(strings.Repeat("a", 151)), true},
		{"null value skipped", types.StringNull(), false},
		{"unknown value skipped", types.StringUnknown(), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valReq := validator.StringRequest{
				Path:        path.Root("title"),
				ConfigValue: tc.val,
			}
			valResp := &validator.StringResponse{}

			for _, v := range titleAttr.Validators {
				v.ValidateString(context.Background(), valReq, valResp)
			}

			if tc.expectError {
				require.True(t, valResp.Diagnostics.HasError())
			} else {
				require.False(t, valResp.Diagnostics.HasError())
			}
		})
	}
}

func TestUptimeStatusPageResource_DescriptionValidation(t *testing.T) {
	r := NewUptimeStatusPageResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	descAttr, ok := resp.Schema.Attributes["description"].(schema.StringAttribute)
	require.True(t, ok)
	require.NotEmpty(t, descAttr.Validators)

	testCases := []struct {
		name        string
		val         types.String
		expectError bool
	}{
		{"valid 2 chars", types.StringValue("ab"), false},
		{"valid 250 chars", types.StringValue(strings.Repeat("a", 250)), false},
		{"invalid 1 char", types.StringValue("a"), true},
		{"invalid 251 chars", types.StringValue(strings.Repeat("a", 251)), true},
		{"null value skipped", types.StringNull(), false},
		{"unknown value skipped", types.StringUnknown(), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valReq := validator.StringRequest{
				Path:        path.Root("description"),
				ConfigValue: tc.val,
			}
			valResp := &validator.StringResponse{}

			for _, v := range descAttr.Validators {
				v.ValidateString(context.Background(), valReq, valResp)
			}

			if tc.expectError {
				require.True(t, valResp.Diagnostics.HasError())
			} else {
				require.False(t, valResp.Diagnostics.HasError())
			}
		})
	}
}

func TestUptimeStatusPageResource_WebsiteUrlValidation(t *testing.T) {
	r := NewUptimeStatusPageResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	urlAttr, ok := resp.Schema.Attributes["website_url"].(schema.StringAttribute)
	require.True(t, ok)
	require.NotEmpty(t, urlAttr.Validators)

	testCases := []struct {
		name        string
		val         types.String
		expectError bool
	}{
		{"valid http url", types.StringValue("http://example.com"), false},
		{"valid https url", types.StringValue("https://status.example.com/system"), false},
		{"invalid ftp url", types.StringValue("ftp://example.com"), true},
		{"invalid missing scheme", types.StringValue("example.com"), true},
		{"invalid over 250 chars", types.StringValue("https://" + strings.Repeat("a", 245)), true},
		{"null value skipped", types.StringNull(), false},
		{"unknown value skipped", types.StringUnknown(), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valReq := validator.StringRequest{
				Path:        path.Root("website_url"),
				ConfigValue: tc.val,
			}
			valResp := &validator.StringResponse{}

			for _, v := range urlAttr.Validators {
				v.ValidateString(context.Background(), valReq, valResp)
			}

			if tc.expectError {
				require.True(t, valResp.Diagnostics.HasError())
			} else {
				require.False(t, valResp.Diagnostics.HasError())
			}
		})
	}
}

func TestUptimeStatusPageResource_TimeframeValidation(t *testing.T) {
	r := NewUptimeStatusPageResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	timeframeAttr, ok := resp.Schema.Attributes["timeframe"].(schema.Int64Attribute)
	require.True(t, ok)
	require.NotEmpty(t, timeframeAttr.Validators)

	testCases := []struct {
		name        string
		val         types.Int64
		expectError bool
	}{
		{"valid 30", types.Int64Value(30), false},
		{"valid 60", types.Int64Value(60), false},
		{"valid 90", types.Int64Value(90), false},
		{"invalid 15", types.Int64Value(15), true},
		{"invalid 45", types.Int64Value(45), true},
		{"invalid 100", types.Int64Value(100), true},
		{"null value skipped", types.Int64Null(), false},
		{"unknown value skipped", types.Int64Unknown(), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valReq := validator.Int64Request{
				Path:        path.Root("timeframe"),
				ConfigValue: tc.val,
			}
			valResp := &validator.Int64Response{}

			for _, v := range timeframeAttr.Validators {
				v.ValidateInt64(context.Background(), valReq, valResp)
			}

			if tc.expectError {
				require.True(t, valResp.Diagnostics.HasError())
			} else {
				require.False(t, valResp.Diagnostics.HasError())
			}
		})
	}
}

func TestUptimeStatusPageResource_SubscriptionChannelsValidation(t *testing.T) {
	r := NewUptimeStatusPageResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	subAttr, ok := resp.Schema.Attributes["subscription_channels"].(schema.ListAttribute)
	require.True(t, ok)
	require.NotEmpty(t, subAttr.Validators)

	testCases := []struct {
		name        string
		val         []attr.Value
		expectError bool
	}{
		{"valid empty", []attr.Value{}, false},
		{"valid single rss", []attr.Value{types.StringValue("rss")}, false},
		{"valid atom and slack", []attr.Value{types.StringValue("atom"), types.StringValue("slack")}, false},
		{"valid all three", []attr.Value{types.StringValue("rss"), types.StringValue("atom"), types.StringValue("slack")}, false},
		{"invalid duplicates", []attr.Value{types.StringValue("rss"), types.StringValue("rss")}, true},
		{"invalid channel name", []attr.Value{types.StringValue("email")}, true},
		{"invalid size greater than 3", []attr.Value{
			types.StringValue("rss"), types.StringValue("atom"), types.StringValue("slack"), types.StringValue("extra"),
		}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			listVal, diags := types.ListValue(types.StringType, tc.val)
			require.False(t, diags.HasError())

			valReq := validator.ListRequest{
				Path:        path.Root("subscription_channels"),
				ConfigValue: listVal,
			}
			valResp := &validator.ListResponse{}

			for _, v := range subAttr.Validators {
				v.ValidateList(context.Background(), valReq, valResp)
			}

			if tc.expectError {
				require.True(t, valResp.Diagnostics.HasError())
			} else {
				require.False(t, valResp.Diagnostics.HasError())
			}
		})
	}
}

func TestUptimeStatusPageResource_AccessTokenValidation(t *testing.T) {
	r := NewUptimeStatusPageResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attr, ok := resp.Schema.Attributes["access_token"].(schema.StringAttribute)
	require.True(t, ok)
	require.NotEmpty(t, attr.Validators)

	testCases := []struct {
		name        string
		val         types.String
		expectError bool
	}{
		{"valid 8 chars", types.StringValue("12345678"), false},
		{"valid 500 chars", types.StringValue(strings.Repeat("a", 500)), false},
		{"invalid 7 chars", types.StringValue("1234567"), true},
		{"invalid 501 chars", types.StringValue(strings.Repeat("a", 501)), true},
		{"null value skipped", types.StringNull(), false},
		{"unknown value skipped", types.StringUnknown(), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valReq := validator.StringRequest{
				Path:        path.Root("access_token"),
				ConfigValue: tc.val,
			}
			valResp := &validator.StringResponse{}

			for _, v := range attr.Validators {
				v.ValidateString(context.Background(), valReq, valResp)
			}

			if tc.expectError {
				require.True(t, valResp.Diagnostics.HasError())
			} else {
				require.False(t, valResp.Diagnostics.HasError())
			}
		})
	}
}

func TestUptimeStatusPageResource_AccessPasswordValidation(t *testing.T) {
	r := NewUptimeStatusPageResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attr, ok := resp.Schema.Attributes["access_password"].(schema.StringAttribute)
	require.True(t, ok)
	require.NotEmpty(t, attr.Validators)

	testCases := []struct {
		name        string
		val         types.String
		expectError bool
	}{
		{"valid 8 chars", types.StringValue("12345678"), false},
		{"valid 500 chars", types.StringValue(strings.Repeat("a", 500)), false},
		{"invalid 7 chars", types.StringValue("1234567"), true},
		{"invalid 501 chars", types.StringValue(strings.Repeat("a", 501)), true},
		{"null value skipped", types.StringNull(), false},
		{"unknown value skipped", types.StringUnknown(), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valReq := validator.StringRequest{
				Path:        path.Root("access_password"),
				ConfigValue: tc.val,
			}
			valResp := &validator.StringResponse{}

			for _, v := range attr.Validators {
				v.ValidateString(context.Background(), valReq, valResp)
			}

			if tc.expectError {
				require.True(t, valResp.Diagnostics.HasError())
			} else {
				require.False(t, valResp.Diagnostics.HasError())
			}
		})
	}
}

func TestUptimeStatusPageResource_AccessIpsValidation(t *testing.T) {
	r := NewUptimeStatusPageResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attrObj, ok := resp.Schema.Attributes["access_ips"].(schema.ListAttribute)
	require.True(t, ok)
	require.NotEmpty(t, attrObj.Validators)

	// Build 50 IPs
	fiftyIps := make([]attr.Value, 50)
	for i := 0; i < 50; i++ {
		fiftyIps[i] = types.StringValue(strings.Repeat("1", i+1))
	}

	testCases := []struct {
		name        string
		val         []attr.Value
		expectError bool
	}{
		{"valid 1 IP", []attr.Value{types.StringValue("192.168.1.1")}, false},
		{"valid 50 IPs", fiftyIps, false},
		{"invalid empty", []attr.Value{}, true},
		{"invalid duplicates", []attr.Value{types.StringValue("1.1.1.1"), types.StringValue("1.1.1.1")}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			listVal, diags := types.ListValue(types.StringType, tc.val)
			require.False(t, diags.HasError())

			valReq := validator.ListRequest{
				Path:        path.Root("access_ips"),
				ConfigValue: listVal,
			}
			valResp := &validator.ListResponse{}

			for _, v := range attrObj.Validators {
				v.ValidateList(context.Background(), valReq, valResp)
			}

			if tc.expectError {
				require.True(t, valResp.Diagnostics.HasError())
			} else {
				require.False(t, valResp.Diagnostics.HasError())
			}
		})
	}
}

func TestUptimeStatusPageResource_ComponentsValidation(t *testing.T) {
	r := NewUptimeStatusPageResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	componentsAttr, ok := resp.Schema.Attributes["components"].(schema.ListNestedAttribute)
	require.True(t, ok)
	require.NotEmpty(t, componentsAttr.Validators)

	compType := types.ObjectType{AttrTypes: ComponentModelAttrTypes}

	testCases := []struct {
		name        string
		val         []attr.Value
		expectError bool
	}{
		{"invalid 0 components", []attr.Value{}, true},
		{"valid 1 component", []attr.Value{
			types.ObjectValueMust(ComponentModelAttrTypes, map[string]attr.Value{
				"componentable_type": types.StringValue("uptime/monitor"),
				"componentable_id":   types.Int64Value(1),
				"name":               types.StringNull(),
				"is_expanded":        types.BoolNull(),
				"components":         types.ListNull(types.ObjectType{AttrTypes: NestedComponentModelAttrTypes}),
			}),
		}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			listVal, diags := types.ListValue(compType, tc.val)
			require.False(t, diags.HasError())

			valReq := validator.ListRequest{
				Path:        path.Root("components"),
				ConfigValue: listVal,
			}
			valResp := &validator.ListResponse{}

			for _, v := range componentsAttr.Validators {
				v.ValidateList(context.Background(), valReq, valResp)
			}

			if tc.expectError {
				require.True(t, valResp.Diagnostics.HasError())
			} else {
				require.False(t, valResp.Diagnostics.HasError())
			}
		})
	}
}

func TestUptimeStatusPageResource_ModifyPlan(t *testing.T) {
	r := &uptimeStatusPageResource{}
	realClient, err := client.NewClient("https://api.phare.io", "token", 10*time.Second, "123", "", "1.0", "1.0", true)
	require.NoError(t, err)

	r.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: realClient,
	}, &resource.ConfigureResponse{})

	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	t.Run("destroy plan skipped", func(t *testing.T) {
		plan := tfsdk.Plan{Raw: tftypes.NewValue(tftypes.Object{}, nil)}
		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)
		require.False(t, resp.Diagnostics.HasError())
	})

	t.Run("valid plan", func(t *testing.T) {
		planData := validStatusPageModel(t)
		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)
		require.False(t, resp.Diagnostics.HasError())
	})

	t.Run("valid plan with group and nested monitor", func(t *testing.T) {
		planData := validStatusPageModel(t)
		planData.Components = createComponentsList(t, []ComponentModel{
			createComponent(t, "uptime/monitor", int64Ptr(101), nil, nil),
			createComponent(t, "uptime/group", nil, stringPtr("Database Group"), []NestedComponentModel{
				createNestedComponent("uptime/monitor", int64Ptr(102)),
			}),
		})

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)
		require.False(t, resp.Diagnostics.HasError())
	})

	t.Run("name trimmed to less than 2 characters", func(t *testing.T) {
		planData := validStatusPageModel(t)
		planData.Name = types.StringValue("  a  ")

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)
		require.True(t, resp.Diagnostics.HasError())

		found := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Summary() == "Invalid Name Length" {
				found = true
			}
		}
		require.True(t, found)
	})

	t.Run("name with whitespace only", func(t *testing.T) {
		planData := validStatusPageModel(t)
		planData.Name = types.StringValue("     ")

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)
		require.True(t, resp.Diagnostics.HasError())

		found := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Summary() == "Invalid Name Length" {
				found = true
			}
		}
		require.True(t, found)
	})

	t.Run("subdomain without letters", func(t *testing.T) {
		planData := validStatusPageModel(t)
		planData.Subdomain = types.StringValue("12345")

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)
		require.True(t, resp.Diagnostics.HasError())

		found := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Summary() == "Invalid Subdomain" {
				found = true
			}
		}
		require.True(t, found)
	})

	t.Run("subdomain with digits and dashes only", func(t *testing.T) {
		planData := validStatusPageModel(t)
		planData.Subdomain = types.StringValue("123-456")

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)
		require.True(t, resp.Diagnostics.HasError())

		found := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Summary() == "Invalid Subdomain" {
				found = true
			}
		}
		require.True(t, found)
	})

	t.Run("top-level monitor missing componentable_id", func(t *testing.T) {
		planData := validStatusPageModel(t)
		planData.Components = createComponentsList(t, []ComponentModel{
			createComponent(t, "uptime/monitor", nil, nil, nil),
		})

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)
		require.True(t, resp.Diagnostics.HasError())

		found := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Summary() == "Missing Component ID" {
				found = true
			}
		}
		require.True(t, found)
	})

	t.Run("group with empty name", func(t *testing.T) {
		planData := validStatusPageModel(t)
		planData.Components = createComponentsList(t, []ComponentModel{
			createComponent(t, "uptime/group", nil, stringPtr("   "), []NestedComponentModel{
				createNestedComponent("uptime/monitor", int64Ptr(102)),
			}),
		})

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)
		require.True(t, resp.Diagnostics.HasError())

		found := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Summary() == "Invalid Group Name" {
				found = true
			}
		}
		require.True(t, found)
	})

	t.Run("group with unknown name allowed during planning", func(t *testing.T) {
		planData := validStatusPageModel(t)
		comp := createComponent(t, "uptime/group", nil, nil, []NestedComponentModel{
			createNestedComponent("uptime/monitor", int64Ptr(102)),
		})
		comp.Name = types.StringUnknown()
		planData.Components = createComponentsList(t, []ComponentModel{comp})

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)
		require.False(t, resp.Diagnostics.HasError())
	})

	t.Run("group with null components", func(t *testing.T) {
		planData := validStatusPageModel(t)
		planData.Components = createComponentsList(t, []ComponentModel{
			createComponent(t, "uptime/group", nil, stringPtr("Group 1"), nil),
		})

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)
		require.True(t, resp.Diagnostics.HasError())

		found := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Summary() == "Missing Group Components" {
				found = true
			}
		}
		require.True(t, found)
	})

	t.Run("group with empty components list", func(t *testing.T) {
		planData := validStatusPageModel(t)
		planData.Components = createComponentsList(t, []ComponentModel{
			createComponent(t, "uptime/group", nil, stringPtr("Group 1"), []NestedComponentModel{}),
		})

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)
		require.True(t, resp.Diagnostics.HasError())

		found := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Summary() == "Invalid Group Components Count" {
				found = true
			}
		}
		require.True(t, found)
	})

	t.Run("group with invalid child component type", func(t *testing.T) {
		planData := validStatusPageModel(t)
		planData.Components = createComponentsList(t, []ComponentModel{
			createComponent(t, "uptime/group", nil, stringPtr("Group 1"), []NestedComponentModel{
				createNestedComponent("uptime/group", int64Ptr(102)),
			}),
		})

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)
		require.True(t, resp.Diagnostics.HasError())

		found := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Summary() == "Invalid Component Type" {
				found = true
			}
		}
		require.True(t, found)
	})

	t.Run("group child monitor missing componentable_id", func(t *testing.T) {
		planData := validStatusPageModel(t)
		planData.Components = createComponentsList(t, []ComponentModel{
			createComponent(t, "uptime/group", nil, stringPtr("Group 1"), []NestedComponentModel{
				createNestedComponent("uptime/monitor", nil),
			}),
		})

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)
		require.True(t, resp.Diagnostics.HasError())

		found := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Summary() == "Missing Component ID" {
				found = true
			}
		}
		require.True(t, found)
	})

	t.Run("duplicate monitor across top-level components", func(t *testing.T) {
		planData := validStatusPageModel(t)
		planData.Components = createComponentsList(t, []ComponentModel{
			createComponent(t, "uptime/monitor", int64Ptr(101), nil, nil),
			createComponent(t, "uptime/monitor", int64Ptr(101), nil, nil),
		})

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)
		require.True(t, resp.Diagnostics.HasError())

		found := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Summary() == "Duplicate Monitor" {
				found = true
			}
		}
		require.True(t, found)
	})

	t.Run("duplicate monitor between top-level and group", func(t *testing.T) {
		planData := validStatusPageModel(t)
		planData.Components = createComponentsList(t, []ComponentModel{
			createComponent(t, "uptime/monitor", int64Ptr(101), nil, nil),
			createComponent(t, "uptime/group", nil, stringPtr("Group 1"), []NestedComponentModel{
				createNestedComponent("uptime/monitor", int64Ptr(101)),
			}),
		})

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)
		require.True(t, resp.Diagnostics.HasError())

		found := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Summary() == "Duplicate Monitor" {
				found = true
			}
		}
		require.True(t, found)
	})

	t.Run("duplicate monitor within the same group", func(t *testing.T) {
		planData := validStatusPageModel(t)
		planData.Components = createComponentsList(t, []ComponentModel{
			createComponent(t, "uptime/group", nil, stringPtr("Group 1"), []NestedComponentModel{
				createNestedComponent("uptime/monitor", int64Ptr(101)),
				createNestedComponent("uptime/monitor", int64Ptr(101)),
			}),
		})

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)
		require.True(t, resp.Diagnostics.HasError())

		found := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Summary() == "Duplicate Monitor" {
				found = true
			}
		}
		require.True(t, found)
	})

	t.Run("duplicate monitor across multiple groups", func(t *testing.T) {
		planData := validStatusPageModel(t)
		planData.Components = createComponentsList(t, []ComponentModel{
			createComponent(t, "uptime/group", nil, stringPtr("Group 1"), []NestedComponentModel{
				createNestedComponent("uptime/monitor", int64Ptr(101)),
			}),
			createComponent(t, "uptime/group", nil, stringPtr("Group 2"), []NestedComponentModel{
				createNestedComponent("uptime/monitor", int64Ptr(101)),
			}),
		})

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)
		require.True(t, resp.Diagnostics.HasError())

		found := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Summary() == "Duplicate Monitor" {
				found = true
			}
		}
		require.True(t, found)
	})
}
