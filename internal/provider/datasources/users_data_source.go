package datasources

import (
	"context"
	"strings"

	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &usersDataSource{}
	_ datasource.DataSourceWithConfigure = &usersDataSource{}
)

// NewUsersDataSource returns a new users data source.
func NewUsersDataSource() datasource.DataSource {
	return &usersDataSource{}
}

// usersDataSource is the data source implementation.
type usersDataSource struct {
	helpers.DataSourceBase
}

// usersDataSourceModel describes the data source data model.
type usersDataSourceModel struct {
	Emails types.List  `tfsdk:"emails"`
	Users  []userModel `tfsdk:"users"`
}

func (d *usersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

func (d *usersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a list of users, optionally filtered by email addresses.",
		Attributes: map[string]schema.Attribute{
			"emails": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "List of email addresses to filter users. If not specified, returns all users (first page, up to 100 users).",
			},
			"users": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of users",
				NestedObject: schema.NestedAttributeObject{
					Attributes: userSchemaAttributes(),
				},
			},
		},
	}
}

func (d *usersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config usersDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get emails filter (may be empty string if not set)
	emails := ""
	if !config.Emails.IsNull() && len(config.Emails.Elements()) > 0 {
		var emailList []string
		resp.Diagnostics.Append(config.Emails.ElementsAs(ctx, &emailList, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		emails = strings.Join(emailList, ",")
	}

	// Fetch users from API (first page, 100 items)
	users, err := d.GetClient().ListUsers(ctx, emails, 1, 100)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Users",
			err.Error(),
		)
		return
	}

	// Map API response to state
	config.Users = make([]userModel, 0, len(users))
	for _, user := range users {
		userState := userModel{
			ID:        types.Int64Value(user.ID),
			Role:      types.StringValue(user.Role),
			Email:     types.StringValue(user.Email),
			CreatedAt: types.StringValue(user.CreatedAt),
			UpdatedAt: types.StringValue(user.UpdatedAt),
		}
		config.Users = append(config.Users, userState)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
