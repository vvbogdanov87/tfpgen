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
	Type         string
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

func loadSchema(file string) (*apiextensionsv1.CustomResourceDefinition, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", file, err)
	}
	defer f.Close()

	bufr := bufio.NewReader(f)
	yamlReader := yaml.NewYAMLReader(bufr)
	data, err := yamlReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read yaml from file %s: %w", file, err)
	}

	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := yaml.Unmarshal(data, crd); err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml from file %s: %w", file, err)
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
		PackageName:      strings.Replace(group, ".", "_", -1) + "_" + resourceName + "_" + strings.ToLower(version.Name),
		SpecProperties:   crdProperties(&spec, false),
		StatusProperties: crdProperties(&status, true),
	}
}

func crdProperties(schema *apiextensionsv1.JSONSchemaProps, computed bool) []*Property {
	properties := make([]*Property, 0, len(schema.Properties))
	// Iterate over the properties of the schema. Recursively call crdProperties.
	for name, sProp := range schema.Properties {
		// Compute types based on the schema type
		var typeName string
		// var tfTypeName string
		var argumentType string
		var elementType string

		var nestedProperties []*Property

		switch sProp.Type {
		case "string":
			typeName = "string"
			argumentType = "schema.StringAttribute"
		case "integer":
			typeName = "int64"
			argumentType = "schema.Int64Attribute"
		case "number":
			typeName = "float64"
			argumentType = "schema.Float64Attribute"
		case "boolean":
			typeName = "bool"
			argumentType = "schema.BoolAttribute"
		case "object":
			// AdditionalProperties and Properties are mutually exclusive
			if sProp.AdditionalProperties != nil { // object with AdditionalProperties is a map
				if sProp.AdditionalProperties.Schema.Type == "object" { // map[string]struct
					typeName = "map"
					argumentType = "schema.MapNestedAttribute"
					nestedProperties = crdProperties(sProp.AdditionalProperties.Schema, computed)
				} else { // map[string]primitive
					argumentType = "schema.MapAttribute"

					typeName = "map[string]"
					elementType = "types."
					switch sProp.AdditionalProperties.Schema.Type {
					case "string":
						typeName += "string"
						elementType += "StringType"
					case "integer":
						typeName += "int64"
						elementType += "Int64Type"
					case "number":
						typeName += "float64"
						elementType += "Float64Type"
					case "boolean":
						typeName += "bool"
						elementType += "BoolType"
					}
				}
			} else if len(sProp.Properties) > 0 { // object with Properties is a struct
				typeName = "struct"
				argumentType = "schema.SingleNestedAttribute"
				nestedProperties = crdProperties(&sProp, computed)
			}
		case "array":
			if sProp.Items.Schema.Type == "object" { // array of struct
				typeName = "array"
				argumentType = "schema.ListNestedAttribute"
				nestedProperties = crdProperties(sProp.Items.Schema, computed)
			} else { // array of primitive
				typeName = "[]" + sProp.Items.Schema.Type
				argumentType = "schema.ListAttribute"
				elementType = "types." + capitalizer.String(sProp.Items.Schema.Type) + "Type"
			}
		}
		if computed {
			typeName = "*" + typeName
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
			Type:         typeName,
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

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
