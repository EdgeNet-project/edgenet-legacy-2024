# Configure permissions as tenant owner/admin

This tutorial describes how you can grant users access to the resources under your tenant's control as the tenant owner or admin.

Kubernetes provides granular control over permission thanks to role-based access control (RBAC). However, it is sometimes difficult for those unfamiliar with Kubernetes to create and maintain user permissions.

EdgeNet offers pre-defined cluster roles to facilitate permission management: admin and collaborator.

## Technologies you will use

You will use [``kubectl``](https://kubernetes.io/docs/reference/kubectl/overview/), the [Kubernetes](https://kubernetes.io/) command-line interface.

## What you will do

You will use your user-specific kubeconfig file provided by EdgeNet to create a role-binding object. This role-binding object will be associated with the e-mail addresses of users you want to grant admin or collaborator roles.

## Steps

### Make sure you have the Kubernetes command-line tool

If you do not already have ``kubectl``, you will need to install it on your system. Follow the [Kubernetes documentation](https://kubernetes.io/docs/tasks/tools/install-kubectl/) for this.

### Be sure the user-specific access credential is ready

A role binding is a Kubernetes object, and to manipulate objects on a Kubernetes system; you need a kubeconfig file. EdgeNet delivers the user-specific kubeconfig file via e-mail right after the registration.

The user-specific file does not allow any actions beyond the roles bound to your user. In case of an authorization issue, don't hesitate to contact us if you are a tenant owner; otherwise, contact your tenant administration.

### Prepare a description of your role binding

The [``.yaml`` format](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/) is used to describe Kubernetes objects. Create one for the role-binding object, following the model of the example shown below. Your ``.yaml`` file must specify the following information regarding your future role binding:
- the **role binding name** the EdgeNet system will use that; it must follow [Kubernetes' rules for names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/) and must be different from any existing role binding names in the namespace
- the **namespace** of the role binding; this is an isolated environment where the role binding applies to
- the **subjects** of the role binding; this field is to associate the role binding with users, groups, or service accounts
  - a **kind** of the subject; it can be User, Group, or ServiceAccount
  - a **name** of the subject
- the **roleRef** of the role binding; the information provided consists of:
  - a **kind** of the role reference; it can be ClusterRole or Role
  - a **name** of the role reference

In what follows, we will assume that this file is saved in your working directory on your system as ``./role_binding.yaml``.

Example:
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: admin
  namespace: lip6-lab
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: User
  name: berat.senel@lip6.fr
- apiGroup: rbac.authorization.k8s.io
  kind: User
  name: maxime.mouchet@lip6.fr
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: edgenet:tenant-admin

```

* Pre-defined cluster roles: `edgenet:tenant-admin` and `edgenet:tenant-collaborator`.

The admin role grants users almost the same privileges that the tenant owner holds, including user management. In comparison, the collaborator role permits users to develop applications.

**P.S.** If you want to create tenant-specific roles, the comprehensive documentation sits on https://kubernetes.io/docs/reference/access-authn-authz/rbac/.

### Create your role binding

Using ``kubectl``, create a role binding object:

```
kubectl create -f ./role_binding.yaml --kubeconfig ./edgenet.cfg
```