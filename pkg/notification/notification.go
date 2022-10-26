/*
Copyright 2022 Contributors to the EdgeNet project.

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

package notification

type Content struct {
	Cluster            string
	User               string
	FirstName          string
	LastName           string
	Subject            string
	Recipient          []string
	RoleRequest        *RoleRequest
	TenantRequest      *TenantRequest
	ClusterRoleRequest *ClusterRoleRequest
}
type RoleRequest struct {
	Name      string
	Namespace string
}
type ClusterRoleRequest struct {
	Name string
}
type TenantRequest struct {
	Tenant string
}

func (c *Content) Init(firstname, lastname, email, subject, clusterUID string, recipient []string) {
	c.Cluster = clusterUID
	c.User = email
	c.FirstName = firstname
	c.LastName = lastname
	c.Subject = subject
	c.Recipient = recipient
}

func (c *Content) SendNotification(purpose string) error {
	var err error
	err = c.email(purpose)
	if c.RoleRequest == nil {
		err = c.slack(purpose)
	}
	return err
}
