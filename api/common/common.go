package common

const (
	DefaultSystemNamespace = "kube-system"

	// LabelKeySecretType contains the type of argo secret
	LabelKeySecretType = "argo.argoproj.io/secret-type"
	// LabelValueSecretTypeCluster indicates a secret type of cluster
	LabelValueSecretTypeCluster = "cluster"
)
