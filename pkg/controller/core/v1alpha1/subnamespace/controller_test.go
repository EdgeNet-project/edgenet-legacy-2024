package subnamespace

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha1"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/multitenancy"
	"github.com/EdgeNet-project/edgenet/pkg/signals"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog"
)

// The main structure of test group
type TestGroup struct {
	tenantObj        corev1alpha.Tenant
	trqObj           corev1alpha.TenantResourceQuota
	resourceQuotaObj corev1.ResourceQuota
	subNamespaceObj  corev1alpha.SubNamespace
}

var kubeclientset kubernetes.Interface = testclient.NewSimpleClientset()
var edgenetclientset versioned.Interface = edgenettestclient.NewSimpleClientset()

func TestMain(m *testing.M) {
	klog.SetOutput(ioutil.Discard)
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)

	flag.String("dir", "../../../../..", "Override the directory.")
	flag.Parse()

	stopCh := signals.SetupSignalHandler()

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeclientset, 0)
	edgenetInformerFactory := informers.NewSharedInformerFactory(edgenetclientset, 0)

	controller := NewController(kubeclientset,
		edgenetclientset,
		kubeInformerFactory.Rbac().V1().Roles(),
		kubeInformerFactory.Rbac().V1().RoleBindings(),
		kubeInformerFactory.Networking().V1().NetworkPolicies(),
		kubeInformerFactory.Core().V1().LimitRanges(),
		kubeInformerFactory.Core().V1().Secrets(),
		kubeInformerFactory.Core().V1().ConfigMaps(),
		kubeInformerFactory.Core().V1().ServiceAccounts(),
		edgenetInformerFactory.Core().V1alpha1().SubNamespaces())

	edgenetInformerFactory.Start(stopCh)

	go func() {
		if err := controller.Run(2, stopCh); err != nil {
			klog.Fatalf("Error running controller: %s", err.Error())
		}
	}()

	kubeSystemNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}}
	kubeclientset.CoreV1().Namespaces().Create(context.TODO(), kubeSystemNamespace, metav1.CreateOptions{})
	multitenancyManager := multitenancy.NewManager(kubeclientset, edgenetclientset)
	multitenancyManager.CreateClusterRoles()

	time.Sleep(500 * time.Millisecond)

	os.Exit(m.Run())
	<-stopCh
}

// Init syncs the test group
func (g *TestGroup) Init() {
	tenantObj := corev1alpha.Tenant{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Tenant",
			APIVersion: "core.edgenet.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
		},
		Spec: corev1alpha.TenantSpec{
			FullName:  "EdgeNet",
			ShortName: "EdgeNet",
			URL:       "https://www.edge-net.org",
			Address: corev1alpha.Address{
				City:    "Paris - NY - CA",
				Country: "France - US",
				Street:  "4 place Jussieu, boite 169",
				ZIP:     "75005",
			},
			Contact: corev1alpha.Contact{
				Email:     "john.doe@edge-net.org",
				FirstName: "John",
				LastName:  "Doe",
				Phone:     "+33NUMBER",
			},
			Enabled: true,
		},
	}
	trqObj := corev1alpha.TenantResourceQuota{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TenantResourceQuota",
			APIVersion: "core.edgenet.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
		},
		Spec: corev1alpha.TenantResourceQuotaSpec{
			Claim: map[string]corev1alpha.ResourceTuning{
				"initial": {
					ResourceList: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("8000m"),
						corev1.ResourceMemory: resource.MustParse("8192Mi"),
					},
				},
			},
		},
	}
	resourceQuotaObj := corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name: "core-quota",
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: map[corev1.ResourceName]resource.Quantity{
				"cpu":              resource.MustParse("8000m"),
				"memory":           resource.MustParse("8192Mi"),
				"requests.storage": resource.MustParse("8Gi"),
			},
		},
	}
	subNamespaceObj := corev1alpha.SubNamespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SubNamespace",
			APIVersion: "core.edgenet.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "edgenet-sub",
			Namespace: "edgenet",
			UID:       "edgenet-sub",
		},
		Spec: corev1alpha.SubNamespaceSpec{
			Workspace: &corev1alpha.Workspace{
				ResourceAllocation: map[corev1.ResourceName]resource.Quantity{
					"cpu":    resource.MustParse("6000m"),
					"memory": resource.MustParse("6Gi"),
				},
				Inheritance: map[string]bool{
					"rbac":           true,
					"networkpolicy":  true,
					"limitrange":     false,
					"configmap":      false,
					"secret":         false,
					"serviceaccount": false,
				},
				Scope: "local",
			},
		},
	}

	g.tenantObj = tenantObj
	g.trqObj = trqObj
	g.resourceQuotaObj = resourceQuotaObj
	g.subNamespaceObj = subNamespaceObj

	// Imitate tenant creation processes
	_, err := edgenetclientset.CoreV1alpha1().Tenants().Create(context.TODO(), g.tenantObj.DeepCopy(), metav1.CreateOptions{})
	if err == nil {
		tenantCoreNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: g.tenantObj.GetName()}}
		namespaceLabels := map[string]string{"edge-net.io/kind": "core", "edge-net.io/tenant": g.tenantObj.GetName()}
		tenantCoreNamespace.SetLabels(namespaceLabels)
		kubeclientset.CoreV1().Namespaces().Create(context.TODO(), tenantCoreNamespace, metav1.CreateOptions{})
		edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Create(context.TODO(), g.trqObj.DeepCopy(), metav1.CreateOptions{})
		kubeclientset.CoreV1().ResourceQuotas(tenantCoreNamespace.GetName()).Create(context.TODO(), g.resourceQuotaObj.DeepCopy(), metav1.CreateOptions{})
		kubeclientset.RbacV1().Roles(tenantCoreNamespace.GetName()).Create(context.TODO(), &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: "edgenet-test"}}, metav1.CreateOptions{})
		kubeclientset.RbacV1().RoleBindings(tenantCoreNamespace.GetName()).Create(context.TODO(), &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "edgenet-test"}}, metav1.CreateOptions{})
		kubeclientset.NetworkingV1().NetworkPolicies(tenantCoreNamespace.GetName()).Create(context.TODO(), &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: "edgenet-test"}}, metav1.CreateOptions{})
	}
}

func TestStartController(t *testing.T) {
	g := TestGroup{}
	g.Init()

	coreResourceQuota, err := kubeclientset.CoreV1().ResourceQuotas(g.tenantObj.GetName()).Get(context.TODO(), fmt.Sprintf("core-quota"), metav1.GetOptions{})
	util.OK(t, err)
	coreQuotaCPU := coreResourceQuota.Spec.Hard.Cpu().Value()
	coreQuotaMemory := coreResourceQuota.Spec.Hard.Memory().Value()

	// Create a subnamespace
	subNamespaceControllerTest := g.subNamespaceObj.DeepCopy()
	subNamespaceControllerTest.SetName("subnamespace-controller")
	childName := subNamespaceControllerTest.GenerateChildName("")
	_, err = edgenetclientset.CoreV1alpha1().SubNamespaces(g.tenantObj.GetName()).Create(context.TODO(), subNamespaceControllerTest, metav1.CreateOptions{})
	util.OK(t, err)
	// Wait for the status update of the created object
	time.Sleep(750 * time.Millisecond)
	// Get the object and check the status
	//aa, err := edgenetclientset.CoreV1alpha1().SubNamespaces(g.tenantObj.GetName()).Get(context.TODO(), subNamespaceControllerTest.GetName(), metav1.GetOptions{})
	//log.Println(aa.Status)
	//util.OK(t, err)

	_, err = kubeclientset.CoreV1().Namespaces().Get(context.TODO(), childName, metav1.GetOptions{})
	util.OK(t, err)
	tunedCoreResourceQuota, err := kubeclientset.CoreV1().ResourceQuotas(g.tenantObj.GetName()).Get(context.TODO(), fmt.Sprintf("core-quota"), metav1.GetOptions{})
	util.OK(t, err)
	tunedCoreQuotaCPU := tunedCoreResourceQuota.Spec.Hard.Cpu().Value()
	tunedCoreQuotaMemory := tunedCoreResourceQuota.Spec.Hard.Memory().Value()

	cpuResource := subNamespaceControllerTest.Spec.Workspace.ResourceAllocation["cpu"]
	cpuDemand := cpuResource.Value()
	memoryResource := subNamespaceControllerTest.Spec.Workspace.ResourceAllocation["memory"]
	memoryDemand := memoryResource.Value()

	util.Equals(t, coreQuotaCPU-cpuDemand, tunedCoreQuotaCPU)
	util.Equals(t, coreQuotaMemory-memoryDemand, tunedCoreQuotaMemory)

	subResourceQuota, err := kubeclientset.CoreV1().ResourceQuotas(childName).Get(context.TODO(), fmt.Sprintf("sub-quota"), metav1.GetOptions{})
	util.OK(t, err)
	subQuotaCPU := subResourceQuota.Spec.Hard.Cpu().Value()
	subQuotaMemory := subResourceQuota.Spec.Hard.Memory().Value()
	util.Equals(t, int64(6), subQuotaCPU)
	util.Equals(t, int64(6442450944), subQuotaMemory)

	subNamespaceControllerNestedTest := g.subNamespaceObj.DeepCopy()
	subNamespaceControllerNestedTest.SetUID("subnamespace-controller-nested")
	subNamespaceControllerNestedTest.Spec.Workspace.ResourceAllocation["cpu"] = resource.MustParse("1000m")
	subNamespaceControllerNestedTest.Spec.Workspace.ResourceAllocation["memory"] = resource.MustParse("1Gi")
	subNamespaceControllerNestedTest.SetName("subnamespace-controller-nested")
	subNamespaceControllerNestedTest.SetNamespace(childName)
	nestedChildName := subNamespaceControllerNestedTest.GenerateChildName("")
	_, err = edgenetclientset.CoreV1alpha1().SubNamespaces(subNamespaceControllerNestedTest.GetNamespace()).Create(context.TODO(), subNamespaceControllerNestedTest, metav1.CreateOptions{})
	util.OK(t, err)
	// Wait for the status update of the created object
	time.Sleep(750 * time.Millisecond)

	subResourceQuota, err = kubeclientset.CoreV1().ResourceQuotas(childName).Get(context.TODO(), fmt.Sprintf("sub-quota"), metav1.GetOptions{})
	util.OK(t, err)
	subQuotaCPU = subResourceQuota.Spec.Hard.Cpu().Value()
	subQuotaMemory = subResourceQuota.Spec.Hard.Memory().Value()
	util.Equals(t, int64(5), subQuotaCPU)
	util.Equals(t, int64(5368709120), subQuotaMemory)

	tunedCoreResourceQuota, err = kubeclientset.CoreV1().ResourceQuotas(g.tenantObj.GetName()).Get(context.TODO(), fmt.Sprintf("core-quota"), metav1.GetOptions{})
	util.OK(t, err)
	tunedCoreQuotaCPU = tunedCoreResourceQuota.Spec.Hard.Cpu().Value()
	tunedCoreQuotaMemory = tunedCoreResourceQuota.Spec.Hard.Memory().Value()
	util.Equals(t, int64(2), tunedCoreQuotaCPU)
	util.Equals(t, int64(2147483648), tunedCoreQuotaMemory)

	nestedSubResourceQuota, err := kubeclientset.CoreV1().ResourceQuotas(nestedChildName).Get(context.TODO(), fmt.Sprintf("sub-quota"), metav1.GetOptions{})
	util.OK(t, err)
	nestedSubQuotaCPU := nestedSubResourceQuota.Spec.Hard.Cpu().Value()
	nestedSubQuotaMemory := nestedSubResourceQuota.Spec.Hard.Memory().Value()
	util.Equals(t, int64(1), nestedSubQuotaCPU)
	util.Equals(t, int64(1073741824), nestedSubQuotaMemory)

	err = edgenetclientset.CoreV1alpha1().SubNamespaces(g.tenantObj.GetName()).Delete(context.TODO(), subNamespaceControllerTest.GetName(), metav1.DeleteOptions{})
	util.OK(t, err)
	time.Sleep(450 * time.Millisecond)
	_, err = kubeclientset.CoreV1().Namespaces().Get(context.TODO(), childName, metav1.GetOptions{})
	util.Equals(t, true, errors.IsNotFound(err))
	latestParentResourceQuota, err := kubeclientset.CoreV1().ResourceQuotas(g.tenantObj.GetName()).Get(context.TODO(), fmt.Sprintf("core-quota"), metav1.GetOptions{})
	util.OK(t, err)
	latestParentQuotaCPU := latestParentResourceQuota.Spec.Hard.Cpu().Value()
	latestParentQuotaMemory := latestParentResourceQuota.Spec.Hard.Memory().Value()
	util.Equals(t, coreQuotaCPU, latestParentQuotaCPU)
	util.Equals(t, coreQuotaMemory, latestParentQuotaMemory)
}

func TestCreate(t *testing.T) {
	g := TestGroup{}
	g.Init()

	subnamespace1 := g.subNamespaceObj.DeepCopy()
	subnamespace1.SetName("all")
	subnamespace1.SetUID("all")
	subnamespace1.Spec.Workspace.ResourceAllocation["cpu"] = resource.MustParse("2000m")
	subnamespace1.Spec.Workspace.ResourceAllocation["memory"] = resource.MustParse("2Gi")
	childName1 := subnamespace1.GenerateChildName("")
	subnamespace1nested := g.subNamespaceObj.DeepCopy()
	subnamespace1nested.SetName("all-nested")
	subnamespace1nested.SetUID("all-nested")
	subnamespace1nested.Spec.Workspace.ResourceAllocation["cpu"] = resource.MustParse("1000m")
	subnamespace1nested.Spec.Workspace.ResourceAllocation["memory"] = resource.MustParse("1Gi")
	subnamespace1nested.SetNamespace(childName1)
	childName1nested := subnamespace1nested.GenerateChildName("")
	subnamespace2 := g.subNamespaceObj.DeepCopy()
	subnamespace2.SetName("rbac")
	subnamespace2.SetUID("rbac")
	subnamespace2.Spec.Workspace.Inheritance["networkpolicy"] = false
	subnamespace2.Spec.Workspace.ResourceAllocation["cpu"] = resource.MustParse("1000m")
	subnamespace2.Spec.Workspace.ResourceAllocation["memory"] = resource.MustParse("1Gi")
	childName2 := subnamespace2.GenerateChildName("")
	subnamespace3 := g.subNamespaceObj.DeepCopy()
	subnamespace3.SetName("networkpolicy")
	subnamespace3.SetUID("networkpolicy")
	subnamespace3.Spec.Workspace.Inheritance["rbac"] = false
	subnamespace3.Spec.Workspace.ResourceAllocation["cpu"] = resource.MustParse("1000m")
	subnamespace3.Spec.Workspace.ResourceAllocation["memory"] = resource.MustParse("1Gi")
	childName3 := subnamespace3.GenerateChildName("")
	subnamespace4 := g.subNamespaceObj.DeepCopy()
	subnamespace4.SetName("expiry")
	subnamespace4.SetUID("expiry")
	subnamespace4.Spec.Workspace.ResourceAllocation["cpu"] = resource.MustParse("1000m")
	subnamespace4.Spec.Workspace.ResourceAllocation["memory"] = resource.MustParse("1Gi")
	childName4 := subnamespace4.GenerateChildName("")

	t.Run("inherit all without expiry date", func(t *testing.T) {
		defer edgenetclientset.CoreV1alpha1().SubNamespaces(g.tenantObj.GetName()).Delete(context.TODO(), subnamespace1.GetName(), metav1.DeleteOptions{})

		_, err := edgenetclientset.CoreV1alpha1().SubNamespaces(g.tenantObj.GetName()).Create(context.TODO(), subnamespace1, metav1.CreateOptions{})
		util.OK(t, err)
		time.Sleep(450 * time.Millisecond)
		childNamespace, err := kubeclientset.CoreV1().Namespaces().Get(context.TODO(), childName1, metav1.GetOptions{})
		util.OK(t, err)

		t.Run("check core resource quota", func(t *testing.T) {
			coreResourceQuota, _ := kubeclientset.CoreV1().ResourceQuotas(g.tenantObj.GetName()).Get(context.TODO(), "core-quota", metav1.GetOptions{})
			util.Equals(t, int64(6), coreResourceQuota.Spec.Hard.Cpu().Value())
			util.Equals(t, int64(6442450944), coreResourceQuota.Spec.Hard.Memory().Value())
		})

		t.Run("check sub resource quota", func(t *testing.T) {
			subResourceQuota, _ := kubeclientset.CoreV1().ResourceQuotas(childNamespace.GetName()).Get(context.TODO(), "sub-quota", metav1.GetOptions{})
			util.Equals(t, int64(2), subResourceQuota.Spec.Hard.Cpu().Value())
			util.Equals(t, int64(2147483648), subResourceQuota.Spec.Hard.Memory().Value())
			t.Run("nested subnamespaces", func(t *testing.T) {
				_, err := edgenetclientset.CoreV1alpha1().SubNamespaces(childNamespace.GetName()).Create(context.TODO(), subnamespace1nested, metav1.CreateOptions{})
				util.OK(t, err)
				time.Sleep(450 * time.Millisecond)
				nestedChildNamespace, err := kubeclientset.CoreV1().Namespaces().Get(context.TODO(), childName1nested, metav1.GetOptions{})
				util.OK(t, err)

				subResourceQuota, _ := kubeclientset.CoreV1().ResourceQuotas(childNamespace.GetName()).Get(context.TODO(), "sub-quota", metav1.GetOptions{})
				util.Equals(t, int64(1), subResourceQuota.Spec.Hard.Cpu().Value())
				util.Equals(t, int64(1073741824), subResourceQuota.Spec.Hard.Memory().Value())

				nestedSubResourceQuota, _ := kubeclientset.CoreV1().ResourceQuotas(nestedChildNamespace.GetName()).Get(context.TODO(), "sub-quota", metav1.GetOptions{})
				util.Equals(t, int64(1), nestedSubResourceQuota.Spec.Hard.Cpu().Value())
				util.Equals(t, int64(1073741824), nestedSubResourceQuota.Spec.Hard.Memory().Value())
			})
		})

		if roleRaw, err := kubeclientset.RbacV1().Roles(subnamespace1.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil {
			for _, roleRow := range roleRaw.Items {
				_, err := kubeclientset.RbacV1().Roles(childNamespace.GetName()).Get(context.TODO(), roleRow.GetName(), metav1.GetOptions{})
				util.OK(t, err)
			}
		}
		if roleBindingRaw, err := kubeclientset.RbacV1().RoleBindings(subnamespace1.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil {
			for _, roleBindingRow := range roleBindingRaw.Items {
				_, err := kubeclientset.RbacV1().RoleBindings(childNamespace.GetName()).Get(context.TODO(), roleBindingRow.GetName(), metav1.GetOptions{})
				util.OK(t, err)
			}
		}
		if networkPolicyRaw, err := kubeclientset.NetworkingV1().NetworkPolicies(subnamespace1.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil {
			for _, networkPolicyRow := range networkPolicyRaw.Items {
				_, err := kubeclientset.NetworkingV1().NetworkPolicies(childNamespace.GetName()).Get(context.TODO(), networkPolicyRow.GetName(), metav1.GetOptions{})
				util.OK(t, err)
			}
		}
	})
	t.Run("inherit rbac without expiry date", func(t *testing.T) {
		defer edgenetclientset.CoreV1alpha1().SubNamespaces(g.tenantObj.GetName()).Delete(context.TODO(), subnamespace2.GetName(), metav1.DeleteOptions{})

		_, err := edgenetclientset.CoreV1alpha1().SubNamespaces(g.tenantObj.GetName()).Create(context.TODO(), subnamespace2, metav1.CreateOptions{})
		util.OK(t, err)
		time.Sleep(450 * time.Millisecond)
		childNamespace, err := kubeclientset.CoreV1().Namespaces().Get(context.TODO(), childName2, metav1.GetOptions{})
		util.OK(t, err)
		if roleRaw, err := kubeclientset.RbacV1().Roles(subnamespace2.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespace2.Spec.Workspace.Inheritance["rbac"] {
			for _, roleRow := range roleRaw.Items {
				_, err := kubeclientset.RbacV1().Roles(childNamespace.GetName()).Get(context.TODO(), roleRow.GetName(), metav1.GetOptions{})
				util.OK(t, err)
			}
		}
		if roleBindingRaw, err := kubeclientset.RbacV1().RoleBindings(subnamespace2.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespace2.Spec.Workspace.Inheritance["rbac"] {
			for _, roleBindingRow := range roleBindingRaw.Items {
				_, err := kubeclientset.RbacV1().RoleBindings(childNamespace.GetName()).Get(context.TODO(), roleBindingRow.GetName(), metav1.GetOptions{})
				util.OK(t, err)
			}
		}
		if networkPolicyRaw, err := kubeclientset.NetworkingV1().NetworkPolicies(subnamespace2.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespace2.Spec.Workspace.Inheritance["networkpolicy"] {
			for _, networkPolicyRow := range networkPolicyRaw.Items {
				_, err := kubeclientset.NetworkingV1().NetworkPolicies(childNamespace.GetName()).Get(context.TODO(), networkPolicyRow.GetName(), metav1.GetOptions{})
				util.Equals(t, true, errors.IsNotFound(err))
			}
		}
	})
	t.Run("inherit networkpolicy without expiry date", func(t *testing.T) {
		defer edgenetclientset.CoreV1alpha1().SubNamespaces(g.tenantObj.GetName()).Delete(context.TODO(), subnamespace3.GetName(), metav1.DeleteOptions{})

		_, err := edgenetclientset.CoreV1alpha1().SubNamespaces(g.tenantObj.GetName()).Create(context.TODO(), subnamespace3, metav1.CreateOptions{})
		util.OK(t, err)
		time.Sleep(450 * time.Millisecond)
		childNamespace, err := kubeclientset.CoreV1().Namespaces().Get(context.TODO(), childName3, metav1.GetOptions{})
		util.OK(t, err)
		if roleRaw, err := kubeclientset.RbacV1().Roles(subnamespace3.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespace3.Spec.Workspace.Inheritance["rbac"] {
			for _, roleRow := range roleRaw.Items {
				_, err := kubeclientset.RbacV1().Roles(childNamespace.GetName()).Get(context.TODO(), roleRow.GetName(), metav1.GetOptions{})
				util.Equals(t, true, errors.IsNotFound(err))
			}
		}
		if roleBindingRaw, err := kubeclientset.RbacV1().RoleBindings(subnamespace3.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespace3.Spec.Workspace.Inheritance["rbac"] {
			for _, roleBindingRow := range roleBindingRaw.Items {
				_, err := kubeclientset.RbacV1().RoleBindings(childNamespace.GetName()).Get(context.TODO(), roleBindingRow.GetName(), metav1.GetOptions{})
				util.Equals(t, true, errors.IsNotFound(err))
			}
		}
		if networkPolicyRaw, err := kubeclientset.NetworkingV1().NetworkPolicies(subnamespace3.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespace3.Spec.Workspace.Inheritance["networkpolicy"] {
			for _, networkPolicyRow := range networkPolicyRaw.Items {
				_, err := kubeclientset.NetworkingV1().NetworkPolicies(childNamespace.GetName()).Get(context.TODO(), networkPolicyRow.GetName(), metav1.GetOptions{})
				util.OK(t, err)
			}
		}
	})
	t.Run("inherit all with expiry date", func(t *testing.T) {
		subnamespace4.Spec.Expiry = &metav1.Time{
			Time: time.Now().Add(500 * time.Millisecond),
		}
		_, err := edgenetclientset.CoreV1alpha1().SubNamespaces(g.tenantObj.GetName()).Create(context.TODO(), subnamespace4, metav1.CreateOptions{})
		util.OK(t, err)
		time.Sleep(450 * time.Millisecond)
		_, err = kubeclientset.CoreV1().Namespaces().Get(context.TODO(), childName4, metav1.GetOptions{})
		util.OK(t, err)
		time.Sleep(500 * time.Millisecond)
		_, err = kubeclientset.CoreV1().Namespaces().Get(context.TODO(), childName4, metav1.GetOptions{})
		util.Equals(t, true, errors.IsNotFound(err))
	})
}

func TestQuota(t *testing.T) {
	g := TestGroup{}
	g.Init()

	subnamespace1 := g.subNamespaceObj.DeepCopy()
	subnamespace1.SetName("all-quota")
	subnamespace1.SetUID("all-quota")
	childName1 := subnamespace1.GenerateChildName("")
	subnamespace2 := g.subNamespaceObj.DeepCopy()
	subnamespace2.SetName("rbac-quota")
	subnamespace2.SetUID("rbac-quota")
	subnamespace2.Spec.Workspace.Inheritance["networkpolicy"] = false
	childName2 := subnamespace2.GenerateChildName("")
	subnamespace3 := g.subNamespaceObj.DeepCopy()
	subnamespace3.SetName("networkpolicy-quota")
	subnamespace3.SetUID("networkpolicy-quota")
	subnamespace3.Spec.Workspace.Inheritance["rbac"] = false
	childName3 := subnamespace3.GenerateChildName("")

	_, err := edgenetclientset.CoreV1alpha1().SubNamespaces(g.tenantObj.GetName()).Create(context.TODO(), subnamespace1, metav1.CreateOptions{})
	util.OK(t, err)
	time.Sleep(750 * time.Millisecond)
	_, err = kubeclientset.CoreV1().Namespaces().Get(context.TODO(), childName1, metav1.GetOptions{})
	util.OK(t, err)

	_, err = edgenetclientset.CoreV1alpha1().SubNamespaces(g.tenantObj.GetName()).Create(context.TODO(), subnamespace2, metav1.CreateOptions{})
	util.OK(t, err)
	time.Sleep(450 * time.Millisecond)
	_, err = kubeclientset.CoreV1().Namespaces().Get(context.TODO(), childName2, metav1.GetOptions{})
	util.Equals(t, true, errors.IsNotFound(err))

	_, err = edgenetclientset.CoreV1alpha1().SubNamespaces(g.tenantObj.GetName()).Create(context.TODO(), subnamespace3, metav1.CreateOptions{})
	util.OK(t, err)
	time.Sleep(450 * time.Millisecond)
	_, err = kubeclientset.CoreV1().Namespaces().Get(context.TODO(), childName3, metav1.GetOptions{})
	util.Equals(t, true, errors.IsNotFound(err))
}
