package generator

import (
	"bufio"
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
	Group            string
	Version          string
	Resource         string
	Kind             string
	ResourceName     string
	PackageName      string
	ModuleName       string
	SpecProperties   []*Property
	StatusProperties []*Property
}

type Property struct {
	Name         string
	TFName       string // Terraform argument name is snake case
	Description  string
	FieldName    string
	GoType       string
	ArgumentType string
	ElementType  string
	Required     bool
	Optional     bool
	Computed     bool
	Immutable    bool
	Properties   []*Property
}

var capitalizer = cases.Title(language.English, cases.NoLower)

func parseSchema(file string) (*Data, error) {
	crd, err := loadSchema(file)
	if err != nil {
		return nil, fmt.Errorf("failed to load schema: %w", err)
	}

	return crdToData(crd), nil
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

func crdToData(crd *apiextensionsv1.CustomResourceDefinition) *Data {
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

	return &Data{
		Kind:             kind,
		Group:            group,
		Resource:         crd.Spec.Names.Plural,
		Version:          version.Name,
		ResourceName:     resourceName,
		PackageName:      strings.ReplaceAll(group, ".", "_") + "_" + resourceName + "_" + strings.ToLower(version.Name),
		SpecProperties:   crdProperties(&spec, false),
		StatusProperties: crdProperties(&status, true),
	}
}

func crdProperties(schema *apiextensionsv1.JSONSchemaProps, computed bool) []*Property {
	properties := make([]*Property, 0, len(schema.Properties))
	// Iterate over the properties of the schema. Recursively call crdProperties.
	for name, sProp := range schema.Properties {
		goType, argumentType, elementType := convertCrdType(sProp, computed)

		var nestedProperties []*Property

		switch goType {
		case "map":
			nestedProperties = crdProperties(sProp.AdditionalProperties.Schema, computed)
		case "struct":
			nestedProperties = crdProperties(&sProp, computed)
		case "array":
			nestedProperties = crdProperties(sProp.Items.Schema, computed)
		}

		description := sProp.Description
		immutable := false

		if strings.HasPrefix(description, "(immutable)") {
			immutable = true
			description = strings.TrimPrefix(description, "(immutable)")
		}

		description = strings.TrimSpace(description)

		prop := &Property{
			Name:         name,
			TFName:       toSnakeCase(name),
			Description:  description,
			FieldName:    capitalizer.String(name),
			GoType:       goType,
			ArgumentType: argumentType,
			ElementType:  elementType,
			Computed:     computed,
			Immutable:    immutable,
			Properties:   nestedProperties,
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

	return properties
}

// convertCrdType converts a JSON schema type to a Go type and a Terraform argument type.
func convertCrdType(sProp apiextensionsv1.JSONSchemaProps, computed bool) (string, string, string) {
	var (
		goType       string
		argumentType string
		elementType  string
	)

	switch sProp.Type {
	case "string":
		goType = "string"
		argumentType = "schema.StringAttribute"
	case "integer":
		goType = "int64"
		argumentType = "schema.Int64Attribute"
	case "number":
		goType = "float64"
		argumentType = "schema.Float64Attribute"
	case "boolean":
		goType = "bool"
		argumentType = "schema.BoolAttribute"
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

	return goType, argumentType, elementType
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

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")

	return strings.ToLower(snake)
}
