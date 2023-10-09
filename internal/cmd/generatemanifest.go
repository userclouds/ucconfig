package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"userclouds.com/cmd/ucconfig/internal/manifest"
	"userclouds.com/idp"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// GenerateNewManifest implements a "ucconfig gen-manifest" subcommand that generates a new manifest.
func GenerateNewManifest(ctx context.Context, idpClient *idp.Client, fqtn string, manifestPath string) {
	uclog.Infof(ctx, "Generating new manifest from live resource state...")

	manifestBasename := filepath.Base(manifestPath)
	externValuesDirName := manifestBasename[:len(manifestBasename)-len(filepath.Ext(manifestBasename))] + "_values"
	externValuesDirPath, err := filepath.Abs(filepath.Dir(manifestPath) + "/" + externValuesDirName)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to get absolute path for storing attribute values externally: %v", ucerr.Wrap(err))
	}
	// Clear out the target directory if it already exists. If the user has
	// deleted some resources (e.g. deleted a transformer), we don't want to
	// keep the associated value files around
	if os.RemoveAll(externValuesDirPath) != nil {
		uclog.Fatalf(ctx, "Failed to clear directory %s for storing attribute values externally: %v", externValuesDirPath, ucerr.Wrap(err))
	}
	if os.MkdirAll(externValuesDirPath, 0755) != nil {
		uclog.Fatalf(ctx, "Failed to create directory %s for storing attribute values externally: %v", externValuesDirPath, ucerr.Wrap(err))
	}

	mfest, err := manifest.GenerateNewManifest(ctx, idpClient, fqtn, &manifest.ExternValuesDirConfig{
		AbsolutePath:             externValuesDirPath,
		RelativePathFromManifest: "./" + externValuesDirName,
	})
	if err != nil {
		uclog.Fatalf(ctx, "Failed to generate manifest: %v", ucerr.Wrap(err))
	}

	var serialized []byte
	switch filepath.Ext(manifestPath) {
	case ".json":
		serialized, err = json.MarshalIndent(mfest, "", "  ")
	case ".yaml":
		serialized, err = yaml.Marshal(mfest)
	default:
		uclog.Fatalf(ctx, "Manifest path must have .json or .yaml extension")
		return
	}
	if err != nil {
		uclog.Fatalf(ctx, "Failed to serialize manifest: %v", ucerr.Wrap(err))
	}

	err = os.WriteFile(manifestPath, serialized, 0644)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to write manifest: %v", ucerr.Wrap(err))
	}
	uclog.Infof(ctx, "Wrote %d resources into manifest: %s", len(mfest.Resources), manifestPath)
}
