package cmd

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"userclouds.com/cmd/ucconfig/internal/liveresource"
	"userclouds.com/cmd/ucconfig/internal/manifest"
	"userclouds.com/cmd/ucconfig/internal/tfconfig"
	"userclouds.com/cmd/ucconfig/internal/tfstate"
	"userclouds.com/idp"
	"userclouds.com/infra/uclog"
)

func writeTerraformRC(ctx context.Context, rcPath string, tfProviderDevDirPath string) {
	abspath, err := filepath.Abs(tfProviderDevDirPath)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to get absolute path to terraform-provider-userclouds dev directory %v: %v", tfProviderDevDirPath, err)
	}
	config := `
		provider_installation {
			dev_overrides {
				"userclouds/userclouds" = "` + abspath + `"
			}
			direct {}
		}
	`
	err = os.WriteFile(rcPath, []byte(config), 0644)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to write %v: %v", rcPath, err)
	}
}

func genTerraform(ctx context.Context, mfestPath string, mfest *manifest.Manifest, fqtn string, resources *[]liveresource.Resource, tfDir string, tfProviderVersionConstraint string) {
	if tfProviderVersionConstraint == "" {
		// Require at least v0.1.8 for support for column search indexing
		tfProviderVersionConstraint = ">= 0.1.8"
	}
	tfText, err := tfconfig.GenConfig(&tfconfig.GenerationContext{
		ManifestFilePath:            mfestPath,
		Manifest:                    mfest,
		FQTN:                        fqtn,
		LiveResources:               resources,
		TFProviderVersionConstraint: tfProviderVersionConstraint,
	})
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
func Apply(ctx context.Context, dryRun, autoApprove bool, idpClient *idp.Client, fqtn string, tenantURL string, clientID string, clientSecret string, manifestPath string, tfProviderVersionConstraint string, tfProviderDevDirPath string) {
	if dryRun && autoApprove {
		uclog.Fatalf(ctx, "dry run and auto approve flags are mutually exclusive")
	}

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
	if err := mfest.Validate(fqtn); err != nil {
		uclog.Fatalf(ctx, "Failed to validate manifest: %v", err)
	}

	uclog.Infof(ctx, "Fetching live resources...")
	resources, err := liveresource.GetLiveResources(ctx, idpClient)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to fetch live resources: %v", err)
	}
	err = mfest.MatchLiveResources(ctx, &resources, fqtn)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to match manifest entries to live resources: %v", err)
	}

	uclog.Infof(ctx, "Generating Terraform...")
	dname, err := os.MkdirTemp("", "ucconfig-terraform")
	if err != nil {
		uclog.Fatalf(ctx, "Failed to create temporary directory: %v", err)
	}
	uclog.Infof(ctx, "Terraform files will be generated in %s", dname)
	genTerraform(ctx, manifestPath, &mfest, fqtn, &resources, dname, tfProviderVersionConstraint)

	env := os.Environ()
	if tfProviderDevDirPath != "" {
		terraformRCPath := dname + "/.terraformrc"
		writeTerraformRC(ctx, terraformRCPath, tfProviderDevDirPath)
		uclog.Infof(ctx, "Setting TF_CLI_CONFIG_FILE=%v to enable usage of local dev build of UC TF provider", terraformRCPath)
		env = append(env, "TF_CLI_CONFIG_FILE="+terraformRCPath)
	}

	uclog.Infof(ctx, "Running terraform init...")
	cmd := exec.Command("terraform", "init")
	cmd.Dir = dname
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env
	err = cmd.Run()
	if err != nil {
		uclog.Fatalf(ctx, "Failed to run terraform init: %v\nGenerated terraform files are in %s", err, dname)
	}

	cmdArgs := make([]string, 0, 3)

	if dryRun {
		cmdArgs = append(cmdArgs, "plan")

	} else {
		cmdArgs = append(cmdArgs, "apply")
		if autoApprove {
			cmdArgs = append(cmdArgs, "-auto-approve")
		}
	}
	uclog.Infof(ctx, "Running terraform %s...", strings.Join(cmdArgs, " "))
	cmd = exec.Command("terraform", cmdArgs...)
	cmd.Dir = dname
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env
	cmd.Env = append(cmd.Env, "USERCLOUDS_TENANT_URL="+tenantURL)
	cmd.Env = append(cmd.Env, "USERCLOUDS_CLIENT_ID="+clientID)
	cmd.Env = append(cmd.Env, "USERCLOUDS_CLIENT_SECRET="+clientSecret)
	err = cmd.Run()
	if err != nil {
		uclog.Fatalf(ctx, "Failed to run terraform apply: %v\nGenerated terraform files are in %s", err, dname)
	}

	// TODO: it could be a nice feature to prompt the user to ask whether to
	// write the manifest with updated matched resource<->manifest ID mappings.
	// However, if they have added comments to the manifest or made any
	// formatting customizations, those would get overwritten, so I have been
	// waiting to see whether anyone asks for this.
}
