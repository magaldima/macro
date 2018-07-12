package v1alpha1

import (
	"github.com/magaldima/macro/pkg/apis/microservice"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SchemeGroupVersion is a group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: microservice.Group, Version: "v1alpha1"}

// SchemaGroupVersionKind is a group version kind used to attach owner references for microservices
var SchemaGroupVersionKind = schema.GroupVersionKind{Group: microservice.Group, Version: "v1alpha1", Kind: microservice.Kind}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes unqualified resource and returns Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	// SchemeBuilder is the builder for this scheme
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds this
	AddToScheme = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&MicroService{},
	)
	v1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
