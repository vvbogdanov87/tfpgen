package prc_com_bucket_v1

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/vvbogdanov87/terraform-provider-crd/internal/provider/common"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	k8sSchema "k8s.io/apimachinery/pkg/runtime/schema"
	k8sTypes "k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/dynamic"

	"k8s.io/utils/ptr"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &tfResource{}
	_ resource.ResourceWithConfigure = &tfResource{}
)

// tfResource is the resource implementation.
type tfResource struct {
	client    dynamic.Interface
	namespace string
}

// NewTFResource is a helper function to simplify the provider implementation.
func NewTFResource() resource.Resource {
	return &tfResource{}
}

// Metadata returns the resource type name.
func (r *tfResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

// Schema defines the schema for the resource.
func (r *tfResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			// Fixed arguments
			"name": schema.StringAttribute{
				Required: true,
				Optional: false,
				Computed: false,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
			}),
			// Fixed attributes
			"resource_version": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			// Custom arguments
			"tags": schema.MapAttribute{
				Required:    false,
				Optional:    true,
				Computed:    false,
				Description: "Tags to apply to the bucket",
				ElementType: types.StringType,
			},
			"prefix": schema.StringAttribute{
				Required:    true,
				Optional:    false,
				Computed:    false,
				Description: "The prefix to use for the bucket name",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			// Computed attributes
			"arn": schema.StringAttribute{
				Computed:    true,
				Description: "ARN of the bucket",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *tfResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan bucketResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get timeout
	createTimeout, diags := plan.Timeouts.Create(ctx, 5*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert model to custom resource
	cr, diags := r.modelToCR(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

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
		Force:           ptr.To(true),
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
	plan.ResourceVersion = types.StringValue(cr.ResourceVersion)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *tfResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
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

	// Get custom resource from Kubernetes
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
	state.Tags, diags = types.MapValueFrom(ctx, types.StringType, cr.Spec.Tags)
	state.ResourceVersion = types.StringValue(cr.ResourceVersion)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *tfResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	// Convert model to custom resource
	cr, diag := r.modelToCR(ctx, &plan)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update resource
	body, err := json.Marshal(cr)
	if err != nil {
		resp.Diagnostics.AddError(
			"marshal resource",
			fmt.Sprintf("Error marshaling CRD:\n%s", err.Error()),
		)
		return
	}

	unstructuredObj := &unstructured.Unstructured{}
	err = unstructuredObj.UnmarshalJSON(body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unmarshal resource",
			fmt.Sprintf("Error unmarshaling CRD:\n%s", err.Error()),
		)
		return
	}

	_, err = r.client.
		Resource(k8sSchema.GroupVersionResource{Group: "prc.com", Version: "v1", Resource: "buckets"}).
		Namespace(cr.Namespace).
		Update(ctx, unstructuredObj, metav1.UpdateOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Update resource",
			fmt.Sprintf("Error updating resource: %s\nBody:\n%s", err.Error(), string(body)),
		)
		return
	}

	// After resource is updated, k8s controller changes Ready status to False while underliying resources(eg s3 bucket) are being updated.
	// It takes some time to controller to change Ready status to False so we need to wait for it.
	time.Sleep(500 * time.Millisecond)
	// wait for Rady status to be True
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
func (r *tfResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Get current state
	var state bucketResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get timeout
	deleteTimeout, diags := state.Timeouts.Delete(ctx, 5*time.Minute)
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

	// Wait for resource to be deleted
	err = retry.RetryContext(ctx, deleteTimeout, func() *retry.RetryError {
		_, err := r.client.
			Resource(k8sSchema.GroupVersionResource{Group: "prc.com", Version: "v1", Resource: "buckets"}).
			Namespace(r.namespace).
			Get(ctx, state.Name.ValueString(), metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return retry.NonRetryableError(fmt.Errorf("getting resource: %w", err))
		}
		return retry.RetryableError(fmt.Errorf("resource still exists"))
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Waiting resource deleted",
			fmt.Sprintf("Error waiting for resource deleted: %s", err.Error()),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *tfResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *tfResource) getResource(ctx context.Context, name string) (*K8sCR, error) {
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

func (r *tfResource) modelToCR(ctx context.Context, model *bucketResourceModel) (*K8sCR, diag.Diagnostics) {
	tagElements := make(map[string]types.String, len(model.Tags.Elements()))
	diags := model.Tags.ElementsAs(ctx, &tagElements, false)
	if diags.HasError() {
		return nil, diags
	}
	tags := make(map[string]string, len(tagElements))
	for k, v := range tagElements {
		tags[k] = v.ValueString()
	}

	return &K8sCR{
		TypeMeta: metav1.TypeMeta{
			APIVersion: k8sApiVersion,
			Kind:       "Bucket",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            model.Name.ValueString(),
			Namespace:       r.namespace,
			ResourceVersion: model.ResourceVersion.ValueString(),
		},
		Spec: K8sSpec{
			Prefix: model.Prefix.ValueString(),
			Tags:   tags,
		},
	}, diags
}

func (r *tfResource) waitReady(ctx context.Context, name string, timeout time.Duration) (*K8sCR, error) {
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
