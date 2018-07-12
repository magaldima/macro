package common

import (
	"github.com/magaldima/macro/pkg/apis/microservice"
)

// MACRO CONTROLLER CONSTANTS
const (
	// MacroControllerConfigMapKey is the key in the configmap to retrieve microservice configuration from.
	// Content encoding is expected to be YAML.
	MacroControllerConfigMapKey = "config"

	// DefaultControllerNamespace is the default namespace where the controller is installed
	DefaultControllerNamespace = "default"

	// DefaultMacroControllerDeploymentName is the default deployment name of the macro controller
	DefaultMacroControllerDeploymentName = "macro-controller"

	//LabelKeyMacroControllerInstanceID is the label which allows to separate application among multiple running sensor controllers.
	LabelKeyMacroControllerInstanceID = microservice.FullName + "/controller-instanceid"

	// LabelKeyMicroservice is a label applied to microservice pods to indicate the name of the microservice
	LabelKeyMicroservice = microservice.FullName + "/name"

	// LabelKeyComplete is the label to mark microservices as complete
	LabelKeyComplete = microservice.FullName + "/complete"

	// EnvVarNamespace contains the namespace of the controller & jobs
	EnvVarNamespace = "SENSOR_NAMESPACE"

	// EnvVarConfigMap is the name of the configmap to use for the controller
	EnvVarConfigMap = "SENSOR_CONFIG_MAP"

	// EnvVarKubeConfig is the path to the Kubernetes configuration
	EnvVarKubeConfig = "KUBE_CONFIG"
)
