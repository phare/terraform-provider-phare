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
		{"unicode accented characters within limit", types.StringValue("  café  "), false},
		{"unicode CJK characters within limit", types.StringValue("  日本語東京  "), false}, // 5 runes, 15 bytes
		{"unicode emoji within limit", types.StringValue("  🚀🎉✨🔥  "), false},           // 4 runes, 16 bytes
		{"below min length after trim", types.StringValue(" a "), true},
		{"whitespace only trimmed to 0", types.StringValue("   "), true},
		{"empty string", types.StringValue(""), true},
		{"exceeds max length", types.StringValue(strings.Repeat("a", 11)), true},
		{"exceeds max length after trim", types.StringValue("  " + strings.Repeat("a", 11) + "  "), true},
		{"unicode exceeds max runes", types.StringValue("日本語日本語日本語日本語日本語"), true}, // 11 runes
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
		{"unicode CJK within limit", types.StringValue("  日本語東京  "), false},  // 5 runes, 15 bytes
		{"unicode emoji within limit", types.StringValue("  🚀🎉✨🔥  "), false}, // 4 runes, 16 bytes
		{"exceeds max length", types.StringValue(strings.Repeat("a", 11)), true},
		{"exceeds max length after trim", types.StringValue("  " + strings.Repeat("a", 11) + "  "), true},
		{"unicode exceeds max runes", types.StringValue("日本語日本語日本語日本語日本語"), true}, // 11 runes
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

func TestTagListValidators(t *testing.T) {
	validators := TagListValidators()
	require.NotEmpty(t, validators)

	newList := func(t *testing.T, values ...string) types.List {
		l, diags := types.ListValueFrom(context.Background(), types.StringType, values)
		require.False(t, diags.HasError())
		return l
	}

	testCases := []struct {
		name        string
		val         types.List
		expectError bool
	}{
		{"null is skipped", types.ListNull(types.StringType), false},
		{"empty list", newList(t), false},
		{"single tag", newList(t, "environment:production"), false},
		{"multiple tags", newList(t, "environment:production", "team:backend"), false},
		{"unicode letters and numbers", newList(t, "café-1", "日本語_2", "ก.3"), false},
		{"exactly 20 tags", newList(t, "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t"), false},
		{"exceeds 20 tags", newList(t, "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u"), true},
		{"duplicate tags", newList(t, "same", "same"), true},
		{"tag with space", newList(t, "prod env"), true},
		{"tag with comma", newList(t, "a,b"), true},
		{"tag with slash", newList(t, "a/b"), true},
		{"empty tag", newList(t, ""), true},
		{"exceeds 100 characters", newList(t, strings.Repeat("a", 101)), true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.ListRequest{
				Path:        path.Root("tags"),
				ConfigValue: tc.val,
			}
			resp := &validator.ListResponse{}
			for _, v := range validators {
				v.ValidateList(context.Background(), req, resp)
			}

			if tc.expectError {
				require.True(t, resp.Diagnostics.HasError())
			} else {
				require.False(t, resp.Diagnostics.HasError())
			}
		})
	}
}

func TestTagSetValidators(t *testing.T) {
	validators := TagSetValidators()
	require.NotEmpty(t, validators)

	newSet := func(t *testing.T, values ...string) types.Set {
		s, diags := types.SetValueFrom(context.Background(), types.StringType, values)
		require.False(t, diags.HasError())
		return s
	}

	testCases := []struct {
		name        string
		val         types.Set
		expectError bool
	}{
		{"null is skipped", types.SetNull(types.StringType), false},
		{"empty set", newSet(t), false},
		{"single tag", newSet(t, "environment:production"), false},
		{"multiple tags", newSet(t, "environment:production", "team:backend"), false},
		{"unicode letters and numbers", newSet(t, "café-1", "日本語_2", "ก.3"), false},
		{"exactly 20 tags", newSet(t, "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t"), false},
		{"exceeds 20 tags", newSet(t, "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u"), true},
		{"tag with space", newSet(t, "prod env"), true},
		{"tag with comma", newSet(t, "a,b"), true},
		{"tag with slash", newSet(t, "a/b"), true},
		{"empty tag", newSet(t, ""), true},
		{"exceeds 100 characters", newSet(t, strings.Repeat("a", 101)), true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.SetRequest{
				Path:        path.Root("tags"),
				ConfigValue: tc.val,
			}
			resp := &validator.SetResponse{}
			for _, v := range validators {
				v.ValidateSet(context.Background(), req, resp)
			}

			if tc.expectError {
				require.True(t, resp.Diagnostics.HasError())
			} else {
				require.False(t, resp.Diagnostics.HasError())
			}
		})
	}
}
