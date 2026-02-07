package datasources

import (
	"context"
	"fmt"

	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &userDataSource{}
	_ datasource.DataSourceWithConfigure = &userDataSource{}
)

// NewUserDataSource returns a new user data source.
func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

// userDataSource is the data source implementation.
type userDataSource struct {
	helpers.DataSourceBase
}

// userModel describes the common user data model used by both
// the single user and users list data sources.
type userModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Role      types.String `tfsdk:"role"`
	Email     types.String `tfsdk:"email"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

// userSchemaAttributes returns the common schema attributes for users
func userSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:    true,
			Description: "User ID",
		},
		"role": schema.StringAttribute{
			Computed:    true,
			Description: "User role (member or admin)",
		},
		"email": schema.StringAttribute{
			Computed:    true,
			Description: "User email address",
		},
		"created_at": schema.StringAttribute{
			Computed:    true,
			Description: "Creation timestamp",
		},
		"updated_at": schema.StringAttribute{
			Computed:    true,
			Description: "Last update timestamp",
		},
	}
}

// userDataSourceModel describes the data source data model.
type userDataSourceModel struct {
	userModel
}

func (d *userDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *userDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a single user by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Required:    true,
				Description: "The ID of the user to fetch",
			},
		},
	}

	// Add the common user attributes
	for key, attr := range userSchemaAttributes() {
		resp.Schema.Attributes[key] = attr
	}
}

func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config userDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch user from API
	userID := config.ID.ValueInt64()
	user, err := d.GetClient().GetUser(ctx, userID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read User",
			fmt.Sprintf("Could not read user ID %d: %s", userID, err.Error()),
		)
		return
	}

	// Map API response to state
	config.Role = types.StringValue(user.Role)
	config.Email = types.StringValue(user.Email)
	config.CreatedAt = types.StringValue(user.CreatedAt)
	config.UpdatedAt = types.StringValue(user.UpdatedAt)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
