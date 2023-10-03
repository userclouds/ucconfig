package main

import (
	"context"
	"net/url"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/gofrs/uuid"

	"userclouds.com/cmd/ucconfig/internal/cmd"
	"userclouds.com/idp"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
)

type cliContext struct {
	Context context.Context
	// fully-qualified tenant name, e.g. "mycompany-mytenant"
	FQTN      string
	IDPClient *idp.Client
}

type applyCmd struct {
	ManifestPath string `arg:"" name:"manifest-path" help:"Path to UC JSON manifest file" type:"path"`
}

// Run implements the apply subcommand
func (c *applyCmd) Run(ctx *cliContext) error {
	cmd.Apply(ctx.Context, ctx.IDPClient, ctx.FQTN, c.ManifestPath)
	return nil
}

type genManifestCmd struct {
	ManifestPath string `arg:"" name:"manifest-path" help:"Path to UC JSON manifest file" type:"path"`
}

// Run implements the gen-manifest subcommand
func (c *genManifestCmd) Run(ctx *cliContext) error {
	cmd.GenerateNewManifest(ctx.Context, ctx.IDPClient, ctx.FQTN, c.ManifestPath)
	return nil
}

var cli struct {
	TenantURL    string `env:"USERCLOUDS_TENANT_URL" required:"" help:"Tenant URL."`
	ClientID     string `env:"USERCLOUDS_CLIENT_ID" required:"" help:"Client ID."`
	ClientSecret string `env:"USERCLOUDS_CLIENT_SECRET" required:"" help:"Client secret."`

	Apply       applyCmd       `cmd:"" help:"Apply a config manifest file, modifying the live tenant to match what the manifest describes."`
	GenManifest genManifestCmd `cmd:"" help:"Generate a JSON manifest file from a live tenant."`
}

func main() {
	ctx := context.Background()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelInfo, uclog.LogLevelVerbose, "ucconfig", logtransports.NoPrefix())
	defer logtransports.Close()

	cliCtx := kong.Parse(&cli)

	url, err := url.Parse(cli.TenantURL)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to parse tenant URL: %v", err)
	}
	fqtn := strings.Split(url.Hostname(), ".")[0]

	// Initialize IDP client based on env vars
	tokenSource := jsonclient.ClientCredentialsTokenSource(cli.TenantURL+"/oidc/token", cli.ClientID, cli.ClientSecret, nil)
	orgID := uuid.Nil
	idpClient, err := idp.NewClient(cli.TenantURL, idp.OrganizationID(orgID), idp.JSONClient(tokenSource))
	if err != nil {
		uclog.Fatalf(ctx, "Failed to initialize IDP client: %v", err)
	}

	err = cliCtx.Run(&cliContext{Context: ctx, IDPClient: idpClient, FQTN: fqtn})
	cliCtx.FatalIfErrorf(err)
}
