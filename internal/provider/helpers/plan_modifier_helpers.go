package helpers

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// trimStringModifier trims whitespace from string attributes during planning.
type trimStringModifier struct{}

func (m trimStringModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	trimmed := strings.TrimSpace(req.PlanValue.ValueString())
	resp.PlanValue = types.StringValue(trimmed)
}

func (m trimStringModifier) Description(ctx context.Context) string {
	return "Trims whitespace from string attributes during planning to prevent state changes when the API trims strings."
}

func (m trimStringModifier) MarkdownDescription(ctx context.Context) string {
	return "Trims whitespace from string attributes during planning to prevent state changes when the API trims strings."
}

// TrimString returns a plan modifier that trims whitespace from string attributes.
func TrimString() planmodifier.String {
	return trimStringModifier{}
}
