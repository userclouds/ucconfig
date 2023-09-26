package resourcetypes

import (
	"context"

	"userclouds.com/idp"
	"userclouds.com/idp/userstore"
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
	// OmitAttributes is a list of resource attributes that should be omitted from manifest
	// generation, e.g. if there are superfluous details returned in ListWhateverObject API calls
	// that we don't need to include
	OmitAttributes []string
}

func getColumnRetentions(ctx context.Context, client *idp.Client, dt userstore.DataLifeCycleState) ([]interface{}, error) {
	response, err := client.ListColumns(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	out := []interface{}{}
	for _, column := range response.Data {
		retentionResponse, err := client.GetColumnRetentionDurations(ctx, column.ID, dt)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		for _, retention := range retentionResponse.RetentionDurations {
			if retention.UseDefault {
				// Skip default retentions inherited from elsewhere
				continue
			}
			out = append(out, retention)
		}
	}
	return out, nil
}

var omitRetentionAttributes = []string{
	// The default is computed from tenant/purpose retention settings (i.e. subject to change, would
	// create TF drift) and only used display in the console UI.
	"default_duration",
	// purpose_id and purpose_name should really be a ResourceID instead. In general, we try to
	// exclude names to avoid creating drift if the referenced resources change
	"purpose_name",
	// This is like an etag, from VersionBaseModel. We can't include it in configuration or else we
	// would have drift after applying every change
	"version",
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
		TerraformTypeSuffix: "userstore_column_pre_delete_retention_duration",
		ListResources: func(ctx context.Context, client *idp.Client) ([]interface{}, error) {
			return getColumnRetentions(ctx, client, userstore.DataLifeCycleStatePreDelete)
		},
		References: map[string]string{
			"column_id":  "userstore_column",
			"purpose_id": "userstore_purpose",
		},
		OmitAttributes: omitRetentionAttributes,
	},
	{
		TerraformTypeSuffix: "userstore_column_post_delete_retention_duration",
		ListResources: func(ctx context.Context, client *idp.Client) ([]interface{}, error) {
			return getColumnRetentions(ctx, client, userstore.DataLifeCycleStatePostDelete)
		},
		References: map[string]string{
			"column_id":  "userstore_column",
			"purpose_id": "userstore_purpose",
		},
		OmitAttributes: omitRetentionAttributes,
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
