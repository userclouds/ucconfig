module userclouds.com/cmd/ucconfig

go 1.20

require (
	github.com/alecthomas/kong v0.8.0
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/hashicorp/hcl/v2 v2.18.0
	github.com/zclconf/go-cty v1.14.0
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9
	gopkg.in/yaml.v3 v3.0.1
	userclouds.com v0.7.7
)

require (
	github.com/agext/levenshtein v1.2.1 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/mitchellh/go-wordwrap v0.0.0-20150314170334-ad45545899c7 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/redis/go-redis/v9 v9.3.0 // indirect
	github.com/rogpeppe/go-internal v1.10.1-0.20230524175051-ec119421bb97 // indirect
	github.com/sergi/go-diff v1.3.1 // indirect
	golang.org/x/text v0.13.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

// Note: the yaml library currently has a bug where it loses newlines:
// https://github.com/go-yaml/yaml/pull/964
// Sadly, the library looks kind of dead, but we can remove this "replace" if this PR ever gets
// merged.
replace gopkg.in/yaml.v3 => github.com/iliakap/yaml v0.0.0-20230523123203-47a88add8517
