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

// Content is the structure for the notification content
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

// RoleRequest is the structure for the role request
type RoleRequest struct {
	Name      string
	Namespace string
}

// ClusterRoleRequest is the structure for the cluster role request
type ClusterRoleRequest struct {
	Name string
}

// TenantRequest is the structure for the tenant request
type TenantRequest struct {
	Tenant string
}

// Init is the function to initialize info for the notification content
func (c *Content) Init(firstname, lastname, email, subject, clusterUID string, recipient []string) {
	c.Cluster = clusterUID
	c.User = email
	c.FirstName = firstname
	c.LastName = lastname
	c.Subject = subject
	c.Recipient = recipient
}

// SendNotification is the function to send notification via email and slack
func (c *Content) SendNotification(purpose string) error {
	var err error
	err = c.email(purpose)
	if c.RoleRequest == nil {
		err = c.slack(purpose)
	}
	return err
}
