package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"terraform-provider-phare/internal/client"
	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// integrationSecretNote is appended to the description of the settings the API never returns
const integrationSecretNote = " The API never returns this value, it is kept from the Terraform state and changes made outside of Terraform are not detected."

// IntegrationBaseModel represents the common fields for all integration types
// This is embedded in the model of every integration resource
type IntegrationBaseModel struct {
	CreatedAt types.String `tfsdk:"created_at"`
	Id        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Paused    types.Bool   `tfsdk:"paused"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

// IntegrationBaseResourceSchema defines the common schema attributes for all integration resources
func IntegrationBaseResourceSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"created_at": schema.StringAttribute{
			Computed:            true,
			Description:         "Date of creation",
			MarkdownDescription: "Date of creation",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"id": schema.Int64Attribute{
			Computed:            true,
			Description:         "Integration ID",
			MarkdownDescription: "Integration ID",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			Required:            true,
			Description:         "Integration name, must be unique across all the integrations of the organization",
			MarkdownDescription: "Integration name, must be unique across all the integrations of the organization",
			PlanModifiers: []planmodifier.String{
				helpers.TrimString(),
			},
			Validators: []validator.String{
				helpers.TrimmedLengthBetween(2, 50),
				helpers.SafeEntityName(),
			},
		},
		"paused": schema.BoolAttribute{
			Computed:            true,
			Description:         "Whether the integration is paused, integrations can only be paused from the control panel",
			MarkdownDescription: "Whether the integration is paused, integrations can only be paused from the control panel",
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"updated_at": schema.StringAttribute{
			Computed:            true,
			Description:         "Date of last update",
			MarkdownDescription: "Date of last update",
		},
	}
}

// integrationModel is implemented by the model of every integration resource
// to convert its app specific settings from and to their API representation
type integrationModel interface {
	// base returns the fields shared by all integrations
	base() *IntegrationBaseModel
	// buildSettings returns the API settings payload built from the model, prior is
	// the model of the prior state on update and nil on create
	buildSettings(ctx context.Context, prior integrationModel, diags *diag.Diagnostics) any
	// readSettings updates the model settings from the API response, secrets
	// are never returned by the API and are kept from the prior model
	readSettings(ctx context.Context, settings json.RawMessage, diags *diag.Diagnostics)
}

var (
	_ resource.Resource                = &integrationResource{}
	_ resource.ResourceWithConfigure   = &integrationResource{}
	_ resource.ResourceWithImportState = &integrationResource{}
)

// integrationResource is the resource implementation shared by all integration apps,
// each app provides its settings schema and model
type integrationResource struct {
	helpers.ResourceBase
	// app is the API app key, also used as the resource type suffix
	app string
	// displayName is the app name used in diagnostics
	displayName string
	// description is the app specific part of the resource description
	description string
	// docsPage is the page of the app in https://docs.phare.io/integrations
	docsPage           string
	settingsAttributes func() map[string]schema.Attribute
	newModel           func() integrationModel
}

func (r *integrationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_" + r.app
}

func (r *integrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := IntegrationBaseResourceSchema()
	attributes["settings"] = schema.SingleNestedAttribute{
		Required:            true,
		Description:         "Settings of the " + r.displayName + " integration",
		MarkdownDescription: "Settings of the " + r.displayName + " integration",
		Attributes:          r.settingsAttributes(),
	}

	description := r.description + " Integrations belong to the organization and are used by alert rules to send notifications."

	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: description + " [See docs](https://docs.phare.io/integrations/" + r.docsPage + ").",
		Attributes:          attributes,
	}
}

// mapAPIResponseToModel maps API response data to the resource model
func (r *integrationResource) mapAPIResponseToModel(ctx context.Context, apiResp *client.IntegrationResponse, model integrationModel, diags *diag.Diagnostics) {
	base := model.base()
	base.Id = types.Int64Value(apiResp.ID)
	base.Name = types.StringValue(apiResp.Name)
	base.Paused = types.BoolValue(apiResp.Paused)
	base.CreatedAt = types.StringValue(apiResp.CreatedAt)
	base.UpdatedAt = types.StringValue(apiResp.UpdatedAt)

	model.readSettings(ctx, apiResp.Settings, diags)
}

// buildAPIRequestFromModel builds an API request from the resource model, prior
// is the model of the prior state on update and nil on create
func (r *integrationResource) buildAPIRequestFromModel(ctx context.Context, model, prior integrationModel, diags *diag.Diagnostics) *client.IntegrationRequest {
	return &client.IntegrationRequest{
		Name:     model.base().Name.ValueString(),
		Settings: model.buildSettings(ctx, prior, diags),
	}
}

func (r *integrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := r.newModel()

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build API request from plan
	apiReq := r.buildAPIRequestFromModel(ctx, plan, nil, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to create integration
	apiResp, err := r.GetClient().CreateIntegration(ctx, r.app, apiReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating integration",
			"Could not create "+r.displayName+" integration: "+err.Error(),
		)
		return
	}

	// Map API response to state
	r.mapAPIResponseToModel(ctx, apiResp, plan, &resp.Diagnostics)

	// Save data into Terraform state even when the response could not be mapped, the integration
	// exists and Terraform marks it as tainted to replace it instead of leaving it behind
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *integrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	state := r.newModel()

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to read integration
	id := state.base().Id.ValueInt64()
	apiResp, err := r.GetClient().GetIntegration(ctx, r.app, id)
	if err != nil {
		if client.IsNotFoundError(err) {
			// Resource deleted outside Terraform
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading integration",
			fmt.Sprintf("Could not read %s integration ID %d: %s", r.displayName, id, err),
		)
		return
	}

	// Update state from API response
	r.mapAPIResponseToModel(ctx, apiResp, state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *integrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	plan := r.newModel()
	state := r.newModel()

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read current state to get the ID
	resp.Diagnostics.Append(req.State.Get(ctx, state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build API request from plan
	apiReq := r.buildAPIRequestFromModel(ctx, plan, state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to update integration using ID from current state
	id := state.base().Id.ValueInt64()
	apiResp, err := r.GetClient().UpdateIntegration(ctx, r.app, id, apiReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating integration",
			fmt.Sprintf("Could not update %s integration ID %d: %s", r.displayName, id, err),
		)
		return
	}

	// Update state from API response
	r.mapAPIResponseToModel(ctx, apiResp, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *integrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	state := r.newModel()

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to delete integration
	id := state.base().Id.ValueInt64()
	err := r.GetClient().DeleteIntegration(ctx, r.app, id)
	if err != nil && !client.IsNotFoundError(err) {
		detail := fmt.Sprintf("Could not delete %s integration ID %d: %s", r.displayName, id, err)

		// Integrations used by alert rules can not be deleted
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict {
			detail += "\n\nDelete or update the alert rules using this integration first."
		}

		resp.Diagnostics.AddError("Error deleting integration", detail)
	}
}

func (r *integrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Integrations belong to the organization, the import ID is the integration ID
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Integration ID must be a valid integer: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// decodeIntegrationSettings decodes the app specific settings of an API response into target
func decodeIntegrationSettings(settings json.RawMessage, target any, diags *diag.Diagnostics) bool {
	if len(settings) == 0 {
		return true
	}

	if err := json.Unmarshal(settings, target); err != nil {
		diags.AddError(
			"Error decoding integration settings",
			"Could not decode the integration settings returned by the API: "+err.Error(),
		)
		return false
	}

	return true
}
