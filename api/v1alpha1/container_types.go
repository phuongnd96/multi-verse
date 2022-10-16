package v1alpha1

import (
	"github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/container/v1beta1"
)

type Container struct {
	ContainerClusters  []ContainerClusterTemplateSpec  `json:"containerClusters,omitempty"`
	ContainerNodePools []ContainerNodePoolTemplateSpec `json:"containerNodePools,omitempty"`
}

type ContainerClusterTemplateSpec struct {
	CustomObjectMeta `json:"customObjectMeta" protobuf:"bytes,1,opt,name=customObjectMeta"`
	Spec             v1beta1.ContainerClusterSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

type ContainerNodePoolTemplateSpec struct {
	CustomObjectMeta `json:"customObjectMeta" protobuf:"bytes,1,opt,name=customObjectMeta"`
	Spec             v1beta1.ContainerNodePoolSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}
