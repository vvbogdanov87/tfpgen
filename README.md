# tfpgen

This project aims to provide a Terraform provider for managing Custom Resource Definitions (CRDs) in Kubernetes. The `tfpgen` tool generates the necessary provider code based on the CRD schemas provided. This provider allows users to easily create, update, and delete CRDs using Terraform. The generated provider code can be installed and used locally or published to the Terraform registry.

## Usage
- Create a repository (the repository address is used as the Go module name)
- Initialize the Go module
    ```shell
    go mod init <module-name>
    ```
    e.g.
    ```shell
    go mod init github.com/vvbogdanov87/terraform-provider-crd
    ```
- Create a `schemas` directory in the repository root and copy CRDs in the directory
- Create a `tfpgen.yaml` file in the repository root
    ```yaml
    name: "crd" # Name is the provider name.
    address: "registry.terraform.io/vvbogdanov87/crd" # Address is the provider address for the Terraform registry.
    moduleName: "github.com/vvbogdanov87/terraform-provider-crd" # ModuleName is the name of the Go module.
    schemasDir: "schemas" # SchemasDir is the directory containing the CRD schemas.
    outputDir: "." # OutputDir is the directory to write the generated provider code.
    ```
- Generate code
    ```shell
    go run github.com/vvbogdanov87/tfpgen --config tfpgen.yaml
    ```
- Add dependencies
    ```shell
    go mod tidy
    ```
- Install the provider
    ```shell
    go install
    ```
- [Point terraform to the installed provider](https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-provider#prepare-terraform-for-local-provider-install). Create file `~/.terraformrc`
    ```hcl
    provider_installation {
        dev_overrides {
            # "provider address" = "$HOME/go/bin"
            "registry.terraform.io/vvbogdanov87/crd" = "/home/viktor/go/bin"
        }
        direct {}
    }
    ```

## CRD properties and TF attributes
CRD camel case property names are mapped to TF snake case attribute names.

| CRD/OpenAPI type                                                | GO type              | TF attribyte type                                  | Support OpenAPI Schema Object default field |
| --------------------------------------------------------------- | -------------------- | -------------------------------------------------- | ------------------------------------------- |
| string                                                          | string               | schema.StringAttribute                             | :white_check_mark:                          |
| integer                                                         | int64                | schema.Int64Attribute                              | :x:                                         |
| number                                                          | float64              | schema.Float64Attribute                            | :x:                                         |
| boolean                                                         | boolean              | schema.BoolAttribute                               | :x:                                         |
| `object` with `AdditionalProperties` and `Schema.Type = object` | map[string]struct    | schema.MapNestedAttribute                          | :x:                                         |
| `object` with `AdditionalProperties`                            | map[string]primitive | schema.MapAttribute                                | :x:                                         |
| `object` with Properties                                        | struct               | schema.SingleNestedAttribute                       | :x:                                         |
| array with `Schema.Type = object`                               | []struct             | schema.ListNestedAttribute                         | :x:                                         |
| array                                                           | []primitive          | schema.ListAttribute                               | :x:                                         |

Note: the field `additionalProperties` is mutually exclusive with `properties`.
[OpenAPI Data Types](https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.0.md#data-types)

## Immutable fields
OpenAPI schema doesn't support immutable fields. Kubernetes uses a [Common Expression Language (CEL)](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#transition-rules) extension to make fields immutable.
To tell the generator that a property is immutable and needs TF attribute plan modifier `RequiresReplace` the prefix `(immutable)` must be added to the property description. E.g.:
```yaml
prefix:
  type: string
  description: "(immutable) The prefix to use for the bucket name"
  x-kubernetes-validations:
  - rule: self == oldSelf
    message: Value is immutable
```

## Computed fields
This project relies on the implicit conversion of Kubernetes Go CRD types to Terraform attributes. This is because generating code for explicit conversion is quite tricky.
In addition to `Null` values (absence of a value) [Terraform type system](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/terraform-concepts#type-system) also has `Unknown` values(that is not yet known). The value is `Unknown` if a Terraform attribute is marked as computed so it will be computed and set by the provider (or rather returned by the API that the provider calls). For example, if a Crossplane composition creates an AWS bucket, the bucket ARN will be known only after the bucket is created. The provider's job is to create the bucket, get the bucket ARN, and save it in the state.

But this creates some complications. If we try to populate the Go CRD type with the plan that has an attribute in the `Unknown` state, the implicit Terraform conversion will fail. To overcome this limitation the project relies on the partial plan retrieval. All computed `CRD fields/TF attributes` (with the exception, see below) must be defined under the `Status` field. Only the `Spec` field is implicitly converted from Terraform attributes to CRD Go types when the plan is read. The idea is that all user input goes under the `Spec` section and everything that is set by Kubernetes or its controllers (eg Crossplane) goes under the `Status` section. The `Status` section is not needed when an object is created in Kubernetes, so it is OK that it is not retrieved when the plan is read. After the object is created the provider retrieves the whole object including the `Status` field and saves it in the state.

The exception is computed fields that also have a default value set. In this case Terraform returns the default value when Terraform plan is converted to the CRD Go type. Therefore a field with a default value can be defined in the `Spec` section.

## Well known Crossplane CRD properties
Crossplane adds fields when it generates a CRD from XRD. The generator skips the next Crossplane-specific fields:
in `Spec`:
- compositeDeletePolicy
- compositionRef
- compositionRevisionRef
- compositionRevisionSelector
- compositionSelector
- compositionUpdatePolicy
- publishConnectionDetailsTo
- resourceRef
- writeConnectionSecretToRef

in `Status`:
- connectionDetails
- in `conditions` we only need `type` and `status`

## Crossplane delete operation
To properly handle the delete operation in Terraform, XRD [defaultCompositeDeletePolicy](https://docs.crossplane.io/v1.16/concepts/composite-resource-definitions/#defaultcompositedeletepolicy) should be set to `Foreground`. This causes Kubernetes to use foreground cascading deletion which deletes all child resources before deleting the parent resource. The claim controller waits for the composite deletion to finish before returning.

## Testing
Replace `/home/runner/go/bin` in `./tests/terraform-provider-crd/.terraformrc` with your absolute `go/bin` path. This is needed because `$HOME` interpolation does not work in the `provider_installation` block. Don't commit the change to the `.terraformrc` file.
```shell
kind create cluster
make test-local
kind delete cluster
```

## Acknowledgements
`tfpgen` is inspired by [terraform-provider-k8s](https://github.com/metio/terraform-provider-k8s)
