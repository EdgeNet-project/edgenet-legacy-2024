# EdgeNet code structure

EdgeNet extends Kubernetes in a few areas that are described here. This document points to the code that enables each of the extensions.


## Multitenancy

Some Kubernetes users need it to support multitenancy. A good example is a large organization with multiple teams, each of which is going to deploy services to a shared cluster. The organization doesn’t want one team’s work to interfere with another’s. EdgeNet presents a similar challenge, except that the teams that share a single EdgeNet cluster come from entirely separate organizations, so there can be no assumption of everyone being bound by common policies, aside from those that they explicitly agree to when joining and using EdgeNet. Because EdgeNet is also multiprovider-based, the need for tenant accountability is particularly acute: individuals and institutions will only provide nodes if they can trust the diverse actors that come from a multitenant environment.

EdgeNet’s multitenancy extensions are built on top of the Kubernetes notions of [namespaces](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) and [resource quotas](https://kubernetes.io/docs/concepts/policy/resource-quotas/). Namespaces provide isolation: users who are deploying services in one namespace do not see and cannot touch services that are deployed by other users in other namespaces. Resource quotas are a means to ensure that the overall resources of the cluster are shared out amongst the namespaces, so that, ideally, users in each namespace have sufficient resources in which to conduct their work.

Using namespaces and resource quotas is a perfectly classic approach to handling Kubernetes multitenancy, and so is EdgeNet’s adoption of a hierarchical naming convention for the namespaces. Currently, the hierarchy is limited to three levels, with “authorities” at the top, an optional “teams” level in the middle, and an optional “slices” level at the bottom. Each user belongs to an authority and optionally is assigned to teams and slices. The services that they deploy in EdgeNet are associated with a namespaces that follows the schemas `authority-<authority-name>-team-<team-name>-slice-<slice-name>, as in, for example, authority-sorbonne-university-team-lip6-lab-slice-cartography-system` and `authority-<authority-name>-slice-<slice-name>`.

(In upcoming versions of EdgeNet, we lift this restriction of three levels, and their terminology, which comes from the PlanetLab, GENI, and Fed4FIRE heritage EdgeNet, and generalize to hierarchical namespaces of any depth that is permitted by the maximum name length, perhaps with a dash as a separator.)

Total resource quotas are associated with authorities, which can then share them out amongst teams, and teams amongst slices.

The code that enables these features is as follows:

Authorities



*   [Authority custom resource](https://github.com/EdgeNet-project/edgenet/tree/master/pkg/controller/v1alpha/authority) creates authorities, and authority admins accordingly.
*   [Authority request custom resource](https://github.com/EdgeNet-project/edgenet/tree/master/pkg/controller/v1alpha/authorityrequest) stores the requests to be approved.
*   [Total resource quota custom resource](https://github.com/EdgeNet-project/edgenet/tree/master/pkg/controller/v1alpha/totalresourcequota) sets general resource quota per authority.

Teams



*   [Team custom resource](https://github.com/EdgeNet-project/edgenet/tree/master/pkg/controller/v1alpha/team) creates a namespace for the participants to create slices independently.

Slices



*   [Slice custom resource](https://github.com/EdgeNet-project/edgenet/tree/master/pkg/controller/v1alpha/slice) creates a namespace with a resource quota in which defined users participate in.


## User management

The way that users are managed in EdgeNet is a direct consequence of EdgeNet’s multitenant structure. As the system grows large, EdgeNet’s central administrators cannot possibly approve and manage each new user, at least not while maintaining the necessary accountability that comes with knowing one’s users. User management is thus handled hierarchically, with the central administrators approving a limited number of “authorities”, each one led by an individual who the administrators can trust. Each authority administrator can in turn approve users of their authority; users who they should know. This delegation of user management could, in principle, extend down the hierarchy, with the administrator of a team within an authority approving users of that team.

The management of users follows the namespace hierarchy described above in the Multitenancy section. But users are not restricted to work only within the namespaces in which their accounts were created. If the administrator of another namespace invites them, they can work in that other namespace as well.  

Each user is associated with an e-mail address, which provides an essential element of trustworthiness. When EdgeNet’s central administrators approve the lead user of a new authority, they confirm that that person has an e-mail address that is assigned by the institution that they belong to. Similarly, the administrator of each authority might wish to approve users only belonging to their institution, and having e-mail addresses that demonstrate that. There is therefore an e-mail verification system that is part of EdgeNet.

Users must agree to an acceptable use policy and be notified of their rights under GDPR, and this is handled here.

Users access rights to resources are handled by RBAC. Roles and role bindings are automatically created based on the type of user which can be an authority admin or normal user. Authority, Team, and Slice controllers use RBAC to create roles and role bindings for permission control. Authority managers can also create their own roles and role bindings under the control of OPA.

Users



*   [Registration package](https://github.com/EdgeNet-project/edgenet/tree/master/pkg/registration) covers the user registration including RBAC management.
*   [Acceptable use policy custom resource](https://github.com/EdgeNet-project/edgenet/tree/master/pkg/controller/v1alpha/acceptableusepolicy) for being GDPR ready.
*   [User custom resource](https://github.com/EdgeNet-project/edgenet/tree/master/pkg/controller/v1alpha/user) provides identification of authority users.
*   [User registration request custom resource](https://github.com/EdgeNet-project/edgenet/tree/master/pkg/controller/v1alpha/userregistrationrequest) stores the requests to be approved.


## Multiprovider support

EdgeNet nodes are not all furnished out of a single datacenter, or even a small number of datacenters, but rather from large numbers of individuals and institutions, who may be users of the system, or simply those wishing to contribute to the system. The “node contribution” extensions enable this.

Node contributions



*   [Node package](https://github.com/EdgeNet-project/edgenet/tree/master/pkg/node) is in use to create a kubeadm token to make a node join into the cluster, etc.
*   [Node labeler custom resource](https://github.com/EdgeNet-project/edgenet/tree/master/pkg/controller/v1/nodelabeler) uses the GeoLite2 database and prepares the labels for nodes.
*   [Node contribution custom resource](https://github.com/EdgeNet-project/edgenet/tree/master/pkg/controller/v1alpha/nodecontribution) establishes an SSH connection by a client, installs the required packages, and runs kubadm join command.


## Selective deployments

The value of EdgeNet to its users lies precisely in its geographic and topological dispersion across the world and around the internet. In order to avail themselves of this value, users need to be able to deploy their services in a distributed fashion, and not to a small number of nodes that are located closely together. The “selective deployment” extension enables this.

Selective deployments



*   [Selective deployment custom resource](https://github.com/EdgeNet-project/edgenet/tree/master/pkg/controller/v1alpha/selectivedeployment) allows us to do geolocation-based deployments.


## Utilities

There are a number of EdgeNet utility functions in support of the extensions described above.

Utility



*   [Authorization package](https://github.com/EdgeNet-project/edgenet/tree/master/pkg/authorization) prepares the environment including the Kubernetes clientset, kubeconfig file, and Namecheap API.
*   [Config package](https://github.com/EdgeNet-project/edgenet/tree/master/pkg/config) includes the functions to manipulate the admin kubeconfig file such as setting a Kubernetes user.
*   [Mailer package](https://github.com/EdgeNet-project/edgenet/tree/master/pkg/mailer) is the mail service of EdgeNet.
*   [Namespace package](https://github.com/EdgeNet-project/edgenet/tree/master/pkg/namespace) holds the namespace related functions including the old API.
*   [Remote IP package](https://github.com/EdgeNet-project/edgenet/tree/master/pkg/remoteip) mainly checks the IP format.
*   [Email verification custom resource](https://github.com/EdgeNet-project/edgenet/tree/master/pkg/controller/v1alpha/emailverification) declares verification objects to be handled by users.
