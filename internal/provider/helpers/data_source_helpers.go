package helpers

import (
	"context"
	"fmt"

	"terraform-provider-phare/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

// DataSourceBase provides common functionality for all data sources
type DataSourceBase struct {
	BaseConfig
}

// Configure implements datasource.DataSourceWithConfigure
func (d *DataSourceBase) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.BaseConfig.Configure(ctx, apiClient, &resp.Diagnostics)
}
