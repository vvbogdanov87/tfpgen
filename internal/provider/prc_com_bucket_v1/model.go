package prc_com_bucket_v1

type bucketResourceModel struct {
	ApiVersion *string `tfsdk:"-" json:"apiVersion"`
	Kind       *string `tfsdk:"-" json:"kind"`

	Metadata struct {
		Name      string `tfsdk:"name" json:"name"`
		Namespace string `tfsdk:"namespace" json:"namespace"`
	} `tfsdk:"metadata" json:"metadata"`

	Spec *struct {
		Prefix *string `tfsdk:"prefix" json:"prefix,omitempty"`
	} `tfsdk:"spec" json:"spec,omitempty"`
}
