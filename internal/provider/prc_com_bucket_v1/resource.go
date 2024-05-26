package prc_com_bucket_v1

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/vvbogdanov87/terraform-provider-crd/internal/provider/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sSchema "k8s.io/apimachinery/pkg/runtime/schema"
	k8sTypes "k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/dynamic"

	"k8s.io/utils/pointer"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &bucketResource{}
	_ resource.ResourceWithConfigure = &bucketResource{}
)

// bucketResource is the resource implementation.
type bucketResource struct {
	client    dynamic.Interface
	namespace string
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
func (r *bucketResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
				Optional: false,
				Computed: false,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"prefix": schema.StringAttribute{
				Required: true,
				Optional: false,
				Computed: false,
			},
			"arn": schema.StringAttribute{

				Computed: true,
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Update: true,
			}),
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
		resp.Diagnostics.AddError(
			"Get state in create", "Error getting state in create",
		)
		return
	}

	// Get timeout
	createTimeout, diags := plan.Timeouts.Create(ctx, 5*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cr := r.modelToCR(&plan)

	// Create new resource
	body, err := json.Marshal(cr)
	if err != nil {
		resp.Diagnostics.AddError(
			"marshal resource",
			fmt.Sprintf("Error marshaling CRD:\n%s", err.Error()),
		)
		return
	}

	patchOptions := metav1.PatchOptions{
		FieldManager:    "terraform-provider-crd",
		Force:           pointer.Bool(true),
		FieldValidation: "Strict",
	}

	_, err = r.client.
		Resource(k8sSchema.GroupVersionResource{Group: "prc.com", Version: "v1", Resource: "buckets"}).
		Namespace(cr.Namespace).
		Patch(ctx, cr.Name, k8sTypes.ApplyPatchType, body, patchOptions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Create resource",
			fmt.Sprintf("Error creating resource: %s\nBody:\n%s", err.Error(), string(body)),
		)
		return
	}

	// wait for resource becomes READY
	cr, err = r.waitReady(ctx, plan.Name.ValueString(), createTimeout)
	if err != nil {
		resp.Diagnostics.AddError(
			"Waiting resource READY",
			fmt.Sprintf("Error waiting for resource READY state: %s", err.Error()),
		)
		return
	}

	// Set computed values
	plan.Arn = types.StringValue(*cr.Status.Arn)

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
		resp.Diagnostics.AddError(
			"Get state in read", "Error getting state in read",
		)
		return
	}

	// Get resource from Kubernetes
	cr, err := r.getResource(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Get resource",
			fmt.Sprintf("Error getting resource:\n%s", err.Error()),
		)
		return
	}

	// Overwrite current state with refreshed state
	state.Prefix = types.StringValue(cr.Spec.Prefix)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *bucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan bucketResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get timeout
	updateTimeout, diags := plan.Timeouts.Update(ctx, 5*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cr := r.modelToCR(&plan)

	// Update resource
	body, err := json.Marshal(cr)
	if err != nil {
		resp.Diagnostics.AddError(
			"marshal resource",
			fmt.Sprintf("Error marshaling CRD:\n%s", err.Error()),
		)
		return
	}

	patchOptions := metav1.PatchOptions{
		FieldManager:    "terraform-provider-crd",
		Force:           pointer.Bool(true),
		FieldValidation: "Strict",
	}

	_, err = r.client.
		Resource(k8sSchema.GroupVersionResource{Group: "prc.com", Version: "v1", Resource: "buckets"}).
		Namespace(cr.Namespace).
		Patch(ctx, cr.Name, k8sTypes.ApplyPatchType, body, patchOptions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Update resource",
			fmt.Sprintf("Error updating resource: %s\nBody:\n%s", err.Error(), string(body)),
		)
		return
	}

	// wait for resource becomes READY
	cr, err = r.waitReady(ctx, plan.Name.ValueString(), updateTimeout)
	if err != nil {
		resp.Diagnostics.AddError(
			"Waiting resource READY",
			fmt.Sprintf("Error waiting for resource READY state: %s", err.Error()),
		)
		return
	}
	if cr == nil {
		resp.Diagnostics.AddError(
			"Get ARN",
			fmt.Sprintf("name: %s, timeout: %s", plan.Name.ValueString(), updateTimeout.String()),
		)
		return
	}
	// Set computed values
	plan.Arn = types.StringValue(*cr.Status.Arn)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *bucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Get current state
	var state bucketResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete resource
	fg := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: &fg,
	}

	err := r.client.
		Resource(k8sSchema.GroupVersionResource{Group: "prc.com", Version: "v1", Resource: "buckets"}).
		Namespace(r.namespace).
		Delete(ctx, state.Name.ValueString(), deleteOptions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Delete resource",
			fmt.Sprintf("Error deleting resource: %s", err.Error()),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *bucketResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	pd, ok := req.ProviderData.(common.ResourceData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *kubernetes.Clientset, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = pd.Clientset
	r.namespace = pd.Namespace
}

func (r *bucketResource) getResource(ctx context.Context, name string) (*K8sCR, error) {
	getResponse, err := r.client.
		Resource(k8sSchema.GroupVersionResource{Group: "prc.com", Version: "v1", Resource: "buckets"}).
		Namespace(r.namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	body, err := getResponse.MarshalJSON()
	if err != nil {
		return nil, err
	}

	var manifest K8sCR
	err = json.Unmarshal(body, &manifest)
	if err != nil {
		return nil, err
	}

	return &manifest, nil
}

func (r *bucketResource) modelToCR(model *bucketResourceModel) *K8sCR {
	return &K8sCR{
		TypeMeta: metav1.TypeMeta{
			APIVersion: k8sApiVersion,
			Kind:       "Bucket",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      model.Name.ValueString(),
			Namespace: r.namespace,
		},
		Spec: K8sSpec{
			Prefix: model.Prefix.ValueString(),
		},
	}
}

func (r *bucketResource) waitReady(ctx context.Context, name string, timeout time.Duration) (*K8sCR, error) {
	var cr *K8sCR
	err := retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		var err error
		cr, err = r.getResource(ctx, name)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("getting resource: %w", err))
		}

		if cr.Status.Conditions == nil {
			return retry.RetryableError(fmt.Errorf("resource doesn't have 'conditions' field"))
		}

		for _, condition := range *cr.Status.Conditions {
			if *condition.Type == "Ready" && *condition.Status == "True" {
				return nil
			}
		}
		return retry.RetryableError(fmt.Errorf("resource is not READY"))
	})
	return cr, err
}
