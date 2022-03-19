package config

import (
	"context"
	"fmt"
	"log"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/danielm-codefresh/argo-multi-cluster/api/clusterauth"
	"github.com/danielm-codefresh/argo-multi-cluster/api/common"
	// apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewClusterAddCommand() *cobra.Command {
	var (
		namespaces []string
	)
	cmd := &cobra.Command{
		Use: "add NAME [CONTEXT_NAME]",
		RunE: func(cmd *cobra.Command, args []string) error {
			startingConfig, err := clientcmd.NewDefaultPathOptions().GetStartingConfig()
			if err != nil {
				return err
			}

			contextName := args[0]
			clstContext := startingConfig.Contexts[contextName]
			if clstContext == nil {
				log.Fatalf("Context %s does not exist in kubeconfig", contextName)
			}

			overrides := clientcmd.ConfigOverrides{
				Context: *clstContext,
			}
			clientConfig := clientcmd.NewDefaultClientConfig(*startingConfig, &overrides)
			restConfig, err := clientConfig.ClientConfig()
			if err != nil {
				return err
			}

			// Install RBAC resources for managing the cluster
			clientset, err := kubernetes.NewForConfig(restConfig)
			if err != nil {
				return err
			}

			managerBearerToken, err := clusterauth.InstallClusterManagerRBAC(clientset, common.DefaultSystemNamespace, namespaces)
			if err != nil {
				return err
			}

			fmt.Printf("BearerToken: %s", managerBearerToken)
			
			cluster := clusterauth.NewCluster(contextName, restConfig, managerBearerToken)
			secret, err := clusterauth.ClusterToSecret(*cluster)
			if err != nil {
				return err
			}

			clientConfig = clientcmd.NewDefaultClientConfig(*startingConfig, nil)
			restConfig, err = clientConfig.ClientConfig()
			clientset, err = kubernetes.NewForConfig(restConfig)
			if err != nil {
				return err
			}

			_, err = clientset.CoreV1().Secrets(common.DefaultSystemNamespace).Create(context.Background(), secret, metav1.CreateOptions{})
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringArrayVar(&namespaces, "namespaces", nil, "List of namespaces which are allowed to manage")
	return cmd
}
