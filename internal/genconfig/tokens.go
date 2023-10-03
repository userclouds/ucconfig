package genconfig

import (
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"

	"userclouds.com/cmd/ucconfig/internal/liveresource"
	"userclouds.com/cmd/ucconfig/internal/manifest"
	"userclouds.com/infra/ucerr"
)

type functionInvocation struct {
	Name       string
	Params     []any
	PathSuffix []string
}

func parseFunctionInvocation(invocation string) *functionInvocation {
	baseRegex := regexp.MustCompile(`^@(?P<funcname>UC_[A-Z_]+)\((?P<params>.*?)?\)(?P<pathsuffix>(?:\.[A-Za-z0-9_-]+)*)$`)
	groups := baseRegex.FindStringSubmatch(invocation)
	if len(groups) == 0 {
		return nil
	}
	out := functionInvocation{
		Name: groups[1],
	}
	params := groups[2]
	for _, param := range strings.Split(params, ",") {
		param = strings.TrimSpace(param)
		if param[0] == '"' && param[len(param)-1] == '"' {
			out.Params = append(out.Params, param[1:len(param)-1])
		} else if b, err := strconv.ParseBool(param); err != nil {
			out.Params = append(out.Params, b)
		} else if u, err := strconv.ParseUint(param, 10, 64); err != nil {
			out.Params = append(out.Params, u)
		} else if i, err := strconv.ParseInt(param, 10, 64); err != nil {
			out.Params = append(out.Params, i)
		} else if f, err := strconv.ParseFloat(param, 64); err != nil {
			out.Params = append(out.Params, f)
		} else {
			return nil
		}
	}
	pathsuffix := groups[3]
	out.PathSuffix = strings.Split(pathsuffix, ".")[1:]
	return &out
}

func ucManifestID(invocation *functionInvocation, mfest *manifest.Manifest, liveResources *[]liveresource.Resource) (hclwrite.Tokens, error) {
	if len(invocation.Params) != 1 {
		return []*hclwrite.Token{}, ucerr.Errorf("UC_MANIFEST_ID takes exactly 1 parameter")
	}
	if reflect.ValueOf(invocation.Params[0]).Kind() != reflect.String {
		return []*hclwrite.Token{}, ucerr.Errorf("UC_MANIFEST_ID takes a string parameter")
	}
	manifestID := invocation.Params[0].(string)
	var matchingResource *manifest.Resource
	for i := range mfest.Resources {
		if mfest.Resources[i].ManifestID == manifestID {
			matchingResource = &mfest.Resources[i]
			break
		}
	}
	if matchingResource == nil {
		return []*hclwrite.Token{}, ucerr.Errorf("could not find resource with manifest ID %s for UC_MANIFEST_ID invocation", manifestID)
	}
	if len(invocation.PathSuffix) == 0 {
		return []*hclwrite.Token{}, ucerr.Errorf("expected manifest to access attributes of resource returned by UC_MANIFEST_ID")
	}
	traversal := hcl.Traversal{
		hcl.TraverseRoot{Name: "userclouds_" + matchingResource.TerraformTypeSuffix},
		hcl.TraverseAttr{Name: "manifestid-" + matchingResource.ManifestID},
	}
	for _, pathPart := range invocation.PathSuffix {
		traversal = append(traversal, hcl.TraverseAttr{Name: pathPart})
	}
	return hclwrite.TokensForTraversal(traversal), nil
}

func ucSystemObject(invocation *functionInvocation, mfest *manifest.Manifest, liveResources *[]liveresource.Resource) (hclwrite.Tokens, error) {
	if len(invocation.Params) != 2 {
		return []*hclwrite.Token{}, ucerr.Errorf("UC_SYSTEM_OBJECT takes exactly 2 parameters")
	}
	if reflect.ValueOf(invocation.Params[0]).Kind() != reflect.String ||
		reflect.ValueOf(invocation.Params[1]).Kind() != reflect.String {
		return []*hclwrite.Token{}, ucerr.Errorf("UC_SYSTEM_OBJECT parameters must be strings")
	}
	terraformTypeSuffix := invocation.Params[0].(string)
	objectName := invocation.Params[1].(string)
	var matchingResource *liveresource.Resource
	for _, resource := range *liveResources {
		if resource.TerraformTypeSuffix == terraformTypeSuffix && resource.IsSystem && resource.Attributes["name"].(string) == objectName {
			matchingResource = &resource
			break
		}
	}
	if matchingResource == nil {
		return []*hclwrite.Token{}, ucerr.Errorf("could not find system object with type %s and name %s for UC_SYSTEM_OBJECT invocation", terraformTypeSuffix, objectName)
	}
	if len(invocation.PathSuffix) != 0 {
		return []*hclwrite.Token{}, ucerr.Errorf("UC_SYSTEM_OBJECT returns a string, so path suffixes may not be used")
	}
	return hclwrite.TokensForValue(cty.StringVal(matchingResource.ResourceUUID)), nil
}

// This function takes an arbitrary value and generates HCL lexer tokens to represent that value in
// HCL. The hclwrite package does have SetAttributeValue and NewExpressionLiteral functions that
// generate tokens for a value, but (1) it takes cty values, and it already takes some work to
// convert an arbitrary golang value into a cty value, and (2) as of writing, hclwrite only supports
// setting *literal* values, which prevents us from generating Terraform with expressions and
// references to other resources. (SetAttributeTraversal supports generating references to other
// resources, but only for top-level attributes, not inside of nested objects or arrays.)
func toHclTokens(val any, mfest *manifest.Manifest, liveResources *[]liveresource.Resource) (hclwrite.Tokens, error) {
	// Nil
	if val == nil {
		return []*hclwrite.Token{{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("null"),
		}}, nil
	}

	reflectVal := reflect.ValueOf(val)

	// Pointers
	if reflectVal.Kind() == reflect.Ptr {
		return toHclTokens(reflectVal.Elem(), mfest, liveResources)
	}

	// Arrays
	if reflectVal.Kind() == reflect.Array || reflectVal.Kind() == reflect.Slice {
		var tokens []hclwrite.Tokens
		for i := 0; i < reflectVal.Len(); i++ {
			itemRawVal := reflectVal.Index(i).Interface()
			itemTokens, err := toHclTokens(itemRawVal, mfest, liveResources)
			if err != nil {
				return []*hclwrite.Token{}, ucerr.Errorf("error generating tokens for array value %+v at index %s: %v", itemRawVal, i, err)
			}
			tokens = append(tokens, itemTokens)
		}
		return hclwrite.TokensForTuple(tokens), nil
	}

	// Objects/maps (can't easily distinguish the two since our data came from a JSON file, where
	// there is no difference)
	if reflectVal.Kind() == reflect.Map {
		var tokens []hclwrite.ObjectAttrTokens
		keys := reflectVal.MapKeys()
		sort.SliceStable(keys, func(i int, j int) bool { return keys[i].String() < keys[j].String() })
		for _, key := range keys {
			itemRawVal := reflectVal.MapIndex(key).Interface()
			itemTokens, err := toHclTokens(itemRawVal, mfest, liveResources)
			if err != nil {
				return []*hclwrite.Token{}, ucerr.Errorf("error generating tokens for map value %+v under key %s: %v", itemRawVal, key.String(), err)
			}
			tokens = append(tokens, hclwrite.ObjectAttrTokens{
				Name:  hclwrite.TokensForValue(cty.StringVal(key.String())),
				Value: itemTokens,
			})
		}
		return hclwrite.TokensForObject(tokens), nil
	}

	// ucconfig functions:
	if reflectVal.Kind() == reflect.String {
		if invocation := parseFunctionInvocation(reflectVal.String()); invocation != nil {
			if invocation.Name == "UC_MANIFEST_ID" {
				return ucManifestID(invocation, mfest, liveResources)
			}
			if invocation.Name == "UC_SYSTEM_OBJECT" {
				return ucSystemObject(invocation, mfest, liveResources)
			}
			return []*hclwrite.Token{}, ucerr.Errorf("unknown function %s", invocation.Name)
		}
	}

	// Primitive types: use out-of-the-box TokensForValue with cty values
	ctyType, err := gocty.ImpliedType(val)
	if err != nil {
		return []*hclwrite.Token{}, ucerr.Errorf("could not infer cty type for %+v: %v", val, err)
	}
	ctyVal, err := gocty.ToCtyValue(val, ctyType)
	if err != nil {
		return []*hclwrite.Token{}, ucerr.Errorf("could not convert value %+v to a cty value: %v", val, err)
	}
	return hclwrite.TokensForValue(ctyVal), nil
}
