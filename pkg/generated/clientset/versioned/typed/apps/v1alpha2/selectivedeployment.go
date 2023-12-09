/*
Copyright 2023 Contributors to the EdgeNet project.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1alpha2

import (
	"context"
	"time"

	v1alpha2 "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha2"
	scheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// SelectiveDeploymentsGetter has a method to return a SelectiveDeploymentInterface.
// A group's client should implement this interface.
type SelectiveDeploymentsGetter interface {
	SelectiveDeployments(namespace string) SelectiveDeploymentInterface
}

// SelectiveDeploymentInterface has methods to work with SelectiveDeployment resources.
type SelectiveDeploymentInterface interface {
	Create(ctx context.Context, selectiveDeployment *v1alpha2.SelectiveDeployment, opts v1.CreateOptions) (*v1alpha2.SelectiveDeployment, error)
	Update(ctx context.Context, selectiveDeployment *v1alpha2.SelectiveDeployment, opts v1.UpdateOptions) (*v1alpha2.SelectiveDeployment, error)
	UpdateStatus(ctx context.Context, selectiveDeployment *v1alpha2.SelectiveDeployment, opts v1.UpdateOptions) (*v1alpha2.SelectiveDeployment, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha2.SelectiveDeployment, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha2.SelectiveDeploymentList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha2.SelectiveDeployment, err error)
	SelectiveDeploymentExpansion
}

// selectiveDeployments implements SelectiveDeploymentInterface
type selectiveDeployments struct {
	client rest.Interface
	ns     string
}

// newSelectiveDeployments returns a SelectiveDeployments
func newSelectiveDeployments(c *AppsV1alpha2Client, namespace string) *selectiveDeployments {
	return &selectiveDeployments{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the selectiveDeployment, and returns the corresponding selectiveDeployment object, and an error if there is any.
func (c *selectiveDeployments) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha2.SelectiveDeployment, err error) {
	result = &v1alpha2.SelectiveDeployment{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("selectivedeployments").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of SelectiveDeployments that match those selectors.
func (c *selectiveDeployments) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha2.SelectiveDeploymentList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha2.SelectiveDeploymentList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("selectivedeployments").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested selectiveDeployments.
func (c *selectiveDeployments) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("selectivedeployments").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a selectiveDeployment and creates it.  Returns the server's representation of the selectiveDeployment, and an error, if there is any.
func (c *selectiveDeployments) Create(ctx context.Context, selectiveDeployment *v1alpha2.SelectiveDeployment, opts v1.CreateOptions) (result *v1alpha2.SelectiveDeployment, err error) {
	result = &v1alpha2.SelectiveDeployment{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("selectivedeployments").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(selectiveDeployment).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a selectiveDeployment and updates it. Returns the server's representation of the selectiveDeployment, and an error, if there is any.
func (c *selectiveDeployments) Update(ctx context.Context, selectiveDeployment *v1alpha2.SelectiveDeployment, opts v1.UpdateOptions) (result *v1alpha2.SelectiveDeployment, err error) {
	result = &v1alpha2.SelectiveDeployment{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("selectivedeployments").
		Name(selectiveDeployment.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(selectiveDeployment).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *selectiveDeployments) UpdateStatus(ctx context.Context, selectiveDeployment *v1alpha2.SelectiveDeployment, opts v1.UpdateOptions) (result *v1alpha2.SelectiveDeployment, err error) {
	result = &v1alpha2.SelectiveDeployment{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("selectivedeployments").
		Name(selectiveDeployment.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(selectiveDeployment).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the selectiveDeployment and deletes it. Returns an error if one occurs.
func (c *selectiveDeployments) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("selectivedeployments").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *selectiveDeployments) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("selectivedeployments").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched selectiveDeployment.
func (c *selectiveDeployments) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha2.SelectiveDeployment, err error) {
	result = &v1alpha2.SelectiveDeployment{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("selectivedeployments").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
