package helpers

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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

// tagValuePattern matches a valid resource tag: Unicode letters, numbers, and . _ : - characters.
var tagValuePattern = regexp.MustCompile(`^[\p{L}\p{N}._:-]+$`)

// TagListValidators returns the shared validators for the tags list attribute.
func TagListValidators() []validator.List {
	return []validator.List{
		listvalidator.SizeAtMost(20),
		listvalidator.UniqueValues(),
		listvalidator.ValueStringsAre(
			TrimmedLengthBetween(1, 100),
			stringvalidator.RegexMatches(tagValuePattern, "must contain only Unicode letters, numbers, and . _ : - characters"),
		),
	}
}

// TagSetValidators returns the shared validators for the tags set attribute.
func TagSetValidators() []validator.Set {
	return []validator.Set{
		setvalidator.SizeAtMost(20),
		setvalidator.ValueStringsAre(
			TrimmedLengthBetween(1, 100),
			stringvalidator.RegexMatches(tagValuePattern, "must contain only Unicode letters, numbers, and . _ : - characters"),
		),
	}
}
