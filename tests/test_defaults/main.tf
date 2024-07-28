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

resource "crd_default" "example" {
  timeouts = {
    create = "1m"
    update = "3m"
    delete = "2m"
    read   = "1m"
  }

  name = "dflt"
  spec = {
    prefix    = "asd"

    string_default_one = "ololo"
  }
}

output "arn" {
  value = crd_default.example.status.arn
}