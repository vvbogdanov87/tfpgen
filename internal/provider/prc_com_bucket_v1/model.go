package prc_com_bucket_v1

import "github.com/hashicorp/terraform-plugin-framework/types"

// bucketResourceModel maps the resource schema data.
type bucketResourceModel struct {
	Metadata metadataModel `tfsdk:"metadata"`
	Spec     bucketSpec    `tfsdk:"spec"`
}

type metadataModel struct {
	Name      types.String `tfsdk:"name"`
	Namespace types.String `tfsdk:"namespace"`
}

type bucketSpec struct {
	Prefix types.String `tfsdk:"prefix"`
}
