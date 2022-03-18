package clusterauth

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	log "github.com/sirupsen/logrus"
)


// ArgoManagerServiceAccount is the name of the service account for managing a cluster
const (
	ArgoManagerServiceAccount     = "argo-manager"
	ArgoManagerClusterRole        = "argo-manager-role"
	ArgoManagerClusterRoleBinding = "argo-manager-role-binding"
)

// ArgoManagerPolicyRules are the policies to give argo-manager
var ArgoManagerClusterPolicyRules = []rbacv1.PolicyRule{
	{
		APIGroups: []string{"*"},
		Resources: []string{"*"},
		Verbs:     []string{"*"},
	},
	{
		NonResourceURLs: []string{"*"},
		Verbs:           []string{"*"},
	},
}

// ArgoManagerNamespacePolicyRules are the namespace level policies to give argo-manager
var ArgoManagerNamespacePolicyRules = []rbacv1.PolicyRule{
	{
		APIGroups: []string{"*"},
		Resources: []string{"*"},
		Verbs:     []string{"*"},
	},
}

// CreateServiceAccount creates a service account in a given namespace
func CreateServiceAccount(
	clientset kubernetes.Interface,
	serviceAccountName string,
	namespace string,
) error {
	serviceAccount := corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: namespace,
		},
	}
	_, err := clientset.CoreV1().ServiceAccounts(namespace).Create(context.Background(), &serviceAccount, metav1.CreateOptions{})
	if err != nil {
		if !apierr.IsAlreadyExists(err) {
			return fmt.Errorf("Failed to create service account %q in namespace %q: %v", serviceAccountName, namespace, err)
		}
		log.Infof("ServiceAccount %q already exists in namespace %q", serviceAccountName, namespace)
		return nil
	}
	log.Infof("ServiceAccount %q created in namespace %q", serviceAccountName, namespace)
	return nil
}

// InstallClusterManagerRBAC installs RBAC resources for a cluster manager to operate a cluster. Returns a token
func InstallClusterManagerRBAC(clientset kubernetes.Interface, ns string, namespaces []string) (string, error) {

	err := CreateServiceAccount(clientset, ArgoManagerServiceAccount, ns)
	if err != nil {
		return "", err
	}

	// if len(namespaces) == 0 {
	// 	err = upsertClusterRole(clientset, ArgoCDManagerClusterRole, ArgoCDManagerClusterPolicyRules)
	// 	if err != nil {
	// 		return "", err
	// 	}
	//
	// 	err = upsertClusterRoleBinding(clientset, ArgoCDManagerClusterRoleBinding, ArgoCDManagerClusterRole, rbacv1.Subject{
	// 		Kind:      rbacv1.ServiceAccountKind,
	// 		Name:      ArgoCDManagerServiceAccount,
	// 		Namespace: ns,
	// 	})
	// 	if err != nil {
	// 		return "", err
	// 	}
	// } else {
	// 	for _, namespace := range namespaces {
	// 		err = upsertRole(clientset, ArgoCDManagerClusterRole, namespace, ArgoCDManagerNamespacePolicyRules)
	// 		if err != nil {
	// 			return "", err
	// 		}
	//
	// 		err = upsertRoleBinding(clientset, ArgoCDManagerClusterRoleBinding, ArgoCDManagerClusterRole, namespace, rbacv1.Subject{
	// 			Kind:      rbacv1.ServiceAccountKind,
	// 			Name:      ArgoCDManagerServiceAccount,
	// 			Namespace: ns,
	// 		})
	// 		if err != nil {
	// 			return "", err
	// 		}
	// 	}
	// }

	// return GetServiceAccountBearerToken(clientset, ns, ArgoCDManagerServiceAccount)
	return "", nil
}
