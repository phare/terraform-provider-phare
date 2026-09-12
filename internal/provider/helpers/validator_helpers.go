package helpers

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type trimmedLengthBetweenValidator struct {
	min int
	max int
}

func (v trimmedLengthBetweenValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("string character length after trimming whitespace must be between %d and %d", v.min, v.max)
}

func (v trimmedLengthBetweenValidator) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("string character length after trimming whitespace must be between %d and %d", v.min, v.max)
}

func (v trimmedLengthBetweenValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	trimmed := strings.TrimSpace(req.ConfigValue.ValueString())
	l := utf8.RuneCountInString(trimmed)
	if l < v.min || l > v.max {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value Length",
			fmt.Sprintf("string character length after trimming whitespace must be between %d and %d, got: %d", v.min, v.max, l),
		)
	}
}

// TrimmedLengthBetween returns a validator that checks string character length after trimming whitespace.
func TrimmedLengthBetween(min, max int) validator.String {
	return trimmedLengthBetweenValidator{min: min, max: max}
}

type trimmedLengthAtMostValidator struct {
	max int
}

func (v trimmedLengthAtMostValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("string character length after trimming whitespace must be at most %d", v.max)
}

func (v trimmedLengthAtMostValidator) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("string character length after trimming whitespace must be at most %d", v.max)
}

func (v trimmedLengthAtMostValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	trimmed := strings.TrimSpace(req.ConfigValue.ValueString())
	l := utf8.RuneCountInString(trimmed)
	if l > v.max {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value Length",
			fmt.Sprintf("string character length after trimming whitespace must be at most %d, got: %d", v.max, l),
		)
	}
}

// TrimmedLengthAtMost returns a validator that checks maximum string character length after trimming whitespace.
func TrimmedLengthAtMost(max int) validator.String {
	return trimmedLengthAtMostValidator{max: max}
}
