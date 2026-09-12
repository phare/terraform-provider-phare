package helpers

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestTrimmedLengthBetween(t *testing.T) {
	v := TrimmedLengthBetween(2, 10)
	require.NotEmpty(t, v.Description(context.Background()))
	require.NotEmpty(t, v.MarkdownDescription(context.Background()))

	testCases := []struct {
		name        string
		val         types.String
		expectError bool
	}{
		{"null is skipped", types.StringNull(), false},
		{"unknown is skipped", types.StringUnknown(), false},
		{"exact min length", types.StringValue("ab"), false},
		{"exact max length", types.StringValue(strings.Repeat("a", 10)), false},
		{"within range", types.StringValue("hello"), false},
		{"within range with surrounding whitespace", types.StringValue("   hello   "), false},
		{"max length with surrounding whitespace", types.StringValue("   " + strings.Repeat("a", 10) + "   "), false},
		{"below min length after trim", types.StringValue(" a "), true},
		{"whitespace only trimmed to 0", types.StringValue("   "), true},
		{"empty string", types.StringValue(""), true},
		{"exceeds max length", types.StringValue(strings.Repeat("a", 11)), true},
		{"exceeds max length after trim", types.StringValue("  " + strings.Repeat("a", 11) + "  "), true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("test"),
				ConfigValue: tc.val,
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			if tc.expectError {
				require.True(t, resp.Diagnostics.HasError())
			} else {
				require.False(t, resp.Diagnostics.HasError())
			}
		})
	}
}

func TestTrimmedLengthAtMost(t *testing.T) {
	v := TrimmedLengthAtMost(10)
	require.NotEmpty(t, v.Description(context.Background()))
	require.NotEmpty(t, v.MarkdownDescription(context.Background()))

	testCases := []struct {
		name        string
		val         types.String
		expectError bool
	}{
		{"null is skipped", types.StringNull(), false},
		{"unknown is skipped", types.StringUnknown(), false},
		{"empty string", types.StringValue(""), false},
		{"whitespace only", types.StringValue("   "), false},
		{"exact max length", types.StringValue(strings.Repeat("a", 10)), false},
		{"max length with surrounding whitespace", types.StringValue("   " + strings.Repeat("a", 10) + "   "), false},
		{"exceeds max length", types.StringValue(strings.Repeat("a", 11)), true},
		{"exceeds max length after trim", types.StringValue("  " + strings.Repeat("a", 11) + "  "), true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("test"),
				ConfigValue: tc.val,
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			if tc.expectError {
				require.True(t, resp.Diagnostics.HasError())
			} else {
				require.False(t, resp.Diagnostics.HasError())
			}
		})
	}
}
