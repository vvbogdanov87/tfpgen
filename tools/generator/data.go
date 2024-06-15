package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type Data struct {
	PackageName      string
	CrdApiVersion    string
	RmTypeName       string
	SpecProperties   []*Property
	StatusProperties []*Property
}

type Property struct {
	FieldName      string
	AnnotationName string
	Type           string
	TFType         string
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
		PackageName:      strings.Replace(group, ".", "_", -1) + "_" + kind + "_" + strings.ToLower(version.Name),
		CrdApiVersion:    group + "/" + version.Name,
		RmTypeName:       strings.ToLower(kind) + "ResourceModel",
		SpecProperties:   crdProperties(spec),
		StatusProperties: crdProperties(status),
	}
}

func crdProperties(schema apiextensions.JSONSchemaProps) []*Property {
	properties := make([]*Property, 0, len(schema.Properties))
	for name, sProp := range schema.Properties {
		prop := &Property{
			FieldName:      capitalizer.String(name),
			AnnotationName: name,
			Type:           sProp.Type,
			TFType:         "types." + capitalizer.String(sProp.Type),
			Properties:     crdProperties(sProp),
		}

		properties = append(properties, prop)
	}

	return properties
}
