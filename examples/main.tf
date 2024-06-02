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
  name = "testbckt"
  prefix = "abc"
  tags = {
    key1 = "value1"
  }
}

output "arn" {
  value = crd_bucket.example.arn
}
