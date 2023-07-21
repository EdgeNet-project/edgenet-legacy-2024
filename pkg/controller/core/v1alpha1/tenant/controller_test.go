package tenant

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha1"
	corev1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha1"
	edgenetfake "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	edgeinformers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"

	antreav1alpha1 "antrea.io/antrea/pkg/apis/crd/v1alpha1"
	antreafake "antrea.io/antrea/pkg/client/clientset/versioned/fake"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

var (
	alwaysReady        = func() bool { return true }
	noResyncPeriodFunc = func() time.Duration { return 0 }
)

type fixture struct {
	t *testing.T

	kubeclientset    *k8sfake.Clientset
	edgenetclientset *edgenetfake.Clientset
	antreaclientset  *antreafake.Clientset

	// Objects to put in the store.
	tenantLister               []*corev1alpha1.Tenant
	namespaceLister            []*corev1.Namespace
	clusterroleLister          []*rbacv1.ClusterRole
	clusterrolebindingLister   []*rbacv1.ClusterRoleBinding
	rolebindingLister          []*rbacv1.RoleBinding
	networkpolicyLister        []*networkingv1.NetworkPolicy
	clusternetworkpolicyLister []*antreav1alpha1.ClusterNetworkPolicy

	// Actions expected to happen on the client.
	kubeactions    []core.Action
	edgenetactions []core.Action
	antreaactions  []core.Action

	// Objects from here preloaded into NewSimpleFake.
	kubeobjects    []runtime.Object
	edgenetobjects []runtime.Object
	antreaobjects  []runtime.Object
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	f.kubeobjects = []runtime.Object{}
	f.edgenetobjects = []runtime.Object{}
	f.antreaobjects = []runtime.Object{}
	return f
}

func newTenant(name string, cnp, enabled bool) *corev1alpha1.Tenant {
	return &corev1alpha1.Tenant{
		TypeMeta: metav1.TypeMeta{APIVersion: corev1alpha1.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1alpha1.TenantSpec{
			FullName:  fmt.Sprintf("EdgeNet %s", name),
			ShortName: name,
			URL:       fmt.Sprintf("https://%s.org", name),
			Address: corev1alpha.Address{
				City:    "Paris",
				Country: "France",
				Street:  "4 place Jussieu, boite 169",
				ZIP:     "75005",
			},
			Contact: corev1alpha.Contact{
				Email:     fmt.Sprintf("john.doe@%s.org", name),
				FirstName: "John",
				LastName:  "Doe",
				Phone:     "+33123456789",
			},
			ClusterNetworkPolicy: cnp,
			Enabled:              enabled,
		},
	}
}
func newNamespace(name string, labels, annotations map[string]string, ownerReferences []metav1.OwnerReference) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			OwnerReferences: ownerReferences,
			Labels:          labels,
			Annotations:     annotations,
		},
	}
}
func newClusterRole(name, resourceName string, ownerReferences []metav1.OwnerReference) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:            fmt.Sprintf("edgenet:tenants:%s-owner", name),
			OwnerReferences: ownerReferences,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{"core.edgenet.io"},
				Resources:     []string{"tenants"},
				ResourceNames: []string{resourceName},
				Verbs:         []string{"get", "update", "patch", "delete"},
			},
			{
				APIGroups:     []string{"core.edgenet.io"},
				Resources:     []string{"tenants/status"},
				ResourceNames: []string{resourceName},
				Verbs:         []string{"get", "list", "watch"},
			},
		},
	}
}
func newClusterRoleBinding(name, email string, labels map[string]string, ownerReferences []metav1.OwnerReference) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:            fmt.Sprintf("edgenet:tenants:%s-owner", name),
			Labels:          labels,
			OwnerReferences: ownerReferences,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     "User",
				APIGroup: "rbac.authorization.k8s.io",
				Name:     email,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: fmt.Sprintf("edgenet:tenants:%s-owner", name),
		},
	}
}
func newRoleBinding(name, namespace, email string, labels map[string]string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     "User",
				APIGroup: "rbac.authorization.k8s.io",
				Name:     email,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: corev1alpha1.TenantOwnerClusterRoleName,
		},
	}
}
func newNetworkPolicy(name, namespace string, labelSelector metav1.LabelSelector) *networkingv1.NetworkPolicy {
	port := intstr.IntOrString{IntVal: 1}
	endPort := int32(32768)
	ingressRules := []networkingv1.NetworkPolicyIngressRule{
		{
			From: []networkingv1.NetworkPolicyPeer{
				{
					NamespaceSelector: &labelSelector,
				},
				{
					IPBlock: &networkingv1.IPBlock{
						CIDR:   "0.0.0.0/0",
						Except: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
					},
				},
			},
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Port:    &port,
					EndPort: &endPort,
				},
			},
		},
	}
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PolicyTypes: []networkingv1.PolicyType{"Ingress"},
			Ingress:     ingressRules,
		},
	}
}
func newClusterNetworkPolicy(name string, labelSelector metav1.LabelSelector, ownerReferences []metav1.OwnerReference) *antreav1alpha1.ClusterNetworkPolicy {
	drop := antreav1alpha1.RuleActionDrop
	allow := antreav1alpha1.RuleActionAllow
	port := intstr.IntOrString{IntVal: 1}
	endPort := int32(32768)
	ingressRules := []antreav1alpha1.Rule{
		{
			Action: &allow,
			From: []antreav1alpha1.NetworkPolicyPeer{
				{
					NamespaceSelector: &labelSelector,
				},
			},
			Ports: []antreav1alpha1.NetworkPolicyPort{
				{
					Port:    &port,
					EndPort: &endPort,
				},
			},
		},
		{
			Action: &drop,
			From: []antreav1alpha1.NetworkPolicyPeer{
				{
					IPBlock: &antreav1alpha1.IPBlock{
						CIDR: "10.0.0.0/8",
					},
				},
				{
					IPBlock: &antreav1alpha1.IPBlock{
						CIDR: "172.16.0.0/12",
					},
				},
				{
					IPBlock: &antreav1alpha1.IPBlock{
						CIDR: "192.168.0.0/16",
					},
				},
			},
			Ports: []antreav1alpha1.NetworkPolicyPort{
				{
					Port:    &port,
					EndPort: &endPort,
				},
			},
		},
		{
			Action: &allow,
			From: []antreav1alpha1.NetworkPolicyPeer{
				{
					IPBlock: &antreav1alpha1.IPBlock{
						CIDR: "0.0.0.0/0",
					},
				},
			},
			Ports: []antreav1alpha1.NetworkPolicyPort{
				{
					Port:    &port,
					EndPort: &endPort,
				},
			},
		},
	}
	appliedTo := []antreav1alpha1.NetworkPolicyPeer{
		{
			NamespaceSelector: &labelSelector,
		},
	}
	return &antreav1alpha1.ClusterNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			OwnerReferences: ownerReferences,
		},
		Spec: antreav1alpha1.ClusterNetworkPolicySpec{
			Tier:      "tenant",
			Priority:  5,
			AppliedTo: appliedTo,
			Ingress:   ingressRules,
		},
	}
}

func (f *fixture) newController() (*Controller, edgeinformers.SharedInformerFactory) {
	f.kubeclientset = k8sfake.NewSimpleClientset(f.kubeobjects...)
	f.edgenetclientset = edgenetfake.NewSimpleClientset(f.edgenetobjects...)
	f.antreaclientset = antreafake.NewSimpleClientset(f.antreaobjects...)

	edgeinformer := edgeinformers.NewSharedInformerFactory(f.edgenetclientset, noResyncPeriodFunc())
	//kubeinformer := kubeinformers.NewSharedInformerFactory(f.kubeclientset, noResyncPeriodFunc())

	controller := NewController(f.kubeclientset, f.edgenetclientset, f.antreaclientset,
		edgeinformer.Core().V1alpha1().Tenants())

	controller.tenantsSynced = alwaysReady
	controller.recorder = &record.FakeRecorder{}

	for _, tenant := range f.tenantLister {
		edgeinformer.Core().V1alpha1().Tenants().Informer().GetIndexer().Add(tenant)
	}

	return controller, edgeinformer
}

func (f *fixture) run(tenantName string) {
	f.runController(tenantName, true, false)
}

func (f *fixture) runExpectError(tenantName string) {
	f.runController(tenantName, true, true)
}

func (f *fixture) runController(tenantName string, startInformers bool, expectError bool) {
	c, edgei := f.newController()
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		edgei.Start(stopCh)
	}

	err := c.syncHandler(tenantName)
	if err != nil {
		if !expectError {
			f.t.Errorf("error syncing tenant: %v", err)
		}
	} else {
		if expectError {
			f.t.Error("expected error syncing tenant, got nil")
		}
	}

	edgenetActions := filterInformerActions(f.edgenetclientset.Actions())
	for i, edgenetAction := range edgenetActions {
		if len(f.edgenetactions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(edgenetActions)-len(f.edgenetactions), edgenetActions[i:])
			break
		}

		expectedAction := f.edgenetactions[i]
		checkAction(expectedAction, edgenetAction, f.t)
	}

	if len(f.edgenetactions) > len(edgenetActions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.edgenetactions)-len(edgenetActions), f.edgenetactions[len(edgenetActions):])
	}

	k8sActions := filterInformerActions(f.kubeclientset.Actions())
	for i, action := range k8sActions {
		if len(f.kubeactions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(k8sActions)-len(f.kubeactions), k8sActions[i:])
			break
		}

		expectedAction := f.kubeactions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.kubeactions) > len(k8sActions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.kubeactions)-len(k8sActions), f.kubeactions[len(k8sActions):])
	}

	antreaActions := filterInformerActions(f.antreaclientset.Actions())
	for i, action := range antreaActions {
		if len(f.antreaactions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(antreaActions)-len(f.antreaactions), antreaActions[i:])
			break
		}

		expectedAction := f.antreaactions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.antreaactions) > len(antreaActions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.antreaactions)-len(antreaActions), f.antreaactions[len(antreaActions):])
	}
}

func checkAction(expected, actual core.Action, t *testing.T) {
	if !(expected.Matches(actual.GetVerb(), actual.GetResource().Resource) && actual.GetSubresource() == expected.GetSubresource()) {
		t.Errorf("Expected\n\t%#v\ngot\n\t%#v", expected, actual)
		return
	}

	if reflect.TypeOf(actual) != reflect.TypeOf(expected) {
		t.Errorf("Action has wrong type. Expected: %t. Got: %t", expected, actual)
		return
	}

	switch a := actual.(type) {
	case core.GetActionImpl:
		e, _ := expected.(core.GetActionImpl)
		expName := e.GetName()
		expResource := e.GetResource().Resource
		name := a.GetName()
		resource := a.GetResource().Resource

		if expName != name || expResource != resource {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expName, name))
		}
	case core.DeleteCollectionActionImpl:
		e, _ := expected.(core.DeleteCollectionActionImpl)
		expNamespace := e.Namespace
		expResource := e.GetResource().Resource
		namespace := a.Namespace
		resource := a.GetResource().Resource

		if expNamespace != namespace || expResource != resource {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expNamespace, namespace))
		}
	case core.DeleteActionImpl:
		e, _ := expected.(core.DeleteActionImpl)
		expName := e.GetName()
		expResource := e.GetResource().Resource
		name := a.GetName()
		resource := a.GetResource().Resource

		if expName != name || expResource != resource {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expName, name))
		}
	case core.CreateActionImpl:
		e, _ := expected.(core.CreateActionImpl)
		expObject := e.GetObject()
		object := a.GetObject()

		if !reflect.DeepEqual(expObject, object) {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expObject, object))
		}
	case core.UpdateActionImpl:
		e, _ := expected.(core.UpdateActionImpl)
		expObject := e.GetObject()
		object := a.GetObject()
		if a.Subresource == "" {
			if !reflect.DeepEqual(expObject, object) {
				t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
					a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expObject, object))
			}
		} else {
			if reflect.DeepEqual(expObject, object) {
				t.Errorf("Action %s %s has same object status\nSame:\n %s",
					a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expObject, object))
			}
		}
	case core.PatchActionImpl:
		e, _ := expected.(core.PatchActionImpl)
		expPatch := e.GetPatch()
		patch := a.GetPatch()

		if !reflect.DeepEqual(expPatch, patch) {
			t.Errorf("Action %s %s has wrong patch\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expPatch, patch))
		}
	default:
		t.Errorf("Uncaptured Action %s %s, you should explicitly add a case to capture it",
			actual.GetVerb(), actual.GetResource().Resource)
	}
}

func filterInformerActions(actions []core.Action) []core.Action {
	ret := []core.Action{}
	for _, action := range actions {
		if len(action.GetNamespace()) == 0 &&
			(action.Matches("list", "tenants") ||
				action.Matches("watch", "tenants") ||
				action.Matches("list", "namespaces") ||
				action.Matches("watch", "namespaces")) {
			continue
		}
		ret = append(ret, action)
	}
	return ret
}

func (f *fixture) expectGetRootAction(name, resource, kind string) {
	switch kind {
	case "kube":
		f.kubeactions = append(f.kubeactions, core.NewRootGetAction(schema.GroupVersionResource{Resource: resource}, name))
	default:
		f.antreaactions = append(f.antreaactions, core.NewRootGetAction(schema.GroupVersionResource{Resource: resource}, name))
	}
}
func (f *fixture) expectGetAction(name, namespace, resource string) {
	f.kubeactions = append(f.kubeactions, core.NewGetAction(schema.GroupVersionResource{Resource: resource}, namespace, name))
}
func (f *fixture) expectCreateNamespaceAction(namespace *corev1.Namespace) {
	f.kubeactions = append(f.kubeactions, core.NewRootCreateAction(schema.GroupVersionResource{Resource: "namespaces"}, namespace))
}
func (f *fixture) expectCreateClusterRoleAction(clusterrole *rbacv1.ClusterRole) {
	f.kubeactions = append(f.kubeactions, core.NewRootCreateAction(schema.GroupVersionResource{Resource: "clusterroles"}, clusterrole))
}
func (f *fixture) expectUpdateClusterRoleAction(clusterrole *rbacv1.ClusterRole) {
	f.kubeactions = append(f.kubeactions, core.NewRootUpdateAction(schema.GroupVersionResource{Resource: "clusterroles"}, clusterrole))
}
func (f *fixture) expectCreateClusterRoleBindingAction(clusterrolebinding *rbacv1.ClusterRoleBinding) {
	f.kubeactions = append(f.kubeactions, core.NewRootCreateAction(schema.GroupVersionResource{Resource: "clusterrolebindings"}, clusterrolebinding))
}
func (f *fixture) expectUpdateClusterRoleBindingAction(clusterrolebinding *rbacv1.ClusterRoleBinding) {
	f.kubeactions = append(f.kubeactions, core.NewRootUpdateAction(schema.GroupVersionResource{Resource: "clusterrolebindings"}, clusterrolebinding))
}
func (f *fixture) expectCreateRoleBindingAction(rolebinding *rbacv1.RoleBinding) {
	f.kubeactions = append(f.kubeactions, core.NewCreateAction(schema.GroupVersionResource{Resource: "rolebindings"}, rolebinding.GetNamespace(), rolebinding))
}
func (f *fixture) expectUpdateRoleBindingAction(rolebinding *rbacv1.RoleBinding) {
	f.kubeactions = append(f.kubeactions, core.NewUpdateAction(schema.GroupVersionResource{Resource: "rolebindings"}, rolebinding.GetNamespace(), rolebinding))
}
func (f *fixture) expectCreateNetworkPolicyAction(networkpolicy *networkingv1.NetworkPolicy) {
	f.kubeactions = append(f.kubeactions, core.NewCreateAction(schema.GroupVersionResource{Resource: "networkpolicies"}, networkpolicy.GetNamespace(), networkpolicy))
}
func (f *fixture) expectCreateClusterNetworkPolicyAction(clusternetworkpolicy *antreav1alpha1.ClusterNetworkPolicy) {
	f.antreaactions = append(f.antreaactions, core.NewRootCreateAction(schema.GroupVersionResource{Resource: "clusternetworkpolicies"}, clusternetworkpolicy))
}
func (f *fixture) expectDeleteClusterNetworkPolicyAction(name string) {
	f.antreaactions = append(f.antreaactions, core.NewRootDeleteAction(schema.GroupVersionResource{Resource: "clusternetworkpolicies"}, name))
}
func (f *fixture) expectUpdateTenantStatusAction(tenant *corev1alpha1.Tenant) {
	f.edgenetactions = append(f.edgenetactions, core.NewRootUpdateSubresourceAction(schema.GroupVersionResource{Resource: "tenants"}, "status", tenant))
}
func (f *fixture) expectUpdateNamespaceAction(namespace *corev1.Namespace) {
	f.kubeactions = append(f.kubeactions, core.NewRootUpdateAction(schema.GroupVersionResource{Resource: "namespaces"}, namespace))
}
func (f *fixture) expectDeleteCollectionAction(namespace, resource, kind string) {
	switch kind {
	case "kube":
		f.kubeactions = append(f.kubeactions, core.NewDeleteCollectionAction(schema.GroupVersionResource{Group: rbacv1.SchemeGroupVersion.Group, Version: rbacv1.SchemeGroupVersion.Version, Resource: resource}, namespace, metav1.ListOptions{}))
	default:
		f.edgenetactions = append(f.edgenetactions, core.NewDeleteCollectionAction(schema.GroupVersionResource{Resource: resource}, namespace, metav1.ListOptions{}))
	}
}
func (f *fixture) expectRootDeleteCollectionAction(resource string, listOptions metav1.ListOptions) {
	f.kubeactions = append(f.kubeactions, core.NewRootDeleteCollectionAction(schema.GroupVersionResource{Group: rbacv1.SchemeGroupVersion.Group, Version: rbacv1.SchemeGroupVersion.Version, Resource: resource}, listOptions))
}

func getKey(tenant *corev1alpha1.Tenant, t *testing.T) string {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(tenant)
	if err != nil {
		t.Errorf("Unexpected error getting key for tenant %v: %v", tenant.Name, err)
		return ""
	}
	return key
}

func TestCreateTenant(t *testing.T) {
	f := newFixture(t)
	tenant := newTenant("tenant1", true, true)

	kubenamespace := newNamespace("kube-system", nil, nil, nil)
	namespace := newNamespace(tenant.GetName(), map[string]string{"edge-net.io/kind": "core", "edge-net.io/tenant": tenant.GetName(), "edge-net.io/tenant-uid": string(tenant.GetUID()), "edge-net.io/cluster-uid": ""}, map[string]string{"scheduler.alpha.kubernetes.io/node-selector": "edge-net.io/access=public,edge-net.io/slice=none"}, []metav1.OwnerReference{tenant.MakeOwnerReference()})
	clusterrole := newClusterRole(tenant.GetName(), tenant.GetName(), []metav1.OwnerReference{tenant.MakeOwnerReference()})
	clusterrolebinding := newClusterRoleBinding(tenant.GetName(), tenant.Spec.Contact.Email, map[string]string{"edge-net.io/generated": "true"}, []metav1.OwnerReference{tenant.MakeOwnerReference()})

	f.tenantLister = append(f.tenantLister, tenant)
	f.edgenetobjects = append(f.edgenetobjects, tenant)

	f.namespaceLister = append(f.namespaceLister, kubenamespace, namespace)
	f.clusterroleLister = append(f.clusterroleLister, clusterrole)
	f.clusterrolebindingLister = append(f.clusterrolebindingLister, clusterrolebinding)
	f.kubeobjects = append(f.kubeobjects, kubenamespace)

	f.expectGetRootAction(kubenamespace.GetName(), "namespaces", "kube")
	f.expectCreateNamespaceAction(namespace)
	f.expectCreateClusterRoleAction(clusterrole)
	f.expectCreateClusterRoleBindingAction(clusterrolebinding)
	f.expectUpdateTenantStatusAction(tenant)

	f.run(getKey(tenant, t))
}

func TestTenantEstablishment(t *testing.T) {
	f := newFixture(t)
	tenant := newTenant("tenant2", true, true)
	tenant.Status.Failed = 0
	tenant.Status.State = corev1alpha1.StatusCoreNamespaceCreated
	tenant.Status.Message = messageCreated

	kubenamespace := newNamespace("kube-system", nil, nil, nil)
	namespace := newNamespace(tenant.GetName(), map[string]string{"edge-net.io/kind": "core", "edge-net.io/tenant": tenant.GetName(), "edge-net.io/tenant-uid": string(tenant.GetUID()), "edge-net.io/cluster-uid": ""}, map[string]string{"scheduler.alpha.kubernetes.io/node-selector": "edge-net.io/access=public,edge-net.io/slice=none"}, []metav1.OwnerReference{tenant.MakeOwnerReference()})
	rolebinding := newRoleBinding(corev1alpha1.TenantOwnerClusterRoleName, tenant.GetName(), tenant.Spec.Contact.Email, map[string]string{"edge-net.io/generated": "true", "edge-net.io/notification": "true"})
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"edge-net.io/subtenant": "false", "edge-net.io/tenant": tenant.GetName(), "edge-net.io/tenant-uid": string(tenant.GetUID()), "edge-net.io/cluster-uid": string(kubenamespace.GetUID())}}
	networkpolicy := newNetworkPolicy("baseline", tenant.GetName(), labelSelector)
	clusternetworkpolicy := newClusterNetworkPolicy(tenant.GetName(), labelSelector, []metav1.OwnerReference{tenant.MakeOwnerReference()})

	f.tenantLister = append(f.tenantLister, tenant)
	f.edgenetobjects = append(f.edgenetobjects, tenant)

	f.namespaceLister = append(f.namespaceLister, kubenamespace, namespace)
	f.networkpolicyLister = append(f.networkpolicyLister, networkpolicy)
	f.clusternetworkpolicyLister = append(f.clusternetworkpolicyLister, clusternetworkpolicy)
	f.rolebindingLister = append(f.rolebindingLister, rolebinding)
	f.kubeobjects = append(f.kubeobjects, kubenamespace, namespace)

	f.expectGetRootAction(kubenamespace.GetName(), "namespaces", "kube")
	f.expectCreateNetworkPolicyAction(networkpolicy)
	f.expectCreateClusterNetworkPolicyAction(clusternetworkpolicy)
	f.expectCreateRoleBindingAction(rolebinding)
	f.expectUpdateTenantStatusAction(tenant)

	f.run(getKey(tenant, t))
}

func TestTenantDisabled(t *testing.T) {
	f := newFixture(t)
	tenant := newTenant("tenant3", true, false)
	tenant.Status.Failed = 0
	tenant.Status.State = corev1alpha1.StatusEstablished
	tenant.Status.Message = messageEstablished

	kubenamespace := newNamespace("kube-system", nil, nil, nil)
	namespace := newNamespace(tenant.GetName(), map[string]string{"edge-net.io/kind": "core", "edge-net.io/tenant": tenant.GetName(), "edge-net.io/tenant-uid": string(tenant.GetUID()), "edge-net.io/cluster-uid": ""}, map[string]string{"scheduler.alpha.kubernetes.io/node-selector": "edge-net.io/access=public,edge-net.io/slice=none"}, []metav1.OwnerReference{tenant.MakeOwnerReference()})
	clusterrole := newClusterRole(tenant.GetName(), tenant.GetName(), []metav1.OwnerReference{tenant.MakeOwnerReference()})
	clusterrolebinding := newClusterRoleBinding(tenant.GetName(), tenant.Spec.Contact.Email, map[string]string{"edge-net.io/generated": "true"}, []metav1.OwnerReference{tenant.MakeOwnerReference()})
	rolebinding := newRoleBinding(corev1alpha1.TenantOwnerClusterRoleName, tenant.GetName(), tenant.Spec.Contact.Email, map[string]string{"edge-net.io/generated": "true", "edge-net.io/notification": "true"})
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"edge-net.io/subtenant": "false", "edge-net.io/tenant": tenant.GetName(), "edge-net.io/tenant-uid": string(tenant.GetUID()), "edge-net.io/cluster-uid": string(kubenamespace.GetUID())}}

	networkpolicy := newNetworkPolicy("baseline", tenant.GetName(), labelSelector)
	clusternetworkpolicy := newClusterNetworkPolicy(tenant.GetName(), labelSelector, []metav1.OwnerReference{tenant.MakeOwnerReference()})

	f.tenantLister = append(f.tenantLister, tenant)
	f.edgenetobjects = append(f.edgenetobjects, tenant)

	f.namespaceLister = append(f.namespaceLister, kubenamespace, namespace)
	f.clusterroleLister = append(f.clusterroleLister, clusterrole)
	f.clusterrolebindingLister = append(f.clusterrolebindingLister, clusterrolebinding)
	f.networkpolicyLister = append(f.networkpolicyLister, networkpolicy)
	f.clusternetworkpolicyLister = append(f.clusternetworkpolicyLister, clusternetworkpolicy)
	f.rolebindingLister = append(f.rolebindingLister, rolebinding)
	f.kubeobjects = append(f.kubeobjects, kubenamespace, namespace, clusterrole, clusterrolebinding, rolebinding, networkpolicy)
	f.antreaobjects = append(f.antreaobjects, clusternetworkpolicy)

	listOptions := metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/tenant=%s,edge-net.io/tenant-uid=%s,edge-net.io/cluster-uid=%s", tenant.GetName(), string(tenant.GetUID()), string(kubenamespace.GetUID()))}
	f.expectGetRootAction(kubenamespace.GetName(), "namespaces", "kube")
	f.expectRootDeleteCollectionAction("clusterroles", listOptions)
	f.expectRootDeleteCollectionAction("clusterrolebindings", listOptions)
	f.expectDeleteCollectionAction(tenant.GetName(), "rolebindings", "kube")
	f.expectDeleteCollectionAction(tenant.GetName(), "sliceclaims", "edgenet")
	f.expectDeleteCollectionAction(tenant.GetName(), "subnamespaces", "edgenet")

	f.run(getKey(tenant, t))
}

func TestReconcileDoNothing(t *testing.T) {
	f := newFixture(t)
	tenant := newTenant("tenant4", true, true)
	tenant.Status.Failed = 0
	tenant.Status.State = corev1alpha1.StatusEstablished
	tenant.Status.Message = messageEstablished

	kubenamespace := newNamespace("kube-system", nil, nil, nil)
	namespace := newNamespace(tenant.GetName(), map[string]string{"edge-net.io/kind": "core", "edge-net.io/tenant": tenant.GetName(), "edge-net.io/tenant-uid": string(tenant.GetUID()), "edge-net.io/cluster-uid": ""}, map[string]string{"scheduler.alpha.kubernetes.io/node-selector": "edge-net.io/access=public,edge-net.io/slice=none"}, []metav1.OwnerReference{tenant.MakeOwnerReference()})
	clusterrole := newClusterRole(tenant.GetName(), tenant.GetName(), []metav1.OwnerReference{tenant.MakeOwnerReference()})
	clusterrolebinding := newClusterRoleBinding(tenant.GetName(), tenant.Spec.Contact.Email, map[string]string{"edge-net.io/generated": "true"}, []metav1.OwnerReference{tenant.MakeOwnerReference()})
	rolebinding := newRoleBinding(corev1alpha1.TenantOwnerClusterRoleName, tenant.GetName(), tenant.Spec.Contact.Email, map[string]string{"edge-net.io/generated": "true", "edge-net.io/notification": "true"})
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"edge-net.io/subtenant": "false", "edge-net.io/tenant": tenant.GetName(), "edge-net.io/tenant-uid": string(tenant.GetUID()), "edge-net.io/cluster-uid": string(kubenamespace.GetUID())}}

	networkpolicy := newNetworkPolicy("baseline", tenant.GetName(), labelSelector)
	clusternetworkpolicy := newClusterNetworkPolicy(tenant.GetName(), labelSelector, []metav1.OwnerReference{tenant.MakeOwnerReference()})

	f.tenantLister = append(f.tenantLister, tenant)
	f.edgenetobjects = append(f.edgenetobjects, tenant)

	f.namespaceLister = append(f.namespaceLister, kubenamespace, namespace)
	f.clusterroleLister = append(f.clusterroleLister, clusterrole)
	f.clusterrolebindingLister = append(f.clusterrolebindingLister, clusterrolebinding)
	f.networkpolicyLister = append(f.networkpolicyLister, networkpolicy)
	f.clusternetworkpolicyLister = append(f.clusternetworkpolicyLister, clusternetworkpolicy)
	f.rolebindingLister = append(f.rolebindingLister, rolebinding)
	f.kubeobjects = append(f.kubeobjects, kubenamespace, namespace, clusterrole, clusterrolebinding, rolebinding, networkpolicy)
	f.antreaobjects = append(f.antreaobjects, clusternetworkpolicy)

	f.expectGetRootAction(kubenamespace.GetName(), "namespaces", "kube")
	f.expectGetAction(rolebinding.GetName(), rolebinding.GetNamespace(), "rolebindings")
	f.expectGetAction(networkpolicy.GetName(), networkpolicy.GetNamespace(), "networkpolicies")
	f.expectGetRootAction(clusternetworkpolicy.GetName(), "clusternetworkpolicies", "antrea")
	f.expectGetRootAction(clusterrolebinding.GetName(), "clusterrolebindings", "kube")
	f.expectGetRootAction(namespace.GetName(), "namespaces", "kube")

	f.run(getKey(tenant, t))
}

func TestReconcile(t *testing.T) {
	f := newFixture(t)
	tenant := newTenant("tenant5", false, true)
	tenant.Status.Failed = 0
	tenant.Status.State = corev1alpha1.StatusEstablished
	tenant.Status.Message = messageEstablished

	kubenamespace := newNamespace("kube-system", nil, nil, nil)
	namespace := newNamespace(tenant.GetName(), map[string]string{"edge-net.io/kind": "core", "edge-net.io/tenant": tenant.GetName(), "edge-net.io/tenant-uid": string(tenant.GetUID()), "edge-net.io/cluster-uid": ""}, map[string]string{"scheduler.alpha.kubernetes.io/node-selector": "edge-net.io/access=public,edge-net.io/slice=none"}, []metav1.OwnerReference{tenant.MakeOwnerReference()})
	clusterrolebinding := newClusterRoleBinding(tenant.GetName(), tenant.Spec.Contact.Email, map[string]string{"edge-net.io/generated": "true"}, []metav1.OwnerReference{tenant.MakeOwnerReference()})
	rolebinding := newRoleBinding(corev1alpha1.TenantOwnerClusterRoleName, tenant.GetName(), tenant.Spec.Contact.Email, map[string]string{"edge-net.io/generated": "true", "edge-net.io/notification": "true"})
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"edge-net.io/subtenant": "false", "edge-net.io/tenant": tenant.GetName(), "edge-net.io/tenant-uid": string(tenant.GetUID()), "edge-net.io/cluster-uid": string(kubenamespace.GetUID())}}
	networkpolicy := newNetworkPolicy("baseline", tenant.GetName(), labelSelector)
	clusternetworkpolicy := newClusterNetworkPolicy(tenant.GetName(), labelSelector, []metav1.OwnerReference{tenant.MakeOwnerReference()})

	f.tenantLister = append(f.tenantLister, tenant)
	f.edgenetobjects = append(f.edgenetobjects, tenant)
	f.namespaceLister = append(f.namespaceLister, kubenamespace)
	f.kubeobjects = append(f.kubeobjects, kubenamespace)

	f.expectGetRootAction(kubenamespace.GetName(), "namespaces", "kube")
	f.expectGetAction(rolebinding.GetName(), rolebinding.GetNamespace(), "rolebindings")
	f.expectGetAction(networkpolicy.GetName(), networkpolicy.GetNamespace(), "networkpolicies")
	f.expectGetRootAction(clusternetworkpolicy.GetName(), "clusternetworkpolicies", "antrea")
	f.expectGetRootAction(clusterrolebinding.GetName(), "clusterrolebindings", "kube")
	f.expectGetRootAction(namespace.GetName(), "namespaces", "kube")
	f.expectUpdateTenantStatusAction(tenant)

	f.run(getKey(tenant, t))
}

func TestReconcileThroughStatusCoreNamespaceCreated(t *testing.T) {
	f := newFixture(t)
	tenant := newTenant("tenant6", false, true)
	tenant.Status.Failed = 0
	tenant.Status.State = corev1alpha1.StatusCoreNamespaceCreated
	tenant.Status.Message = messageCreated

	kubenamespace := newNamespace("kube-system", nil, nil, nil)
	namespace := newNamespace(tenant.GetName(), map[string]string{"edge-net.io/kind": "core", "edge-net.io/tenant": tenant.GetName(), "edge-net.io/tenant-uid": string(tenant.GetUID()), "edge-net.io/cluster-uid": ""}, map[string]string{"scheduler.alpha.kubernetes.io/node-selector": "edge-net.io/access=public,edge-net.io/slice=none"}, []metav1.OwnerReference{tenant.MakeOwnerReference()})
	clusterrole := newClusterRole(tenant.GetName(), tenant.GetName(), []metav1.OwnerReference{tenant.MakeOwnerReference()})
	clusterrolebinding := newClusterRoleBinding(tenant.GetName(), tenant.Spec.Contact.Email, map[string]string{"edge-net.io/generated": "true"}, []metav1.OwnerReference{tenant.MakeOwnerReference()})
	rolebinding := newRoleBinding(corev1alpha1.TenantOwnerClusterRoleName, tenant.GetName(), tenant.Spec.Contact.Email, map[string]string{"edge-net.io/generated": "true", "edge-net.io/notification": "true"})
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"edge-net.io/subtenant": "false", "edge-net.io/tenant": tenant.GetName(), "edge-net.io/tenant-uid": string(tenant.GetUID()), "edge-net.io/cluster-uid": string(kubenamespace.GetUID())}}

	networkpolicy := newNetworkPolicy("baseline", tenant.GetName(), labelSelector)
	clusternetworkpolicy := newClusterNetworkPolicy(tenant.GetName(), labelSelector, []metav1.OwnerReference{tenant.MakeOwnerReference()})

	f.tenantLister = append(f.tenantLister, tenant)
	f.edgenetobjects = append(f.edgenetobjects, tenant)

	f.namespaceLister = append(f.namespaceLister, kubenamespace, namespace)
	f.clusterroleLister = append(f.clusterroleLister, clusterrole)
	f.clusterrolebindingLister = append(f.clusterrolebindingLister, clusterrolebinding)
	f.networkpolicyLister = append(f.networkpolicyLister, networkpolicy)
	f.clusternetworkpolicyLister = append(f.clusternetworkpolicyLister, clusternetworkpolicy)
	f.rolebindingLister = append(f.rolebindingLister, rolebinding)
	f.kubeobjects = append(f.kubeobjects, kubenamespace, namespace, clusterrole, clusterrolebinding, rolebinding, networkpolicy)
	f.antreaobjects = append(f.antreaobjects, clusternetworkpolicy)

	f.expectGetRootAction(kubenamespace.GetName(), "namespaces", "kube")
	f.expectCreateNetworkPolicyAction(networkpolicy)
	f.expectDeleteClusterNetworkPolicyAction(clusternetworkpolicy.GetName())
	f.expectCreateRoleBindingAction(rolebinding)
	f.expectGetAction(rolebinding.GetName(), rolebinding.GetNamespace(), "rolebindings")
	f.expectUpdateRoleBindingAction(rolebinding)
	f.expectUpdateTenantStatusAction(tenant)

	f.run(getKey(tenant, t))
}

func TestReconcileThroughStatusReconciliation(t *testing.T) {
	f := newFixture(t)
	tenant := newTenant("tenant7", true, true)
	tenant.Status.Failed = 0
	tenant.Status.State = corev1alpha1.StatusReconciliation
	tenant.Status.Message = messageReconciliation

	kubenamespace := newNamespace("kube-system", nil, nil, nil)
	namespace := newNamespace(tenant.GetName(), map[string]string{"edge-net.io/kind": "core", "edge-net.io/tenant": tenant.GetName(), "edge-net.io/tenant-uid": string(tenant.GetUID()), "edge-net.io/cluster-uid": ""}, map[string]string{"scheduler.alpha.kubernetes.io/node-selector": "edge-net.io/access=public,edge-net.io/slice=none"}, []metav1.OwnerReference{tenant.MakeOwnerReference()})
	clusterrole := newClusterRole(tenant.GetName(), tenant.GetName(), []metav1.OwnerReference{tenant.MakeOwnerReference()})
	clusterrolebinding := newClusterRoleBinding(tenant.GetName(), tenant.Spec.Contact.Email, map[string]string{"edge-net.io/generated": "true"}, []metav1.OwnerReference{tenant.MakeOwnerReference()})

	f.tenantLister = append(f.tenantLister, tenant)
	f.edgenetobjects = append(f.edgenetobjects, tenant)

	f.namespaceLister = append(f.namespaceLister, kubenamespace, namespace)
	f.clusterroleLister = append(f.clusterroleLister, clusterrole)
	f.clusterrolebindingLister = append(f.clusterrolebindingLister, clusterrolebinding)
	f.kubeobjects = append(f.kubeobjects, kubenamespace, namespace, clusterrole, clusterrolebinding)

	f.expectGetRootAction(kubenamespace.GetName(), "namespaces", "kube")
	f.expectCreateNamespaceAction(namespace)
	f.expectGetRootAction(namespace.GetName(), "namespaces", "kube")
	f.expectUpdateNamespaceAction(namespace)
	f.expectCreateClusterRoleAction(clusterrole)
	f.expectGetRootAction(clusterrole.GetName(), "clusterroles", "kube")
	f.expectUpdateClusterRoleAction(clusterrole)
	f.expectCreateClusterRoleBindingAction(clusterrolebinding)
	f.expectGetRootAction(clusterrolebinding.GetName(), "clusterrolebindings", "kube")
	f.expectUpdateClusterRoleBindingAction(clusterrolebinding)
	f.expectUpdateTenantStatusAction(tenant)

	f.run(getKey(tenant, t))
}
