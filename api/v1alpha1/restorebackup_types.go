package v1alpha1

type SQLRestore struct {
	SQLRestores []SQLRestoresTemplateSpec `json:"sqlRestores,omitempty"`
}

type SQLRestoresTemplateSpec struct {
	CustomObjectMeta `json:"customObjectMeta" protobuf:"bytes,1,opt,name=customObjectMeta"`
	Spec             SQLRestoreSpec `json:"spec,omitempty"`
}

type SQLRestoreSpec struct {
	// BackupRunId: The ID of the backup run to restore from.
	BackupRunId int64 `json:"backupRunId,omitempty,string"`

	// InstanceId: The ID of the instance that the backup was taken from.
	InstanceId string `json:"instanceId,omitempty"`

	// Kind: This is always `sql#restoreBackupContext`.
	Kind string `json:"kind,omitempty"`

	// Project: The full project ID of the source instance.
	Project string `json:"project,omitempty"`
}
