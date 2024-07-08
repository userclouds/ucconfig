module userclouds.com/cmd/ucconfig

go 1.21

toolchain go1.22.2

require (
	github.com/alecthomas/kong v0.8.0
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/hashicorp/hcl/v2 v2.18.0
	github.com/zclconf/go-cty v1.14.0
	golang.org/x/exp v0.0.0-20240416160154-fe59bbe5cc7f
	gopkg.in/yaml.v3 v3.0.1
	userclouds.com v1.2.0
)

require (
	github.com/agext/levenshtein v1.2.1 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-http-utils/headers v0.0.0-20181008091004-fed159eddc2a // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/redis/go-redis/v9 v9.5.3 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/sergi/go-diff v1.3.1 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

// Note: the yaml library currently has a bug where it loses newlines:
// https://github.com/go-yaml/yaml/pull/964
// Sadly, the library looks kind of dead, but we can remove this "replace" if this PR ever gets
// merged.
replace gopkg.in/yaml.v3 => github.com/iliakap/yaml v0.0.0-20230523123203-47a88add8517
