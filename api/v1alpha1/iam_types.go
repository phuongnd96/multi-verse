package v1alpha1

import (
	"github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/iam/v1beta1"
)

type IAM struct {
	IAMServiceAccounts []IAMServiceAccountTemplateSpec `json:"serviceaccounts,omitempty"`
}

type IAMServiceAccountTemplateSpec struct {
	CustomObjectMeta `json:"customObjectMeta" protobuf:"bytes,1,opt,name=customObjectMeta"`
	Spec             v1beta1.IAMServiceAccountSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}
