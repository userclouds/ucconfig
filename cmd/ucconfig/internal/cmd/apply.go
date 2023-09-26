package cmd

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"userclouds.com/cmd/ucconfig/internal/genconfig"
	"userclouds.com/cmd/ucconfig/internal/liveresource"
	"userclouds.com/cmd/ucconfig/internal/manifest"
	"userclouds.com/cmd/ucconfig/internal/tfstate"
	"userclouds.com/idp"
	"userclouds.com/infra/uclog"
)

func genTerraform(ctx context.Context, mfest *manifest.Manifest, envName string, resources *[]liveresource.Resource, tfDir string) {
	tfText, err := genconfig.GenConfig(mfest, envName, resources)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to generate Terraform config: %v", err)
	}

	err = os.WriteFile(tfDir+"/main.tf", []byte(tfText), 0644)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to write generated Terraform config: %v", err)
	}

	// We also need to generate Terraform state for the existing resources, so that Terraform
	// doesn't try to create resources that already exist, and so that it will delete resources that
	// exist but are no longer in the manifest.
	state, err := tfstate.CreateState(resources)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to generate Terraform state: %v", err)
	}
	stateBytes, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		uclog.Fatalf(ctx, "Failed to marshal Terraform state: %v", err)
	}
	err = os.WriteFile(tfDir+"/terraform.tfstate", stateBytes, 0644)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to write Terraform state: %v", err)
	}
}

// Apply implements a "ucconfig apply" subcommand that applies a manifest.
func Apply(ctx context.Context, idpClient *idp.Client, manifestPath string, envName string) {
	uclog.Infof(ctx, "Reading manifest from %s...", manifestPath)
	manifestText, err := os.ReadFile(manifestPath)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to read manifest file: %v", err)
	}
	mfest := manifest.Manifest{}
	switch filepath.Ext(manifestPath) {
	case ".json":
		if err := json.Unmarshal(manifestText, &mfest); err != nil {
			uclog.Fatalf(ctx, "Failed to decode JSON: %v", err)
		}
	case ".yaml":
		if err := yaml.Unmarshal(manifestText, &mfest); err != nil {
			uclog.Fatalf(ctx, "Failed to decode YAML: %v", err)
		}
	default:
		uclog.Fatalf(ctx, "Manifest path must have .json or .yaml extension")
	}
	if err := mfest.Validate(envName); err != nil {
		uclog.Fatalf(ctx, "Failed to validate manifest: %v", err)
	}

	uclog.Infof(ctx, "Fetching live resources...")
	resources, err := liveresource.GetLiveResources(ctx, idpClient)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to fetch live resources: %v", err)
	}
	err = mfest.MatchLiveResources(ctx, &resources, envName)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to match manifest entries to live resources: %v", err)
	}

	uclog.Infof(ctx, "Generating Terraform...")
	dname, err := os.MkdirTemp("", "ucconfig-terraform")
	if err != nil {
		uclog.Fatalf(ctx, "Failed to create temporary directory: %v", err)
	}
	uclog.Infof(ctx, "Terraform files will be generated in %s", dname)
	genTerraform(ctx, &mfest, envName, &resources, dname)

	uclog.Infof(ctx, "Running terraform apply...")
	cmd := exec.Command("terraform", "apply")
	cmd.Dir = dname
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		uclog.Fatalf(ctx, "Failed to run terraform apply: %v", err)
	}

	// TODO: write manifest with updated matched resource<->manifest ID mappings
}
