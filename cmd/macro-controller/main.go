package main

import (
	"context"
	"os"

	"github.com/magaldima/macro/common"
	"github.com/magaldima/macro/controller"
	microclientset "github.com/magaldima/macro/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

func main() {
	// kubernetes configuration
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}

	// controller configuration
	configMap, ok := os.LookupEnv(common.EnvVarConfigMap)
	if !ok {
		configMap = common.DefaultConfigMapName(common.DefaultMacroControllerDeploymentName)
	}

	kubeclientset := kubernetes.NewForConfigOrDie(restConfig)
	microclientset := microclientset.NewForConfigOrDie(restConfig)

	mc := controller.NewMacroController(restConfig, kubeclientset, microclientset, configMap)
	err = mc.ResyncConfig()
	if err != nil {
		panic(err)
	}

	go mc.Run(context.Background(), 1, 1)

	// Wait forever
	select {}
}
