/*
Copyright The Kubernetes Authors.

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

package versioned

import (
	"fmt"

	appsv1alpha "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/typed/apps/v1alpha"
	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/typed/core/v1alpha"
	networkingv1alpha "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/typed/networking/v1alpha"
	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/typed/registration/v1alpha"
	discovery "k8s.io/client-go/discovery"
	rest "k8s.io/client-go/rest"
	flowcontrol "k8s.io/client-go/util/flowcontrol"
)

type Interface interface {
	Discovery() discovery.DiscoveryInterface
	AppsV1alpha() appsv1alpha.AppsV1alphaInterface
	CoreV1alpha() corev1alpha.CoreV1alphaInterface
	NetworkingV1alpha() networkingv1alpha.NetworkingV1alphaInterface
	RegistrationV1alpha() registrationv1alpha.RegistrationV1alphaInterface
}

// Clientset contains the clients for groups. Each group has exactly one
// version included in a Clientset.
type Clientset struct {
	*discovery.DiscoveryClient
	appsV1alpha         *appsv1alpha.AppsV1alphaClient
	coreV1alpha         *corev1alpha.CoreV1alphaClient
	networkingV1alpha   *networkingv1alpha.NetworkingV1alphaClient
	registrationV1alpha *registrationv1alpha.RegistrationV1alphaClient
}

// AppsV1alpha retrieves the AppsV1alphaClient
func (c *Clientset) AppsV1alpha() appsv1alpha.AppsV1alphaInterface {
	return c.appsV1alpha
}

// CoreV1alpha retrieves the CoreV1alphaClient
func (c *Clientset) CoreV1alpha() corev1alpha.CoreV1alphaInterface {
	return c.coreV1alpha
}

// NetworkingV1alpha retrieves the NetworkingV1alphaClient
func (c *Clientset) NetworkingV1alpha() networkingv1alpha.NetworkingV1alphaInterface {
	return c.networkingV1alpha
}

// RegistrationV1alpha retrieves the RegistrationV1alphaClient
func (c *Clientset) RegistrationV1alpha() registrationv1alpha.RegistrationV1alphaInterface {
	return c.registrationV1alpha
}

// Discovery retrieves the DiscoveryClient
func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	if c == nil {
		return nil
	}
	return c.DiscoveryClient
}

// NewForConfig creates a new Clientset for the given config.
// If config's RateLimiter is not set and QPS and Burst are acceptable,
// NewForConfig will generate a rate-limiter in configShallowCopy.
func NewForConfig(c *rest.Config) (*Clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.RateLimiter == nil && configShallowCopy.QPS > 0 {
		if configShallowCopy.Burst <= 0 {
			return nil, fmt.Errorf("burst is required to be greater than 0 when RateLimiter is not set and QPS is set to greater than 0")
		}
		configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configShallowCopy.QPS, configShallowCopy.Burst)
	}
	var cs Clientset
	var err error
	cs.appsV1alpha, err = appsv1alpha.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.coreV1alpha, err = corev1alpha.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.networkingV1alpha, err = networkingv1alpha.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.registrationV1alpha, err = registrationv1alpha.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	cs.DiscoveryClient, err = discovery.NewDiscoveryClientForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	return &cs, nil
}

// NewForConfigOrDie creates a new Clientset for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *Clientset {
	var cs Clientset
	cs.appsV1alpha = appsv1alpha.NewForConfigOrDie(c)
	cs.coreV1alpha = corev1alpha.NewForConfigOrDie(c)
	cs.networkingV1alpha = networkingv1alpha.NewForConfigOrDie(c)
	cs.registrationV1alpha = registrationv1alpha.NewForConfigOrDie(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClientForConfigOrDie(c)
	return &cs
}

// New creates a new Clientset for the given RESTClient.
func New(c rest.Interface) *Clientset {
	var cs Clientset
	cs.appsV1alpha = appsv1alpha.New(c)
	cs.coreV1alpha = corev1alpha.New(c)
	cs.networkingV1alpha = networkingv1alpha.New(c)
	cs.registrationV1alpha = registrationv1alpha.New(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClient(c)
	return &cs
}
