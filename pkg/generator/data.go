package generator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type Data struct {
	Group             string
	Version           string
	Resource          string
	Kind              string
	ResourceName      string
	PackageName       string
	ModuleName        string
	AdditionalImports AdditionalImports
	SpecProperties    []*Property
	StatusProperties  []*Property
}

type Property struct {
	Name           string
	TFName         string // Terraform argument name is snake case
	Description    string
	FieldName      string
	GoType         string
	ArgumentType   string
	ElementType    string
	Required       bool
	Optional       bool
	Computed       bool
	Immutable      bool
	Default        string
	ValidatorsType string
	Validators     []string

	Properties []*Property
}

type AdditionalImports struct {
	DefaultsString  bool
	DefaultsInt64   bool
	DefaultsFloat64 bool
	DefaultsBool    bool

	ValidatorString  bool
	ValidatorInt64   bool
	ValidatorFloat64 bool
}

var capitalizer = cases.Title(language.English, cases.NoLower)

func parseSchema(file string) (*Data, error) {
	crd, err := loadSchema(file)
	if err != nil {
		return nil, fmt.Errorf("failed to load schema: %w", err)
	}

	return crdToData(crd)
}

func loadSchema(filename string) (*apiextensionsv1.CustomResourceDefinition, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	bufr := bufio.NewReader(file)
	yamlReader := yaml.NewYAMLReader(bufr)

	data, err := yamlReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read yaml from file %s: %w", filename, err)
	}

	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := yaml.Unmarshal(data, crd); err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml from file %s: %w", filename, err)
	}

	return crd, nil
}

func crdToData(crd *apiextensionsv1.CustomResourceDefinition) (*Data, error) {
	group := crd.Spec.Group
	kind := crd.Spec.Names.Kind
	resourceName := strings.ToLower(kind)

	// We assume that there is only one version
	version := crd.Spec.Versions[0]

	schema := version.Schema.OpenAPIV3Schema

	// Delete crossplane specific spec fields
	spec := schema.Properties["spec"]
	crossplaneSpecFields := [...]string{
		"compositeDeletePolicy",
		"compositionRef",
		"compositionRevisionRef",
		"compositionRevisionSelector",
		"compositionSelector",
		"compositionUpdatePolicy",
		"publishConnectionDetailsTo",
		"resourceRef",
		"writeConnectionSecretToRef",
	}

	for _, field := range crossplaneSpecFields {
		delete(spec.Properties, field)
	}

	// Delete crossplane specific status fields
	status := schema.Properties["status"]
	crossplaneStatusFields := [...]string{
		"connectionDetails",
		"conditions",
	}

	for _, field := range crossplaneStatusFields {
		delete(status.Properties, field)
	}

	var additionalImports AdditionalImports

	specProperties, err := crdProperties(&spec, &additionalImports, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get spec properties: %w", err)
	}
	statusProperties, err := crdProperties(&status, &additionalImports, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get status properties: %w", err)
	}

	return &Data{
		Kind:              kind,
		Group:             group,
		Resource:          crd.Spec.Names.Plural,
		Version:           version.Name,
		ResourceName:      resourceName,
		PackageName:       strings.ReplaceAll(group, ".", "_") + "_" + resourceName + "_" + strings.ToLower(version.Name),
		AdditionalImports: additionalImports,
		SpecProperties:    specProperties,
		StatusProperties:  statusProperties,
	}, nil
}

func crdProperties(schema *apiextensionsv1.JSONSchemaProps, additionalImports *AdditionalImports, computed bool) ([]*Property, error) {
	properties := make([]*Property, 0, len(schema.Properties))
	// Iterate over the properties of the schema. Recursively call crdProperties.
	for name, sProp := range schema.Properties {
		goType, argumentType, elementType, dflt, err := convertCrdType(sProp, additionalImports, computed)
		if err != nil {
			return nil, fmt.Errorf("failed to convert CRD type: %w", err)
		}

		validatorsType, validators := getValidators(sProp, additionalImports)

		var nestedProperties []*Property

		switch goType {
		case "map":
			nestedProperties, err = crdProperties(sProp.AdditionalProperties.Schema, additionalImports, computed)
		case "struct":
			nestedProperties, err = crdProperties(&sProp, additionalImports, computed)
		case "array":
			nestedProperties, err = crdProperties(sProp.Items.Schema, additionalImports, computed)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to get nested CRD properties: %w", err)
		}

		description := description(sProp.Description)
		immutable := false

		if strings.HasPrefix(description, "(immutable)") {
			immutable = true
			description = strings.TrimPrefix(description, "(immutable)")
		}

		description = strings.TrimSpace(description)

		prop := &Property{
			Name:           name,
			TFName:         toSnakeCase(name),
			Description:    description,
			FieldName:      capitalizer.String(name),
			GoType:         goType,
			ArgumentType:   argumentType,
			ElementType:    elementType,
			Computed:       computed || dflt != "",
			Immutable:      immutable,
			Default:        dflt,
			ValidatorsType: validatorsType,
			Validators:     validators,

			Properties: nestedProperties,
		}

		properties = append(properties, prop)
	}

	// Mark required properties
	for _, prop := range properties {
		if slices.Contains(schema.Required, prop.Name) {
			prop.Required = true
			prop.Optional = false
		} else {
			prop.Required = false
			prop.Optional = true
		}
	}

	// Sort properties by name
	// This is important for the generated code to be deterministic
	sort.SliceStable(properties, func(i, j int) bool {
		return properties[i].Name < properties[j].Name
	})

	return properties, nil
}

// convertCrdType converts a JSON schema type to a Go type and a Terraform argument type.
func convertCrdType(sProp apiextensionsv1.JSONSchemaProps, additionalImports *AdditionalImports, computed bool) (string, string, string, string, error) {
	var (
		goType       string
		argumentType string
		elementType  string
		dflt         string
	)

	switch sProp.Type {
	case "string":
		goType = "string"
		argumentType = "schema.StringAttribute"

		if sProp.Default != nil {
			var s string
			if err := json.Unmarshal(sProp.Default.Raw, &s); err != nil {
				return "", "", "", "", fmt.Errorf("failed to unmarshal default string: %w", err)
			}

			dflt = fmt.Sprintf("stringdefault.StaticString(\"%s\")", s)
			additionalImports.DefaultsString = true
		}
	case "integer":
		goType = "int64"
		argumentType = "schema.Int64Attribute"

		if sProp.Default != nil {
			var i int64
			if err := json.Unmarshal(sProp.Default.Raw, &i); err != nil {
				return "", "", "", "", fmt.Errorf("failed to unmarshal default int64: %w", err)
			}

			dflt = fmt.Sprintf("int64default.StaticInt64(%d)", i)
			additionalImports.DefaultsInt64 = true
		}
	case "number":
		goType = "float64"
		argumentType = "schema.Float64Attribute"

		if sProp.Default != nil {
			var f float64
			if err := json.Unmarshal(sProp.Default.Raw, &f); err != nil {
				return "", "", "", "", fmt.Errorf("failed to unmarshal default float64: %w", err)
			}

			dflt = fmt.Sprintf("float64default.StaticFloat64(%g)", f)
			additionalImports.DefaultsFloat64 = true
		}
	case "boolean":
		goType = "bool"
		argumentType = "schema.BoolAttribute"

		if sProp.Default != nil {
			var b bool
			if err := json.Unmarshal(sProp.Default.Raw, &b); err != nil {
				return "", "", "", "", fmt.Errorf("failed to unmarshal default bool: %w", err)
			}

			dflt = fmt.Sprintf("booldefault.StaticBool(%t)", b)
			additionalImports.DefaultsBool = true
		}
	case "object":
		// AdditionalProperties and Properties are mutually exclusive
		if sProp.AdditionalProperties != nil { // object with AdditionalProperties is a map
			if sProp.AdditionalProperties.Schema.Type == "object" { // map[string]struct
				goType = "map"
				argumentType = "schema.MapNestedAttribute"
			} else { // map[string]primitive
				argumentType = "schema.MapAttribute"
				goType, elementType = getTfPrimitiveType(sProp.AdditionalProperties.Schema.Type)
				goType = "map[string]" + goType
			}
		} else if len(sProp.Properties) > 0 { // object with Properties is a struct
			goType = "struct"
			argumentType = "schema.SingleNestedAttribute"
		}
	case "array":
		if sProp.Items.Schema.Type == "object" { // array of struct
			goType = "array"
			argumentType = "schema.ListNestedAttribute"
		} else { // array of primitive
			argumentType = "schema.ListAttribute"
			goType, elementType = getTfPrimitiveType(sProp.Items.Schema.Type)
			goType = "[]" + goType
		}
	}

	if computed {
		goType = "*" + goType
	}

	return goType, argumentType, elementType, dflt, nil
}

func getTfPrimitiveType(crdPrimitiveType string) (string, string) {
	var tfType string

	elementType := "types."

	switch crdPrimitiveType {
	case "string":
		tfType = "string"
		elementType += "StringType"
	case "integer":
		tfType = "int64"
		elementType += "Int64Type"
	case "number":
		tfType = "float64"
		elementType += "Float64Type"
	case "boolean":
		tfType = "bool"
		elementType += "BoolType"
	}

	return tfType, elementType
}

func getValidators(sProp apiextensionsv1.JSONSchemaProps, additionalImports *AdditionalImports) (string, []string) {
	var (
		validatorsType string
		validators     []string
	)

	switch sProp.Type {
	case "string":
		validatorsType = "validator.String"

		// validator for string enums
		if len(sProp.Enum) > 0 {
			additionalImports.ValidatorString = true

			var enums []string

			for _, enum := range sProp.Enum {
				if str := string(enum.Raw); str != "" {
					enums = append(enums, str)
				}
			}

			validators = append(validators, fmt.Sprintf("stringvalidator.OneOf(%s)", strings.Join(enums, ", ")))
		}
	case "integer":
		validatorsType = "validator.Int64"
	case "number":
		validatorsType = "validator.Float64"
	}

	return validatorsType, validators
}

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
	matchDashes   = regexp.MustCompile("-")
	matchDots     = regexp.MustCompile(`\.`)
	matchSlashes  = regexp.MustCompile("/")
	matchColons   = regexp.MustCompile(":")
)

// toSnakeCase converts a CamelCase string to snake_case.
// From https://github.com/metio/terraform-provider-k8s/blob/faae52f524637d0778ff84c94930cd08eebf3a89/tools/internal/generator/converter.go#L166
func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	snake = matchDashes.ReplaceAllString(snake, "_")
	snake = matchDots.ReplaceAllString(snake, "_")
	snake = matchSlashes.ReplaceAllString(snake, "_")
	snake = matchColons.ReplaceAllString(snake, "_")

	return strings.ToLower(snake)
}

var (
	matchBackticks    = regexp.MustCompile(`\x60`)
	matchDoubleQuotes = regexp.MustCompile("\"")
	matchNewlines     = regexp.MustCompile("\n")
	matchBackslashes  = regexp.MustCompile(`\\`)
)

// description cleans up the description field of a property.
// From https://github.com/metio/terraform-provider-k8s/blob/faae52f524637d0778ff84c94930cd08eebf3a89/tools/internal/generator/converter.go#L337
func description(description string) string {
	clean := matchBackticks.ReplaceAllString(description, "'")
	clean = matchDoubleQuotes.ReplaceAllString(clean, "'")
	clean = matchNewlines.ReplaceAllString(clean, "")
	clean = matchBackslashes.ReplaceAllString(clean, "")

	return clean
}
