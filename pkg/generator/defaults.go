package generator

import (
	"encoding/json"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func getStringDefault(sProp apiextensionsv1.JSONSchemaProps, additionalImports *AdditionalImports) (string, error) {
	if sProp.Default == nil {
		return "", nil
	}

	var str string
	if err := json.Unmarshal(sProp.Default.Raw, &str); err != nil {
		return "", fmt.Errorf("failed to unmarshal default string: %w", err)
	}

	additionalImports.DefaultsString = true

	return fmt.Sprintf("stringdefault.StaticString(\"%s\")", str), nil
}

func getIntegerDefault(sProp apiextensionsv1.JSONSchemaProps, additionalImports *AdditionalImports) (string, error) {
	if sProp.Default == nil {
		return "", nil
	}

	var integer int64
	if err := json.Unmarshal(sProp.Default.Raw, &integer); err != nil {
		return "", fmt.Errorf("failed to unmarshal default int64: %w", err)
	}

	additionalImports.DefaultsInt64 = true

	return fmt.Sprintf("int64default.StaticInt64(%d)", integer), nil
}

func getNumberDefault(sProp apiextensionsv1.JSONSchemaProps, additionalImports *AdditionalImports) (string, error) {
	if sProp.Default == nil {
		return "", nil
	}

	var number float64
	if err := json.Unmarshal(sProp.Default.Raw, &number); err != nil {
		return "", fmt.Errorf("failed to unmarshal default float64: %w", err)
	}

	additionalImports.DefaultsFloat64 = true

	return fmt.Sprintf("float64default.StaticFloat64(%f)", number), nil
}

func getBooleanDefault(sProp apiextensionsv1.JSONSchemaProps, additionalImports *AdditionalImports) (string, error) {
	if sProp.Default == nil {
		return "", nil
	}

	var boolean bool
	if err := json.Unmarshal(sProp.Default.Raw, &boolean); err != nil {
		return "", fmt.Errorf("failed to unmarshal default bool: %w", err)
	}

	additionalImports.DefaultsBool = true

	return fmt.Sprintf("booldefault.StaticBool(%t)", boolean), nil
}
