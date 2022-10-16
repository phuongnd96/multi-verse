package v1alpha1

import (
	"github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/storage/v1beta1"
)

type Storage struct {
	StorageBuckets []StorageBucketTemplateSpec `json:"storageBuckets,omitempty"`
}

type StorageBucketTemplateSpec struct {
	CustomObjectMeta `json:"customObjectMeta" protobuf:"bytes,1,opt,name=customObjectMeta"`
	Spec             v1beta1.StorageBucketSpec `json:"spec,omitempty"`
}
