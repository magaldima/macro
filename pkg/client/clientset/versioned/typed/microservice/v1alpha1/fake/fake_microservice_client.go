// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/magaldima/macro/pkg/client/clientset/versioned/typed/microservice/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeMicroV1alpha1 struct {
	*testing.Fake
}

func (c *FakeMicroV1alpha1) MicroServices(namespace string) v1alpha1.MicroServiceInterface {
	return &FakeMicroServices{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeMicroV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
