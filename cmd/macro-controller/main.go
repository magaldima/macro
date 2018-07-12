package main

import (
	"context"
	"os"

	"github.com/magaldima/macro/common"
	"github.com/magaldima/macro/controller"
	microclientset "github.com/magaldima/macro/pkg/client/clientset/versioned"
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

	microclientset := microclientset.NewForConfigOrDie(config)

	wfController := controller.NewMacroController(config, kubeclientset, microclientset, rootArgs.configMap)
	err = controller.ResyncConfig()
	if err != nil {
		panic(err)
	}

	go controller.Run(context.Background(), 1, 1)

	// Wait forever
	select {}
}
