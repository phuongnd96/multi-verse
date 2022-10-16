package v1alpha1

import (
	"github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/sql/v1beta1"
)

type SQL struct {
	SQLInstances []SQLInstanceTemplateSpec `json:"sqlInstances,omitempty"`
	SQLDatabases []SQLDatabaseTemplate     `json:"sqlDatabases,omitempty"`
}

type SQLInstanceTemplateSpec struct {
	CustomObjectMeta `json:"customObjectMeta" protobuf:"bytes,1,opt,name=customObjectMeta"`
	Spec             v1beta1.SQLInstanceSpec `json:"spec,omitempty"`
}

type SQLDatabaseTemplate struct {
	CustomObjectMeta `json:"customObjectMeta" protobuf:"bytes,1,opt,name=customObjectMeta"`
	Spec             v1beta1.SQLDatabaseSpec `json:"spec,omitempty"`
}
