package clusterauth

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/apimachinery/pkg/util/wait"
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

	if len(namespaces) == 0 {
		err = upsertClusterRole(clientset, ArgoManagerClusterRole, ArgoManagerClusterPolicyRules)
		if err != nil {
			return "", err
		}

		err = upsertClusterRoleBinding(clientset, ArgoManagerClusterRoleBinding, ArgoManagerClusterRole, rbacv1.Subject{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      ArgoManagerServiceAccount,
			Namespace: ns,
		})
		if err != nil {
			return "", err
		}
	} else {
		for _, namespace := range namespaces {
			err = upsertRole(clientset, ArgoManagerClusterRole, namespace, ArgoManagerNamespacePolicyRules)
			if err != nil {
				return "", err
			}

			err = upsertRoleBinding(clientset, ArgoManagerClusterRoleBinding, ArgoManagerClusterRole, namespace, rbacv1.Subject{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      ArgoManagerServiceAccount,
				Namespace: ns,
			})
			if err != nil {
				return "", err
			}
		}
	}

	return GetServiceAccountBearerToken(clientset, ns, ArgoManagerServiceAccount)
}


func upsert(kind string, name string, create func() (interface{}, error), update func() (interface{}, error)) error {
	_, err := create()
	if err != nil {
		if !apierr.IsAlreadyExists(err) {
			return fmt.Errorf("Failed to create %s %q: %v", kind, name, err)
		}
		_, err = update()
		if err != nil {
			return fmt.Errorf("Failed to update %s %q: %v", kind, name, err)
		}
		log.Infof("%s %q updated", kind, name)
	} else {
		log.Infof("%s %q created", kind, name)
	}
	return nil
}

func upsertClusterRole(clientset kubernetes.Interface, name string, rules []rbacv1.PolicyRule) error {
	clusterRole := rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Rules: rules,
	}
	return upsert("ClusterRole", name, func() (interface{}, error) {
		return clientset.RbacV1().ClusterRoles().Create(context.Background(), &clusterRole, metav1.CreateOptions{})
	}, func() (interface{}, error) {
		return clientset.RbacV1().ClusterRoles().Update(context.Background(), &clusterRole, metav1.UpdateOptions{})
	})
}

func upsertRole(clientset kubernetes.Interface, name string, namespace string, rules []rbacv1.PolicyRule) error {
	role := rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Rules: rules,
	}
	return upsert("Role", fmt.Sprintf("%s/%s", namespace, name), func() (interface{}, error) {
		return clientset.RbacV1().Roles(namespace).Create(context.Background(), &role, metav1.CreateOptions{})
	}, func() (interface{}, error) {
		return clientset.RbacV1().Roles(namespace).Update(context.Background(), &role, metav1.UpdateOptions{})
	})
}

func upsertClusterRoleBinding(clientset kubernetes.Interface, name string, clusterRoleName string, subject rbacv1.Subject) error {
	roleBinding := rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRoleName,
		},
		Subjects: []rbacv1.Subject{subject},
	}
	return upsert("ClusterRoleBinding", name, func() (interface{}, error) {
		return clientset.RbacV1().ClusterRoleBindings().Create(context.Background(), &roleBinding, metav1.CreateOptions{})
	}, func() (interface{}, error) {
		return clientset.RbacV1().ClusterRoleBindings().Update(context.Background(), &roleBinding, metav1.UpdateOptions{})
	})
}

func upsertRoleBinding(clientset kubernetes.Interface, name string, roleName string, namespace string, subject rbacv1.Subject) error {
	roleBinding := rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     roleName,
		},
		Subjects: []rbacv1.Subject{subject},
	}
	return upsert("RoleBinding", fmt.Sprintf("%s/%s", namespace, name), func() (interface{}, error) {
		return clientset.RbacV1().RoleBindings(namespace).Create(context.Background(), &roleBinding, metav1.CreateOptions{})
	}, func() (interface{}, error) {
		return clientset.RbacV1().RoleBindings(namespace).Update(context.Background(), &roleBinding, metav1.UpdateOptions{})
	})
}

// GetServiceAccountBearerToken will attempt to get the provided service account until it
// exists, iterate the secrets associated with it looking for one of type
// kubernetes.io/service-account-token, and return it's token if found.
func GetServiceAccountBearerToken(clientset kubernetes.Interface, ns string, sa string) (string, error) {
	var serviceAccount *corev1.ServiceAccount
	var secret *corev1.Secret
	var err error
	err = wait.Poll(500*time.Millisecond, 30*time.Second, func() (bool, error) {
		serviceAccount, err = clientset.CoreV1().ServiceAccounts(ns).Get(context.Background(), sa, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		// Scan all secrets looking for one of the correct type:
		for _, oRef := range serviceAccount.Secrets {
			var getErr error
			secret, err = clientset.CoreV1().Secrets(ns).Get(context.Background(), oRef.Name, metav1.GetOptions{})
			if err != nil {
				return false, fmt.Errorf("Failed to retrieve secret %q: %v", oRef.Name, getErr)
			}
			if secret.Type == corev1.SecretTypeServiceAccountToken {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		return "", fmt.Errorf("Failed to wait for service account secret: %v", err)
	}
	token, ok := secret.Data["token"]
	if !ok {
		return "", fmt.Errorf("Secret %q for service account %q did not have a token", secret.Name, serviceAccount)
	}
	return string(token), nil
}
