# ucconfig

ucconfig (UserClouds Config) enables declarative configuration of UserClouds
resources.

ucconfig is implemented as a usability layer on top of Terraform. It is not as
powerful or flexible as Terraform, but it avoids many of the common
frustrations of working with Terraform, such as manually writing configuration
or managing Terraform state. Since ucconfig generates Terraform configuration
and state, you can "eject" from ucconfig and manage UserClouds resources
directly with Terraform at any point.

## Getting Started

TODO

## Usage

For all commands, ucconfig requires `USERCLOUDS_TENANT_URL`, `USERCLOUDS_CLIENT_ID`, and `USERCLOUDS_CLIENT_SECRET` environment variables to be set.

### Generating a manifest

Rather than needing to write your configuration by hand, the ucconfig
`gen-manifest` subcommand can generate a manifest file based on the existing
resources in a tenant.

`gen-manifest` takes a file path to write the manifest to:

```
ucconfig gen-manifest <manifest-path>
```

JSON and YAML are supported.

For example:

```
ucconfig gen-manifest output.yaml
```

### Applying a manifest

A manifest is a complete description of a tenant's resources. You can use the
`apply` subcommand to create new resources described by a manifest, delete
resources that don't appear in the manifest, and update existing resources
whose properties differ from what the manifest describes.

`apply` takes the path to the manifest to apply:

```
ucconfig apply <manifest-path>
```

### Manifest IDs

### Functions
