terraform {
  required_providers {
    crd = {
      source = "registry.terraform.io/vvbogdanov87/crd"
    }
  }
}

provider "crd" {
  namespace = "test"
}

resource "crd_bucket" "example" {
  metadata = {
    name      = "testbckt"
  }

  spec = {
    prefix = "abc"
  }
}
