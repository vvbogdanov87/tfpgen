package prc_com_bucket_v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const k8sApiVersion = "prc.com/v1"

type K8sCR struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   K8sSpec   `json:"spec,omitempty"`
	Status K8sStatus `json:"status"`
}

type K8sSpec struct {
	Prefix string `json:"prefix,omitempty"`
}

type K8sStatus struct {
	Arn *string `json:"arn"`

	Conditions *[]struct {
		Type   *string `json:"type"`
		Status *string `json:"status"`
	} `json:"conditions"`
}
