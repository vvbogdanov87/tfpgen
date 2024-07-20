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
    # test primitive types
    prefix    = "asd"
    int_prop  = 42
    num_prop  = 3.14
    bool_prop = true
    # test map[string]primitive types
    map_str = {
      "key1" = "value1"
      "key2" = "value2"
    }
    map_int = {
      "key1" = 1
      "key2" = 2
    }
    map_num = {
      "key1" = 1.1
      "key2" = 2.2
    }
    map_bool = {
      "key1" = true
      "key2" = false
    }
    # test [map]object types
    map_obj = {
      "key1" = {
        "objprop1" = "value1"
        "objprop2" = "value2"
      }
      "key2" = {
        "objprop1" = "value1"
        "objprop2" = "value2"
      }
    }
    # test object types
    obj_str = {
      prop1 = "value1"
      prop2 = "value2"
    }
    # test primitive array types
    arr_str  = ["value1", "value2"]
    arr_int  = [1, 2]
    arr_num  = [1.1, 2.2]
    arr_bool = [true, false]
    # test array of object types
    arr_obj = [
      {
        arrprop1 = "value1"
        arrprop2 = "value2"
      },
      {
        arrprop1 = "value1"
        arrprop2 = "value2"
      }
    ]
    environment_configs = [
      {
        ref = {
          name = "someRef"
        }
        selector = {
          match_labels = [
            {
              from_field_path_policy = "Required"
              type                   = "Value"
              value                  = "someValue"
            },
            {
              from_field_path_policy = "Optional"
              type                   = "Value"
              value                  = "someValue2"
            }
          ]
          max_match = 2
          mode = "Single"
          sort_by_field_path = "metadata.name"
        }
        type = "Reference"
      }
    ]
  }
}

output "arn" {
  value = crd_bucket.example.status.arn
}
