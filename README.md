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

| CRD/OpenAPI type                                                | GO type              | TF attribyte type                                  |
| --------------------------------------------------------------- | -------------------- | -------------------------------------------------- |
| string                                                          | string               | schema.StringAttribute                             |
| integer                                                         | int64                | schema.Int64Attribute                              |
| number                                                          | float64              | schema.Float64Attribute                            |
| boolean                                                         | boolean              | schema.BoolAttribute                               |
| `object` with `AdditionalProperties` and `Schema.Type = object` | map[string]struct    | schema.MapNestedAttribute                          |
| `object` with `AdditionalProperties`                            | map[string]primitive | schema.MapAttribute                                |
| `object` with Properties                                        | struct               | schema.SingleNestedAttribute                       |
| array with `Schema.Type = object`                               | []struct             | schema.ListNestedAttribute                         |
| array                                                           | []primitive          | schema.ListAttribute                               |

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

## Crossplane delete operation
To properly handle the delete operation in Terraform, XRD [defaultCompositeDeletePolicy](https://docs.crossplane.io/v1.16/concepts/composite-resource-definitions/#defaultcompositedeletepolicy) should be set to `Foreground`. This causes Kubernetes to use foreground cascading deletion which deletes all child resources before deleting the parent resource. The claim controller waits for the composite deletion to finish before returning.

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

## Testing
Replace `/home/runner/go/bin` in `./tests/terraform-provider-crd/.terraformrc` with your absolute `go/bin` path. This is needed because `$HOME` interpolation does not work in the `provider_installation` block. Don't commit the change to the `.terraformrc` file.
```shell
kind create cluster
make test-local
kind delete cluster
```

## Acknowledgements
`tfpgen` is inspired by [terraform-provider-k8s](https://github.com/metio/terraform-provider-k8s)
