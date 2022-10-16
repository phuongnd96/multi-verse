package common

import "os"

var (
	NotCreatedStatus          = "NotReady"
	RunningStatus             = "Healthy"
	ErrorStatus               = "Error"
	CreatingStatus            = "Creating"
	ReconcilingStatus         = "Reconciling"
	ClusterServiceAccountName = "preview-cluster-sa"
	ExternalSecretSaId        = "external-secret@org-playground.iam.gserviceaccount.com"
	ResourcePatcherSaId       = os.Getenv("RESOURCE_PATCHER_SA_ID")
	VeleroSaId                = os.Getenv("VELERO_SA_ID")
	VeleroBackupBucket        = os.Getenv("VELERO_BACKUP_BUCKET")
	CloudFlareApiToken        = "CLOUDFLARE_API_TOKEN"
	CloudFlareDomainFilter    = "CLOUDFLARE_DOMAIN_FILTER"
	CloudFlareZoneIdFilter    = "CLOUDFLARE_ZONEID_FILTER"
)

const GlobalResourceTimeOutMinutes = 60
