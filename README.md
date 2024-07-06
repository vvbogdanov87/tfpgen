# terraform-provider-crd

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