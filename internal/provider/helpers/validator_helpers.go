package helpers

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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

// safeEntityNameAllowedChars lists the symbols allowed in entity names on top of letters, numbers and spaces.
const safeEntityNameAllowedChars = ". + ' | ! - _ — & : # ( ) [ ] ,"

var (
	// safeEntityNamePattern matches Unicode letters, numbers, spaces and the allowed symbols.
	safeEntityNamePattern = regexp.MustCompile(`^[\p{L}\p{N} .+'|!\-_—&:#()\[\],]+$`)
	// rightToLeftPattern matches characters of the scripts written from right to left.
	rightToLeftPattern = regexp.MustCompile(`[\p{Hebrew}\p{Arabic}\p{Syriac}\p{Thaana}\p{Nko}\p{Samaritan}\p{Mandaic}\p{Adlam}]`)
)

type safeEntityNameValidator struct{}

func (v safeEntityNameValidator) Description(ctx context.Context) string {
	return "string can only contain letters, numbers, spaces and: " + safeEntityNameAllowedChars
}

func (v safeEntityNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v safeEntityNameValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if rightToLeftPattern.MatchString(value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value",
			"right-to-left text is not supported",
		)
		return
	}

	if !safeEntityNamePattern.MatchString(value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value",
			"can only contain letters, numbers, spaces and: "+safeEntityNameAllowedChars,
		)
	}
}

// SafeEntityName returns a validator that checks a name only contains the characters accepted by the API
// (letters, numbers, spaces and a restricted set of symbols, no emojis or right-to-left text).
func SafeEntityName() validator.String {
	return safeEntityNameValidator{}
}

var (
	// hostnameLabelPattern matches a single RFC 1123 hostname label.
	hostnameLabelPattern = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`)
	// hostnameTLDPattern matches the top-level domain the API requires.
	hostnameTLDPattern = regexp.MustCompile(`\.\w{2}`)
)

type hostnameValidator struct{}

func (v hostnameValidator) Description(ctx context.Context) string {
	return "string must be a public hostname without scheme, port or path, e.g. ntfy.sh"
}

func (v hostnameValidator) MarkdownDescription(ctx context.Context) string {
	return "string must be a public hostname without scheme, port or path, e.g. `ntfy.sh`"
}

func (v hostnameValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	switch {
	case net.ParseIP(value) != nil:
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Hostname", "must be a hostname, not an IP address")
	case !isValidHostname(value):
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Hostname", fmt.Sprintf("%q must be a public hostname without scheme, port or path, e.g. ntfy.sh", value))
	}
}

// isValidHostname checks that the value is made of RFC 1123 labels and has a top-level domain,
// which also rejects local names like localhost.
func isValidHostname(value string) bool {
	if len(value) > 253 || !hostnameTLDPattern.MatchString(value) {
		return false
	}

	for _, label := range strings.Split(value, ".") {
		if !hostnameLabelPattern.MatchString(label) {
			return false
		}
	}

	return true
}

// Hostname returns a validator that checks a string is a public hostname, without scheme, port or path.
func Hostname() validator.String {
	return hostnameValidator{}
}

type hasPrefixValidator struct {
	prefix string
}

func (v hasPrefixValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("string must start with %s", v.prefix)
}

func (v hasPrefixValidator) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("string must start with `%s`", v.prefix)
}

func (v hasPrefixValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if !strings.HasPrefix(req.ConfigValue.ValueString(), v.prefix) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value",
			fmt.Sprintf("must start with %s", v.prefix),
		)
	}
}

// HasPrefix returns a validator that checks a string starts with the given prefix.
// Unlike stringvalidator.RegexMatches, the error does not include the value, so it can be used on sensitive attributes.
func HasPrefix(prefix string) validator.String {
	return hasPrefixValidator{prefix: prefix}
}

type uniqueAttributeIgnoreCaseValidator struct {
	attribute string
}

func (v uniqueAttributeIgnoreCaseValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("%s must be unique across the list, ignoring case", v.attribute)
}

func (v uniqueAttributeIgnoreCaseValidator) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("`%s` must be unique across the list, ignoring case", v.attribute)
}

func (v uniqueAttributeIgnoreCaseValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	seen := make(map[string]struct{})
	for i, element := range req.ConfigValue.Elements() {
		object, ok := element.(basetypes.ObjectValue)
		if !ok || object.IsNull() || object.IsUnknown() {
			continue
		}

		value, ok := object.Attributes()[v.attribute].(basetypes.StringValue)
		if !ok || value.IsNull() || value.IsUnknown() {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(value.ValueString()))
		if _, duplicate := seen[key]; duplicate {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtListIndex(i).AtName(v.attribute),
				"Duplicate Value",
				fmt.Sprintf("%q is used more than once, %s must be unique, ignoring case.", value.ValueString(), v.attribute),
			)
			continue
		}
		seen[key] = struct{}{}
	}
}

// UniqueAttributeIgnoreCase returns a validator that checks that a string attribute of the
// objects of a list is unique across the list, ignoring case and surrounding whitespace.
func UniqueAttributeIgnoreCase(attribute string) validator.List {
	return uniqueAttributeIgnoreCaseValidator{attribute: attribute}
}
