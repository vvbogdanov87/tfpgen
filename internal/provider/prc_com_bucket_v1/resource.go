package prc_com_bucket_v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &bucketResource{}
	_ resource.ResourceWithConfigure = &bucketResource{}
)

// bucketResource is the resource implementation.
type bucketResource struct {
	client *kubernetes.Clientset
}

// NewBucketResource is a helper function to simplify the provider implementation.
func NewBucketResource() resource.Resource {
	return &bucketResource{}
}

// Metadata returns the resource type name.
func (r *bucketResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

// Schema defines the schema for the resource.
func (r *bucketResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				Required: true,
				Optional: false,
				Computed: false,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required: true,
						Optional: false,
						Computed: false,
					},

					"namespace": schema.StringAttribute{
						Required: true,
						Optional: false,
						Computed: false,
					},
				},
			},
			"spec": schema.SingleNestedAttribute{
				Required: false,
				Optional: true,
				Computed: false,
				Attributes: map[string]schema.Attribute{
					"prefix": schema.StringAttribute{
						Required: true,
						Optional: false,
						Computed: false,
					},
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *bucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan bucketResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	manifest := &Buckets{
		TypeMeta: v1.TypeMeta{
			APIVersion: BucketsApi,
			Kind:       "Bucket",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: plan.Metadata.Name.ValueString(),
		},
		Spec: BucketsSpec{
			Prefix: plan.Spec.Prefix.ValueString(),
		},
	}

	// Create new resource
	path := fmt.Sprintf("/apis/%s/namespaces/%s/%s", BucketsApi, plan.Metadata.Namespace.ValueString(), "Buckets")
	path = strings.ToLower(path)
	body, err := json.Marshal(manifest)
	if err != nil {
		resp.Diagnostics.AddError(
			"marshal resource",
			fmt.Sprintf("Error marshaling CRD:\n%s", err.Error()),
		)
		return
	}
	_, err = r.client.RESTClient().Post().AbsPath(path).Body(body).DoRaw(context.TODO())
	if err != nil {
		resp.Diagnostics.AddError(
			"Create resource",
			fmt.Sprintf("Error creating resource \"%s\":\n%s\nBody:\n%s", path, err.Error(), string(body)),
		)
		return
	}

	// TODO: populate computed values (status)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *bucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state bucketResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get resource from Kubernetes
	path := fmt.Sprintf("/apis/%s/namespaces/%s/%s/%s", BucketsApi, state.Metadata.Namespace.ValueString(), "Buckets", state.Metadata.Name.ValueString())
	path = strings.ToLower(path)
	body, err := r.client.RESTClient().Get().AbsPath(path).DoRaw(context.TODO())
	if err != nil {
		resp.Diagnostics.AddError(
			"Get resource",
			fmt.Sprintf("Error getting resource from Kubernetes \"%s\":\n%s", path, err.Error()),
		)
		return
	}
	var manifest Buckets
	err = json.Unmarshal(body, &manifest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unmarshal resource",
			fmt.Sprintf("Error unmarshaling CRD:\n%s", err.Error()),
		)
		return
	}

	// Overwrite current state with refreshed state
	state.Spec.Prefix = types.StringValue(manifest.Spec.Prefix)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *bucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *bucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

// Configure adds the provider configured client to the resource.
func (r *bucketResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*kubernetes.Clientset)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *kubernetes.Clientset, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}
