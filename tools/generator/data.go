package main

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type Data struct {
	ResourceName     string
	PackageName      string
	CrdApiVersion    string
	RmTypeName       string
	SpecProperties   []*Property
	StatusProperties []*Property
}

type Property struct {
	Name         string
	Description  string
	FieldName    string
	Type         string
	TFType       string
	ArgumentType string
	ElementType  string
	Properties   []*Property
	Required     bool
	Optional     bool
	Computed     bool
	Immutable    bool
}

var capitalizer = cases.Title(language.English, cases.NoLower)

func parseSchema(file string) (*Data, error) {
	crd, err := loadSchema(file)
	if err != nil {
		return nil, fmt.Errorf("failed to load schema: %w", err)
	}

	return crdToData(crd), nil
}

func loadSchema(file string) (*apiextensions.CustomResourceDefinition, error) {
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

	crd := &apiextensions.CustomResourceDefinition{}
	if err := yaml.Unmarshal(data, crd); err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml from file %s: %w", file, err)
	}

	return crd, nil
}

func crdToData(crd *apiextensions.CustomResourceDefinition) *Data {
	group := crd.Spec.Group
	kind := strings.ToLower(crd.Spec.Names.Kind)

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
		ResourceName:     kind,
		PackageName:      strings.Replace(group, ".", "_", -1) + "_" + kind + "_" + strings.ToLower(version.Name),
		CrdApiVersion:    group + "/" + version.Name,
		RmTypeName:       strings.ToLower(kind) + "ResourceModel",
		SpecProperties:   crdProperties(spec, false),
		StatusProperties: crdProperties(status, true),
	}
}

func crdProperties(schema apiextensions.JSONSchemaProps, computed bool) []*Property {
	properties := make([]*Property, 0, len(schema.Properties))
	// Iterate over the properties of the schema. Recursively call crdProperties.
	for name, sProp := range schema.Properties {
		// Compute types based on the schema type
		var typeName string
		var tfTypeName string
		var argumentTypeName string
		var elementTypeName string
		switch sProp.Type {
		case "string":
			typeName = "string"
			tfTypeName = "types.String"
			argumentTypeName = "schema.StringAttribute"
		case "object":
			typeName = "map[string]string"
			tfTypeName = "types.Map"
			argumentTypeName = "schema.MapAttribute"
			elementTypeName = "types.StringType"
		}
		if computed {
			typeName = "*" + typeName
		}

		description := sProp.Description
		immutable := false
		if strings.HasPrefix(description, "#immutable#") {
			immutable = true
			description = strings.TrimPrefix(description, "#immutable#")
		}

		prop := &Property{
			Name:         name,
			Description:  description,
			FieldName:    capitalizer.String(name),
			Type:         typeName,
			TFType:       tfTypeName,
			ArgumentType: argumentTypeName,
			ElementType:  elementTypeName,
			Computed:     computed,
			Immutable:    immutable,
			Properties:   crdProperties(sProp, computed),
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
