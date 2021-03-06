// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/magaldima/macro/pkg/apis/microservice/v1alpha1"
	scheme "github.com/magaldima/macro/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// MicroServicesGetter has a method to return a MicroServiceInterface.
// A group's client should implement this interface.
type MicroServicesGetter interface {
	MicroServices(namespace string) MicroServiceInterface
}

// MicroServiceInterface has methods to work with MicroService resources.
type MicroServiceInterface interface {
	Create(*v1alpha1.MicroService) (*v1alpha1.MicroService, error)
	Update(*v1alpha1.MicroService) (*v1alpha1.MicroService, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.MicroService, error)
	List(opts v1.ListOptions) (*v1alpha1.MicroServiceList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.MicroService, err error)
	MicroServiceExpansion
}

// microServices implements MicroServiceInterface
type microServices struct {
	client rest.Interface
	ns     string
}

// newMicroServices returns a MicroServices
func newMicroServices(c *MicroV1alpha1Client, namespace string) *microServices {
	return &microServices{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the microService, and returns the corresponding microService object, and an error if there is any.
func (c *microServices) Get(name string, options v1.GetOptions) (result *v1alpha1.MicroService, err error) {
	result = &v1alpha1.MicroService{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("microservices").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of MicroServices that match those selectors.
func (c *microServices) List(opts v1.ListOptions) (result *v1alpha1.MicroServiceList, err error) {
	result = &v1alpha1.MicroServiceList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("microservices").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested microServices.
func (c *microServices) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("microservices").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a microService and creates it.  Returns the server's representation of the microService, and an error, if there is any.
func (c *microServices) Create(microService *v1alpha1.MicroService) (result *v1alpha1.MicroService, err error) {
	result = &v1alpha1.MicroService{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("microservices").
		Body(microService).
		Do().
		Into(result)
	return
}

// Update takes the representation of a microService and updates it. Returns the server's representation of the microService, and an error, if there is any.
func (c *microServices) Update(microService *v1alpha1.MicroService) (result *v1alpha1.MicroService, err error) {
	result = &v1alpha1.MicroService{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("microservices").
		Name(microService.Name).
		Body(microService).
		Do().
		Into(result)
	return
}

// Delete takes name of the microService and deletes it. Returns an error if one occurs.
func (c *microServices) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("microservices").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *microServices) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("microservices").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched microService.
func (c *microServices) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.MicroService, err error) {
	result = &v1alpha1.MicroService{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("microservices").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
