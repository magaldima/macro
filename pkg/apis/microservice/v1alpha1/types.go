package v1alpha1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MicroStatus is a label for the condition of a node at the current time.
type MicroStatus string

// Microservice and node statuses
const (
	NodeRunning MicroStatus = "Running"
	NodeFailed  MicroStatus = "Failed"
	NodeError   MicroStatus = "Error"
)

// MicroService is the definition of a microservice
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MicroService struct {
	v1.TypeMeta   `json:",inline"`
	v1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec          ServiceSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status        ServiceStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

// MicroServiceList is the list of microservices
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MicroServiceList struct {
	v1.TypeMeta `json:",inline"`
	v1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items       []MicroService `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// ServiceSpec contains the specification for a microservice
type ServiceSpec struct {
	Endpoints []*Endpoint `json:"endpoints" protobuf:"bytes,1,rep,name=endpoints"`
	Nodes     []*Node     `json:"nodes" protobuf:"bytes,2,rep,name=nodes"`
}

//
type Node struct {
	ID       string            `json:"id"`
	Address  string            `json:"address"`
	Port     int               `json:"port"`
	Metadata map[string]string `json:"metadata"`
}

type Endpoint struct {
	Name     string            `json:"name"`
	Metadata map[string]string `json:"metadata"`
}

// ServiceStatus contains the overall status information about a microservice
type ServiceStatus struct {
	// Status of the service
	Status MicroStatus `json:"phase,omitempty" protobuf:"bytes,1,opt,name=status"`

	// Time at which this microservice started
	StartedAt v1.Time `json:"startedAt,omitempty" protobuf:"bytes,2,opt,name=startedAt"`

	// Time at which this microservice was decommissioned
	DecommissionedAt v1.Time `json:"decommissionedAt,omitempty" protobuf:"bytes,2,opt,name=decommissionedAt"`

	// A human readable message indicating details about why the node is in this condition.
	Message string `json:"message,omitempty" protobuf:"bytes,3,opt,name=message"`

	// Nodes is a mapping between a node ID and the node's status.
	Nodes map[string]NodeStatus `json:"nodes,omitempty" protobuf:"bytes,4,rep,name=nodes"`
}

// NodeStatus contains status information about an individual node in the microservice
type NodeStatus struct {
	// ID is a unique identifier of a node within the microservice
	// It is implemented as a hash of the node name, which makes the ID deterministic
	ID string `json:"id" protobuf:"bytes,1,opt,name=id"`

	// Name is unique name in the node tree used to generate the node ID
	Name string `json:"name" protobuf:"bytes,2,opt,name=name"`

	// Time at which this node started
	StartedAt v1.Time `json:"startedAt,omitempty" protobuf:"bytes,3,opt,name=startedAt"`
}
