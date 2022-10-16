package v1alpha1

import (
	"github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/compute/v1beta1"
)

type Compute struct {
	ComputeNetworks    []ComputeNetworkTemplateSpec    `json:"computeNetworks,omitempty"`
	ComputeSubnetworks []ComputeSubnetworkTemplateSpec `json:"computeSubNetworks,omitempty"`
	ComputeRouters     []ComputeRouterTemplateSpec     `json:"computeRouters,omitempty"`
	ComputeRouterNATs  []ComputeRouterNATTemplateSpec  `json:"computeRouterNATs,omitempty"`
}

type ComputeNetworkTemplateSpec struct {
	CustomObjectMeta `json:"customObjectMeta" protobuf:"bytes,1,opt,name=customObjectMeta"`
	Spec             v1beta1.ComputeNetworkSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

type ComputeSubnetworkTemplateSpec struct {
	CustomObjectMeta `json:"customObjectMeta" protobuf:"bytes,1,opt,name=customObjectMeta"`
	Spec             v1beta1.ComputeSubnetworkSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

type ComputeRouterTemplateSpec struct {
	CustomObjectMeta `json:"customObjectMeta" protobuf:"bytes,1,opt,name=customObjectMeta"`
	Spec             v1beta1.ComputeRouterSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

type ComputeRouterNATTemplateSpec struct {
	CustomObjectMeta `json:"customObjectMeta" protobuf:"bytes,1,opt,name=customObjectMeta"`
	Spec             v1beta1.ComputeRouterNATSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}
