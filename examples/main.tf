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
  spec = {
    prefix = "asd"
    mapstr = {
      "key1" = "value1"
      "key2" = "value2"
    }
    mapobj = {
      "key1" = {
        "objprop1" = "value1"
        "objprop2" = "value2"
      }
      "key2" = {
        "objprop1" = "value1"
        "objprop2" = "value2"
      }
    }
    strobj = {
      prop1 = "value1"
      prop2 = "value2"
    }
    arrstr = ["value1", "value2"]
    arrobj = [
      {
        arrprop1 = "value1"
        arrprop2 = "value2"
      },
      {
        arrprop1 = "value1"
        arrprop2 = "value2"
      }
    ]
  }
}

output "arn" {
  value = crd_bucket.example.status.arn
}