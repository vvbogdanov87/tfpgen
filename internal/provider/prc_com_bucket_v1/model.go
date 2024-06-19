package prc_com_bucket_v1

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type resourceModel struct {
	Name            types.String   `tfsdk:"name"`
	Timeouts        timeouts.Value `tfsdk:"timeouts"`
	ResourceVersion types.String   `tfsdk:"resource_version"`
	Prefix          types.String   `tfsdk:"prefix"`
	Arn             types.String   `tfsdk:"arn"`
}
