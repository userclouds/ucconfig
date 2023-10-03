package tfstate

import (
	"encoding/json"
	"testing"

	"userclouds.com/cmd/ucconfig/internal/liveresource"
	"userclouds.com/infra/assert"
)

func TestCreateState(t *testing.T) {
	resources := []liveresource.Resource{
		{
			TerraformTypeSuffix: "userstore_column",
			ManifestID:          "entry1",
			ResourceUUID:        "fe20fd48-a006-4ad8-9208-4aad540d8794",
		},
		{
			TerraformTypeSuffix: "userstore_column",
			ManifestID:          "entry2",
			ResourceUUID:        "c860a6d7-c632-4f81-8f5f-597290a9f437",
		},
	}
	state, err := CreateState(&resources)
	assert.NoErr(t, err)
	// Set non-random lineage
	state.Lineage = "random-uuid"
	marshalled, err := json.MarshalIndent(state, "", "  ")
	assert.NoErr(t, err)
	assert.Equal(t, string(marshalled), `{
  "version": 4,
  "terraform_version": "1.5.3",
  "serial": 1,
  "lineage": "random-uuid",
  "outputs": {},
  "resources": [
    {
      "mode": "managed",
      "type": "userclouds_userstore_column",
      "name": "manifestid-entry1",
      "provider": "provider[\"registry.terraform.io/userclouds/userclouds\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "id": "fe20fd48-a006-4ad8-9208-4aad540d8794"
          },
          "sensitive_attributes": [],
          "dependencies": null
        }
      ]
    },
    {
      "mode": "managed",
      "type": "userclouds_userstore_column",
      "name": "manifestid-entry2",
      "provider": "provider[\"registry.terraform.io/userclouds/userclouds\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "id": "c860a6d7-c632-4f81-8f5f-597290a9f437"
          },
          "sensitive_attributes": [],
          "dependencies": null
        }
      ]
    }
  ],
  "check_results": []
}`)
}
