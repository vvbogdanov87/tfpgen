package prc_com_bucket_v1

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type bucketResourceModel struct {
	Name            types.String   `tfsdk:"name"`
	Prefix          types.String   `tfsdk:"prefix"`
	Tags            types.Map      `tfsdk:"tags"`
	Arn             types.String   `tfsdk:"arn"`
	Timeouts        timeouts.Value `tfsdk:"timeouts"`
	ResourceVersion types.String   `tfsdk:"resource_version"`
}
