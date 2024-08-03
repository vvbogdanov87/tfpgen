terraform {
  required_providers {
    crd = {
      source = "registry.terraform.io/vvbogdanov87/crd"
    }
  }
}

provider "crd" {
  namespace = "default"
}

resource "crd_enum" "example" {
  timeouts = {
    create = "1m"
    update = "3m"
    delete = "2m"
    read   = "1m"
  }

  name = "enm"
  spec = {
    prefix    = "asd"

    str_prop = "ololo" # wrong value, "terraform apply" is expected to fail with "Error: Invalid Attribute Value Match"
  }
}

output "arn" {
  value = crd_enum.example.status.arn
}