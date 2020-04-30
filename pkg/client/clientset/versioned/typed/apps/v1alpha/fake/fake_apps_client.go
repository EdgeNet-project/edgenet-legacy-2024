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

package fake

import (
	v1alpha "headnode/pkg/client/clientset/versioned/typed/apps/v1alpha"

	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeAppsV1alpha struct {
	*testing.Fake
}

func (c *FakeAppsV1alpha) Projects(namespace string) v1alpha.ProjectInterface {
	return &FakeProjects{c, namespace}
}

func (c *FakeAppsV1alpha) SelectiveDeployments(namespace string) v1alpha.SelectiveDeploymentInterface {
	return &FakeSelectiveDeployments{c, namespace}
}

func (c *FakeAppsV1alpha) Sites() v1alpha.SiteInterface {
	return &FakeSites{c}
}

func (c *FakeAppsV1alpha) SiteRegistrationRequests() v1alpha.SiteRegistrationRequestInterface {
	return &FakeSiteRegistrationRequests{c}
}

func (c *FakeAppsV1alpha) Slices(namespace string) v1alpha.SliceInterface {
	return &FakeSlices{c, namespace}
}

func (c *FakeAppsV1alpha) Users(namespace string) v1alpha.UserInterface {
	return &FakeUsers{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeAppsV1alpha) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
