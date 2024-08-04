package generator

import (
	"fmt"
	"slices"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func getStringValidators(sProp *apiextensionsv1.JSONSchemaProps, additionalImports *AdditionalImports) []string {
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

	// MinLength validator
	if sProp.MinLength != nil {
		additionalImports.ValidatorString = true

		validators = append(validators, fmt.Sprintf("stringvalidator.LengthAtLeast(%v)", *sProp.MinLength))
	}

	// MaxLength validator
	if sProp.MaxLength != nil {
		additionalImports.ValidatorString = true

		validators = append(validators, fmt.Sprintf("stringvalidator.LengthAtMost(%v)", *sProp.MaxLength))
	}

	// Pattern validator
	if sProp.Pattern != "" {
		additionalImports.ValidatorString = true

		validators = append(validators, fmt.Sprintf("stringvalidator.RegexMatches(regexp.MustCompile(%s), \"\")", escapeRegexPattern(sProp.Pattern)))
	}

	// Byte format validator
	if sProp.Format == "byte" {
		validators = append(validators, "validators.Base64Validator()")
	}

	// Datetime format validator
	if sProp.Format == "date-time" {
		validators = append(validators, "validators.DateTime64Validator()")
	}

	return validators
}

func getIntegerValidators(sProp *apiextensionsv1.JSONSchemaProps, additionalImports *AdditionalImports) []string {
	var validators []string

	// Enum validator
	if len(sProp.Enum) > 0 {
		additionalImports.ValidatorInt64 = true

		var enums []string

		for _, enum := range sProp.Enum {
			if str := string(enum.Raw); str != "" {
				enums = append(enums, str)
			}
		}

		validators = append(validators, fmt.Sprintf("int64validator.OneOf(%s)", strings.Join(enums, ", ")))
	}

	// Minimum validator
	if sProp.Minimum != nil {
		additionalImports.ValidatorInt64 = true

		validators = append(validators, fmt.Sprintf("int64validator.AtLeast(%v)", crdv1MinValue(sProp)))
	}

	// Maximum validator
	if sProp.Maximum != nil {
		additionalImports.ValidatorInt64 = true

		validators = append(validators, fmt.Sprintf("int64validator.AtMost(%v)", crdv1MaxValue(sProp)))
	}

	return validators
}

func getNumberValidators(sProp *apiextensionsv1.JSONSchemaProps, additionalImports *AdditionalImports) []string {
	var validators []string

	// Enum validator
	if len(sProp.Enum) > 0 {
		additionalImports.ValidatorFloat64 = true

		var enums []string

		for _, enum := range sProp.Enum {
			if str := string(enum.Raw); str != "" {
				enums = append(enums, str)
			}
		}

		validators = append(validators, fmt.Sprintf("float64validator.OneOf(%s)", strings.Join(enums, ", ")))
	}

	// Minimum validator
	if sProp.Minimum != nil {
		additionalImports.ValidatorFloat64 = true

		validators = append(validators, fmt.Sprintf("float64validator.AtLeast(%v)", crdv1MinValue(sProp)))
	}

	// Maximum validator
	if sProp.Maximum != nil {
		additionalImports.ValidatorFloat64 = true

		validators = append(validators, fmt.Sprintf("float64validator.AtMost(%v)", crdv1MaxValue(sProp)))
	}

	return validators
}

// handle JSON schema exclusiveMaximum
// From https://github.com/metio/terraform-provider-k8s/blob/faae52f524637d0778ff84c94930cd08eebf3a89/tools/internal/generator/crdv1_validator_extractor.go#L160-L161
func crdv1MaxValue(prop *apiextensionsv1.JSONSchemaProps) float64 {
	max := *prop.Maximum
	if prop.ExclusiveMaximum {
		max = max - 1
	}
	return max
}

// handle JSON schema exclusiveMinimum
func crdv1MinValue(prop *apiextensionsv1.JSONSchemaProps) float64 {
	min := *prop.Minimum
	if prop.ExclusiveMinimum {
		min = min + 1
	}
	return min
}

// escapeRegexPattern escapes a regex pattern for use in a Go string.
func escapeRegexPattern(pattern string) string {
	splits := strings.Split(pattern, "`")
	splits = slices.DeleteFunc(splits, func(s string) bool {
		return s == ""
	})
	if strings.Contains(pattern, "`") {
		var sb strings.Builder
		if strings.HasPrefix(pattern, "`") {
			if len(splits) > 0 {
				sb.WriteString(fmt.Sprintf(`"%c"+`, '`'))
			} else {
				sb.WriteString(fmt.Sprintf(`"%c"`, '`'))
			}
		}
		for index, value := range splits {
			if index > 0 && splits[index-1] != "" {
				sb.WriteString(fmt.Sprintf(`+%c%s%c`, '`', value, '`'))
			} else {
				sb.WriteString(fmt.Sprintf(`%c%s%c`, '`', value, '`'))
			}
			if index < len(splits)-1 {
				sb.WriteString(fmt.Sprintf(`+"%c"`, '`'))
			}
		}
		if strings.HasSuffix(pattern, "`") {
			if len(splits) > 0 {
				sb.WriteString(fmt.Sprintf(`+"%c"`, '`'))
			}
		}
		return sb.String()
	} else {
		return fmt.Sprintf("%c%s%c", '`', pattern, '`')
	}
}
