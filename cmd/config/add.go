package config

import (
	// context "context"
	// "fmt"
	// "github.com/argoproj-labs/multi-cluster-kubernetes/api/config"
	"context"
	"fmt"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/danielm-codefresh/argo-multi-cluster/api/clusterauth"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	// "k8s.io/client-go/kubernetes"
	// "k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	// "k8s.io/client-go/util/homedir"
	// "path/filepath"
)

// func NewAddCommand() *cobra.Command {
// 	var (
// 		kubeconfig string
// 		namespace  string
// 	)
// 	cmd := &cobra.Command{
// 		Use: "add NAME [CONTEXT_NAME]",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			ctx := context.Background()
//
// 			startingConfig, err := clientcmd.NewDefaultPathOptions().GetStartingConfig()
// 			if err != nil {
// 				return err
// 			}
//
// 			name := args[0]
// 			if len(args) == 2 {
// 				startingConfig.CurrentContext = args[1]
// 			}
//
// 			clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}, &clientcmd.ConfigOverrides{})
// 			restConfig, err := clientConfig.ClientConfig()
// 			if err != nil {
// 				return err
// 			}
//
// 			// secretsInterface := kubernetes.NewForConfigOrDie(restConfig).CoreV1().Secrets("").List(metav1.ListOptions{LabelSelector: "argo-secret-type=cluster"})
// 			secretsInterface := kubernetes.NewForConfigOrDie(restConfig).CoreV1().Secrets("")
//
// 			err = clientcmdapi.MinifyConfig(startingConfig)
// 			if err != nil {
// 				return err
// 			}
// 			err = config.New(secretsInterface).Add(ctx, name, startingConfig)
// 			if err != nil {
// 				return err
// 			}
//
// 			fmt.Printf("config %q from context %q added\n", name, startingConfig.CurrentContext)
//
// 			return nil
// 		},
// 	}
// 	// cmd.Flags().StringVar(&kubeconfig, "kubeconfig", filepath.Join(homedir.HomeDir(), ".kube", "config"), "(optional) absolute path to the kubeconfig file")
// 	// cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace")
// 	return cmd
// }


func NewClusterAddCommand() *cobra.Command {
	var (
		kubeconfig string
		namespace  string
	)
	cmd := &cobra.Command{
		Use: "add NAME [CONTEXT_NAME]",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			startingConfig, err := clientcmd.NewDefaultPathOptions().GetStartingConfig()
			if err != nil {
				return err
			}

			name := args[0]
			if len(args) == 2 {
				startingConfig.CurrentContext = args[1]
			}

			clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}, &clientcmd.ConfigOverrides{})
			restConfig, err := clientConfig.ClientConfig()
			if err != nil {
				return err
			}

			// Install RBAC resources for managing the cluster
			clientset, err := kubernetes.NewForConfig(restConfig)
			errors.CheckError(err)
			managerBearerToken, err := clusterauth.InstallClusterManagerRBAC(clientset, clusterOpts.SystemNamespace, clusterOpts.Namespaces)
			errors.CheckError(err)

			fmt.Printf("config %q from context %q added\n", name, startingConfig.CurrentContext)

			return nil
		},
	}
	// cmd.Flags().StringVar(&kubeconfig, "kubeconfig", filepath.Join(homedir.HomeDir(), ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	// cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace")
	return cmd
}

// AskToProceed prompts the user with a message (typically a yes or no question) and returns whether
// or not they responded in the affirmative or negative.
func AskToProceed(message string) bool {
	for {
		fmt.Print(message)
		reader := bufio.NewReader(os.Stdin)
		proceedRaw, err := reader.ReadString('\n')
		errors.CheckError(err)
		switch strings.ToLower(strings.TrimSpace(proceedRaw)) {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		}
	}
}
