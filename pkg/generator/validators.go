package generator

import (
	"fmt"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func getStringValidators(sProp apiextensionsv1.JSONSchemaProps, additionalImports *AdditionalImports) []string {
	var validators []string

	// Enum validator
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

	return validators
}
