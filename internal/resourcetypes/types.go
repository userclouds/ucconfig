package resourcetypes

import (
	"context"

	"userclouds.com/idp"
	"userclouds.com/infra/ucerr"
)

// ResourceType represents a Terraform provider resource type.
type ResourceType struct {
	TerraformTypeSuffix string
	// The returned structs must have an ID field.
	ListResources func(ctx context.Context, client *idp.Client) ([]interface{}, error)
	// References maps attribute paths (e.g. "columns.column") to the terraform type suffix of the
	// resource UUIDs they reference
	References map[string]string
	// WriteAttributesExternally maps attribute paths (e.g.
	// "child_object.attr_with_long_value") to file extensions (e.g. ".js") for
	// attributes whose values should be stored in an external file, rather than
	// inline in a generated manifest
	WriteAttributesExternally map[string]string
	// OmitAttributes is a list of resource attributes that should be omitted from manifest
	// generation, e.g. if there are superfluous details returned in ListWhateverObject API calls
	// that we don't need to include
	OmitAttributes []string
}

// ResourceTypes lists the resource types supported by ucconfig.
var ResourceTypes = []ResourceType{
	{
		TerraformTypeSuffix: "userstore_column",
		ListResources: func(ctx context.Context, client *idp.Client) ([]interface{}, error) {
			response, err := client.ListColumns(ctx)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			out := []interface{}{}
			for _, column := range response.Data {
				out = append(out, column)
			}
			return out, nil
		},
	},
	{
		TerraformTypeSuffix: "userstore_accessor",
		ListResources: func(ctx context.Context, client *idp.Client) ([]interface{}, error) {
			response, err := client.ListAccessors(ctx)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			out := []interface{}{}
			for _, d := range response.Data {
				out = append(out, d)
			}
			return out, nil
		},
		References: map[string]string{
			"access_policy":       "access_policy",
			"columns.column":      "userstore_column",
			"columns.transformer": "transformer",
			"purposes":            "userstore_purpose",
		},
	},
	{
		TerraformTypeSuffix: "userstore_mutator",
		ListResources: func(ctx context.Context, client *idp.Client) ([]interface{}, error) {
			response, err := client.ListMutators(ctx)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			out := []interface{}{}
			for _, d := range response.Data {
				out = append(out, d)
			}
			return out, nil
		},
		References: map[string]string{
			"access_policy":     "access_policy",
			"columns.column":    "userstore_column",
			"columns.validator": "transformer",
		},
	},
	{
		TerraformTypeSuffix: "userstore_purpose",
		ListResources: func(ctx context.Context, client *idp.Client) ([]interface{}, error) {
			response, err := client.ListPurposes(ctx)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			out := []interface{}{}
			for _, d := range response.Data {
				out = append(out, d)
			}
			return out, nil
		},
	},
	{
		TerraformTypeSuffix: "access_policy",
		ListResources: func(ctx context.Context, client *idp.Client) ([]interface{}, error) {
			response, err := client.TokenizerClient.ListAccessPolicies(ctx, false)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			out := []interface{}{}
			for _, d := range response.Data {
				out = append(out, d)
			}
			return out, nil
		},
		References: map[string]string{
			"components.policy":   "access_policy",
			"components.template": "access_policy_template",
		},
	},
	{
		TerraformTypeSuffix: "access_policy_template",
		ListResources: func(ctx context.Context, client *idp.Client) ([]interface{}, error) {
			response, err := client.TokenizerClient.ListAccessPolicyTemplates(ctx, false)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			out := []interface{}{}
			for _, d := range response.Data {
				out = append(out, d)
			}
			return out, nil
		},
		WriteAttributesExternally: map[string]string{
			// The JS function should be stored separately to facilitate linting
			// and syntax highlighting
			"function": ".js",
		},
	},
	{
		TerraformTypeSuffix: "transformer",
		ListResources: func(ctx context.Context, client *idp.Client) ([]interface{}, error) {
			response, err := client.TokenizerClient.ListTransformers(ctx)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			out := []interface{}{}
			for _, d := range response.Data {
				out = append(out, d)
			}
			return out, nil
		},
		WriteAttributesExternally: map[string]string{
			// The JS function should be stored separately to facilitate linting
			// and syntax highlighting
			"function": ".js",
		},
	},
}

// GetByTerraformTypeSuffix returns the ResourceType with the given TerraformTypeSuffix.
func GetByTerraformTypeSuffix(s string) *ResourceType {
	for _, rt := range ResourceTypes {
		if rt.TerraformTypeSuffix == s {
			return &rt
		}
	}
	return nil
}

// ValidateTerraformTypeSuffix returns true if s is a supported Terraform type
// suffix.
func ValidateTerraformTypeSuffix(s string) bool {
	return GetByTerraformTypeSuffix(s) != nil
}
