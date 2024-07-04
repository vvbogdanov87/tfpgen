package prc_com_bucket_v1

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type K8sCR struct {
	metav1.TypeMeta `tfsdk:"-" json:",inline"`
	Metadata        metav1.ObjectMeta `tfsdk:"-" json:"metadata,omitempty"`

	Name            types.String   `tfsdk:"name" json:"-"`
	Timeouts        timeouts.Value `tfsdk:"timeouts" json:"-"`
	ResourceVersion types.String   `tfsdk:"resource_version" json:"-"`

	Spec   *K8sSpec   `tfsdk:"spec" json:"spec,omitempty"`
	Status *K8sStatus `tfsdk:"status" json:"status"`
}

type K8sSpec struct {
	Mapstr map[string]string `tfsdk:"mapstr" json:"mapstr"`

	Strobj *struct {
		Prop1 string `tfsdk:"prop1" json:"prop1"`
		Prop2 string `tfsdk:"prop2" json:"prop2"`
	} `tfsdk:"strobj" json:"strobj"`

	Arrobj []struct {
		Arrprop1 string `tfsdk:"arrprop1" json:"arrprop1"`
		Arrprop2 string `tfsdk:"arrprop2" json:"arrprop2"`
	} `tfsdk:"arrobj" json:"arrobj"`

	Mapobj map[string]struct {
		Objprop1 string `tfsdk:"objprop1" json:"objprop1"`
		Objprop2 string `tfsdk:"objprop2" json:"objprop2"`
	} `tfsdk:"mapobj" json:"mapobj"`

	Arrstr []string `tfsdk:"arrstr" json:"arrstr"`

	Prefix string `tfsdk:"prefix" json:"prefix,omitempty"`
}

type K8sStatus struct {
	Arn *string `tfsdk:"arn" json:"arn"`

	Conditions *[]struct {
		Type   *string `tfsdk:"-" json:"type"`
		Status *string `tfsdk:"-" json:"status"`
	} `tfsdk:"-" json:"conditions"`
}
