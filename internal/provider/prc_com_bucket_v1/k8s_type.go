package prc_com_bucket_v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const BucketsApi = "prc.com/v1"

type Buckets struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec BucketsSpec `json:"spec,omitempty"`
}

type BucketsSpec struct {
	Prefix string `json:"prefix"`
}
