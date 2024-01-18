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

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha1"
	scheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ClusterRoleRequestsGetter has a method to return a ClusterRoleRequestInterface.
// A group's client should implement this interface.
type ClusterRoleRequestsGetter interface {
	ClusterRoleRequests() ClusterRoleRequestInterface
}

// ClusterRoleRequestInterface has methods to work with ClusterRoleRequest resources.
type ClusterRoleRequestInterface interface {
	Create(ctx context.Context, clusterRoleRequest *v1alpha1.ClusterRoleRequest, opts v1.CreateOptions) (*v1alpha1.ClusterRoleRequest, error)
	Update(ctx context.Context, clusterRoleRequest *v1alpha1.ClusterRoleRequest, opts v1.UpdateOptions) (*v1alpha1.ClusterRoleRequest, error)
	UpdateStatus(ctx context.Context, clusterRoleRequest *v1alpha1.ClusterRoleRequest, opts v1.UpdateOptions) (*v1alpha1.ClusterRoleRequest, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.ClusterRoleRequest, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.ClusterRoleRequestList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ClusterRoleRequest, err error)
	ClusterRoleRequestExpansion
}

// clusterRoleRequests implements ClusterRoleRequestInterface
type clusterRoleRequests struct {
	client rest.Interface
}

// newClusterRoleRequests returns a ClusterRoleRequests
func newClusterRoleRequests(c *RegistrationV1alpha1Client) *clusterRoleRequests {
	return &clusterRoleRequests{
		client: c.RESTClient(),
	}
}

// Get takes name of the clusterRoleRequest, and returns the corresponding clusterRoleRequest object, and an error if there is any.
func (c *clusterRoleRequests) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.ClusterRoleRequest, err error) {
	result = &v1alpha1.ClusterRoleRequest{}
	err = c.client.Get().
		Resource("clusterrolerequests").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ClusterRoleRequests that match those selectors.
func (c *clusterRoleRequests) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ClusterRoleRequestList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.ClusterRoleRequestList{}
	err = c.client.Get().
		Resource("clusterrolerequests").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested clusterRoleRequests.
func (c *clusterRoleRequests) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("clusterrolerequests").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a clusterRoleRequest and creates it.  Returns the server's representation of the clusterRoleRequest, and an error, if there is any.
func (c *clusterRoleRequests) Create(ctx context.Context, clusterRoleRequest *v1alpha1.ClusterRoleRequest, opts v1.CreateOptions) (result *v1alpha1.ClusterRoleRequest, err error) {
	result = &v1alpha1.ClusterRoleRequest{}
	err = c.client.Post().
		Resource("clusterrolerequests").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(clusterRoleRequest).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a clusterRoleRequest and updates it. Returns the server's representation of the clusterRoleRequest, and an error, if there is any.
func (c *clusterRoleRequests) Update(ctx context.Context, clusterRoleRequest *v1alpha1.ClusterRoleRequest, opts v1.UpdateOptions) (result *v1alpha1.ClusterRoleRequest, err error) {
	result = &v1alpha1.ClusterRoleRequest{}
	err = c.client.Put().
		Resource("clusterrolerequests").
		Name(clusterRoleRequest.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(clusterRoleRequest).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *clusterRoleRequests) UpdateStatus(ctx context.Context, clusterRoleRequest *v1alpha1.ClusterRoleRequest, opts v1.UpdateOptions) (result *v1alpha1.ClusterRoleRequest, err error) {
	result = &v1alpha1.ClusterRoleRequest{}
	err = c.client.Put().
		Resource("clusterrolerequests").
		Name(clusterRoleRequest.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(clusterRoleRequest).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the clusterRoleRequest and deletes it. Returns an error if one occurs.
func (c *clusterRoleRequests) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("clusterrolerequests").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *clusterRoleRequests) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("clusterrolerequests").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched clusterRoleRequest.
func (c *clusterRoleRequests) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ClusterRoleRequest, err error) {
	result = &v1alpha1.ClusterRoleRequest{}
	err = c.client.Patch(pt).
		Resource("clusterrolerequests").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
