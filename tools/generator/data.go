package main

import (
	"bufio"
	"fmt"
	"os"
	"slices"
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
	CrdApiVersion    string
	SpecProperties   []*Property
	StatusProperties []*Property
}

type Property struct {
	Name           string
	Description    string
	FieldName      string
	Type           string
	TFType         string
	ArgumentType   string
	ElementType    string
	Required       bool
	Optional       bool
	Computed       bool
	Immutable      bool
	GetValueMethod string
	Properties     []*Property
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
		CrdApiVersion:    group + "/" + version.Name,
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
		var tfTypeName string
		var argumentTypeName string
		var elementTypeName string
		var getValueMethod string
		getValueMethod = "Value" + capitalizer.String(typeName) + "()"

		var nestedProperties []*Property

		switch sProp.Type {
		case "string":
			typeName = "string"
			tfTypeName = "types.String"
			argumentTypeName = "schema.StringAttribute"
			getValueMethod = "ValueString()"
		case "object":
			// AdditionalProperties and Properties are mutually exclusive
			if sProp.AdditionalProperties != nil { // object with AdditionalProperties is a map
				if sProp.AdditionalProperties.Schema.Type == "object" { // map[string]struct
					typeName = "map"
					tfTypeName = "types.Map"
					argumentTypeName = "schema.MapNestedAttribute"
					nestedProperties = crdProperties(sProp.AdditionalProperties.Schema, computed)
				} else { // map[string]primitive
					typeName = "map[string]" + sProp.AdditionalProperties.Schema.Type
					tfTypeName = "types.Map"
					argumentTypeName = "schema.MapAttribute"
					elementTypeName = "types." + capitalizer.String(sProp.AdditionalProperties.Schema.Type) + "Type"
					getValueMethod = "Elements()"
				}
			} else if len(sProp.Properties) > 0 { // object with Properties is a struct
				typeName = "struct"
				tfTypeName = "types.Object"
				argumentTypeName = "schema.SingleNestedAttribute"
				nestedProperties = crdProperties(&sProp, computed)
			}
		case "array":
			if sProp.Items.Schema.Type == "object" { // array of struct
				typeName = "array"
				tfTypeName = "types.List"
				argumentTypeName = "schema.ListNestedAttribute"
				nestedProperties = crdProperties(sProp.Items.Schema, computed)
			} else { // array of primitive
				typeName = "[]" + sProp.Items.Schema.Type
				tfTypeName = "types.List"
				argumentTypeName = "schema.ListAttribute"
				elementTypeName = "types." + capitalizer.String(sProp.Items.Schema.Type) + "Type"
			}
		}
		if computed {
			typeName = "*" + typeName
		}

		description := sProp.Description
		immutable := false
		if strings.HasPrefix(description, "(immutable)") {
			immutable = true
			description = strings.TrimPrefix(description, "#immutable#")
		}

		prop := &Property{
			Name:           name,
			Description:    description,
			FieldName:      capitalizer.String(name),
			Type:           typeName,
			TFType:         tfTypeName,
			ArgumentType:   argumentTypeName,
			ElementType:    elementTypeName,
			Computed:       computed,
			Immutable:      immutable,
			GetValueMethod: getValueMethod,
			Properties:     nestedProperties,
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

	return properties
}
