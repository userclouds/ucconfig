package manifest

import (
	"context"
	"encoding/json"
	"testing"

	"userclouds.com/cmd/ucconfig/internal/liveresource"
	"userclouds.com/cmd/ucconfig/internal/resourcetypes"
	"userclouds.com/infra/assert"
)

func TestGenerateNewManifest(t *testing.T) {
	ctx := context.Background()
	resources := []liveresource.Resource{
		// Normal resource, should show up in the manifest:
		{
			TerraformTypeSuffix: "userstore_column",
			ResourceUUID:        "fe20fd48-a006-4ad8-9208-4aad540d8794",
			IsSystem:            false,
			Attributes: map[string]any{
				"name":            "col1",
				"extra_attribute": "extra1",
				"bool_attribute":  true,
				"int_attribute":   7,
			},
		},
		// System resource, should be omitted from the manifest:
		{
			TerraformTypeSuffix: "userstore_column",
			ResourceUUID:        "c860a6d7-c632-4f81-8f5f-597290a9f437",
			IsSystem:            true,
			Attributes: map[string]any{
				"name": "col2",
			},
		},
	}
	mfest, err := generateFromLiveResources(ctx, &resources, "prod")
	assert.NoErr(t, err)
	mfestJSON, err := json.MarshalIndent(mfest, "", "\t")
	assert.NoErr(t, err)
	assert.Equal(t, string(mfestJSON), `{
	"resources": [
		{
			"uc_terraform_type": "userstore_column",
			"manifest_id": "userstore_column_col1",
			"resource_uuids": {
				"__DEFAULT": "fe20fd48-a006-4ad8-9208-4aad540d8794",
				"prod": "fe20fd48-a006-4ad8-9208-4aad540d8794"
			},
			"attributes": {
				"bool_attribute": true,
				"extra_attribute": "extra1",
				"int_attribute": 7,
				"name": "col1"
			}
		}
	]
}`, assert.Diff())
}

func TestRewriteManifestReferencesUCManifestID(t *testing.T) {
	// Validate generating @UC_MANIFEST_ID() references
	val := []any{
		map[string]any{
			"column":      "12b3f133-4ad1-4f11-9d7d-313eb7cb95fa",
			"transformer": "c0b5b2a1-0b1f-4b9f-8b1a-1b1f4b9f8b1a",
		},
	}
	mfest := Manifest{
		Resources: []Resource{{
			TerraformTypeSuffix: "userstore_column",
			ManifestID:          "examplecol",
			ResourceUUIDs: map[string]string{
				"prod": "12b3f133-4ad1-4f11-9d7d-313eb7cb95fa",
			},
		}, {
			TerraformTypeSuffix: "transformer",
			ManifestID:          "exampletransformer",
			ResourceUUIDs: map[string]string{
				"prod": "c0b5b2a1-0b1f-4b9f-8b1a-1b1f4b9f8b1a",
			},
		}},
	}
	out, err := rewriteManifestReferences(val, "columns", *resourcetypes.GetByTerraformTypeSuffix("userstore_accessor"), &mfest, "prod", &[]liveresource.Resource{})
	assert.NoErr(t, err)
	assert.Equal(t, out.([]any)[0].(map[string]any)["column"], `@UC_MANIFEST_ID("examplecol").id`)
	assert.Equal(t, out.([]any)[0].(map[string]any)["transformer"], `@UC_MANIFEST_ID("exampletransformer").id`)
}

func TestRewriteManifestReferencesUCSystemObject(t *testing.T) {
	// Validate generating @UC_SYSTEM_OBJECT() references
	val := []any{
		map[string]any{
			"column":      "12b3f133-4ad1-4f11-9d7d-313eb7cb95fa",
			"transformer": "c0b5b2a1-0b1f-4b9f-8b1a-1b1f4b9f8b1a",
		},
	}
	systemResources := []liveresource.Resource{{
		TerraformTypeSuffix: "userstore_column",
		ManifestID:          "examplecol",
		ResourceUUID:        "12b3f133-4ad1-4f11-9d7d-313eb7cb95fa",
		IsSystem:            true,
		Attributes: map[string]interface{}{
			"name": "example",
		},
	}, {
		TerraformTypeSuffix: "transformer",
		ManifestID:          "exampletransformer",
		ResourceUUID:        "c0b5b2a1-0b1f-4b9f-8b1a-1b1f4b9f8b1a",
		IsSystem:            true,
		Attributes: map[string]interface{}{
			"name": "tform",
		},
	}}
	out, err := rewriteManifestReferences(val, "columns", *resourcetypes.GetByTerraformTypeSuffix("userstore_accessor"), &Manifest{}, "prod", &systemResources)
	assert.NoErr(t, err)
	assert.Equal(t, out.([]any)[0].(map[string]any)["column"], `@UC_SYSTEM_OBJECT("userstore_column", "example")`)
	assert.Equal(t, out.([]any)[0].(map[string]any)["transformer"], `@UC_SYSTEM_OBJECT("transformer", "tform")`)
}
