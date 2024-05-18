terraform {
  required_providers {
    crd = {
      source = "registry.terraform.io/vvbogdanov87/crd"
    }
  }
}

provider "crd" {}

resource "crd_bucket" "example" {
  metadata = {
    name      = "testbckt"
    namespace = "test"
  }

  spec = {
    prefix = "abc"
  }
}
