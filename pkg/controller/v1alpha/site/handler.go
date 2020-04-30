package site

import (
	"fmt"

	apps_v1alpha "headnode/pkg/apis/apps/v1alpha"
	"headnode/pkg/authorization"
	"headnode/pkg/client/clientset/versioned"

	log "github.com/Sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// HandlerInterface interface contains the methods that are required
type HandlerInterface interface {
	Init() error
	ObjectCreated(obj interface{})
	ObjectUpdated(obj interface{})
	ObjectDeleted(obj interface{})
}

// Handler implementation
type Handler struct {
	clientset        *kubernetes.Clientset
	edgenetClientset *versioned.Clientset
	resourceQuota    *corev1.ResourceQuota
}

// Init handles any handler initialization
func (t *Handler) Init() error {
	log.Info("SiteHandler.Init")
	var err error
	t.clientset, err = authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	t.edgenetClientset, err = authorization.CreateEdgeNetClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	t.resourceQuota = &corev1.ResourceQuota{}
	t.resourceQuota.Name = "site-quota"
	t.resourceQuota.Spec = corev1.ResourceQuotaSpec{
		Hard: map[corev1.ResourceName]resource.Quantity{
			"cpu":                           resource.MustParse("5m"),
			"memory":                        resource.MustParse("1Mi"),
			"requests.storage":              resource.MustParse("1Mi"),
			"pods":                          resource.Quantity{Format: "0"},
			"count/persistentvolumeclaims":  resource.Quantity{Format: "0"},
			"count/services":                resource.Quantity{Format: "0"},
			"count/configmaps":              resource.Quantity{Format: "0"},
			"count/replicationcontrollers":  resource.Quantity{Format: "0"},
			"count/deployments.apps":        resource.Quantity{Format: "0"},
			"count/deployments.extensions":  resource.Quantity{Format: "0"},
			"count/replicasets.apps":        resource.Quantity{Format: "0"},
			"count/replicasets.extensions":  resource.Quantity{Format: "0"},
			"count/statefulsets.apps":       resource.Quantity{Format: "0"},
			"count/statefulsets.extensions": resource.Quantity{Format: "0"},
			"count/jobs.batch":              resource.Quantity{Format: "0"},
			"count/cronjobs.batch":          resource.Quantity{Format: "0"},
		},
	}
	return err
}

// ObjectCreated is called when an object is created
func (t *Handler) ObjectCreated(obj interface{}) {
	log.Info("SiteHandler.ObjectCreated")
	// Create a copy of the site object to make changes on it
	siteCopy := obj.(*apps_v1alpha.Site).DeepCopy()

	if siteCopy.GetGeneration() == 1 {
		// Create a cluster role to be used by site users
		policyRule := []rbacv1.PolicyRule{{APIGroups: []string{"apps.edgenet.io"}, Resources: []string{"sites"}, ResourceNames: []string{siteCopy.GetName()}, Verbs: []string{"get"}}}
		siteRole := &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("site-%s", siteCopy.GetName())}, Rules: policyRule}
		_, err := t.clientset.RbacV1().ClusterRoles().Create(siteRole)
		if err != nil {
			log.Infof("Couldn't create site-%s role: %s", siteCopy.GetName(), err)
		}

		// Automatically enable site and update site status
		siteCopy.Status.Enabled = true
		defer t.edgenetClientset.AppsV1alpha().Sites().UpdateStatus(siteCopy)

		// Automatically creates a namespace to host users, slices, and projects
		// When a site is deleted, the owner references feature allows the namespace to be automatically removed
		siteChildNamespaceOwnerReferences := t.setOwnerReferences(siteCopy)
		// Every namespace of a site has the prefix as "site" to provide singularity
		siteChildNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("site-%s", siteCopy.GetName()), OwnerReferences: siteChildNamespaceOwnerReferences}}
		// Namespace labels indicate this namespace created by a site, not by a project or slice
		namespaceLabels := map[string]string{"owner": "site", "owner-name": siteCopy.GetName(), "site-name": siteCopy.GetName()}
		siteChildNamespace.SetLabels(namespaceLabels)
		siteChildNamespaceCreated, _ := t.clientset.CoreV1().Namespaces().Create(siteChildNamespace)

		// Create the resource quota to ban users from using this namespace for their applications
		t.clientset.CoreV1().ResourceQuotas(siteChildNamespaceCreated.GetName()).Create(t.resourceQuota)
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj interface{}) {
	log.Info("SiteHandler.ObjectUpdated")
	// Create a copy of the site object to make changes on it
	siteCopy := obj.(*apps_v1alpha.Site).DeepCopy()

	// Check whether the site disabled
	if siteCopy.Status.Enabled == false {
		// Delete all RoleBindings, Projects, and Slices in the namespace of site
		t.edgenetClientset.AppsV1alpha().Slices(fmt.Sprintf("site-%s", siteCopy.GetName())).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
		t.edgenetClientset.AppsV1alpha().Projects(fmt.Sprintf("site-%s", siteCopy.GetName())).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
		t.clientset.RbacV1().RoleBindings(fmt.Sprintf("site-%s", siteCopy.GetName())).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})

		// List all site users to deactivate
		usersRaw, _ := t.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("site-%s", siteCopy.GetName())).List(metav1.ListOptions{})
		for _, user := range usersRaw.Items {
			userCopy := user.DeepCopy()
			userCopy.Status.Active = false
			t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).UpdateStatus(userCopy)
		}
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("SiteHandler.ObjectDeleted")
	// Delete or disable nodes added by site, TBD.
}

// setOwnerReferences
func (t *Handler) setOwnerReferences(siteCopy *apps_v1alpha.Site) []metav1.OwnerReference {
	ownerReferences := []metav1.OwnerReference{}
	newNamespaceRef := *metav1.NewControllerRef(siteCopy, apps_v1alpha.SchemeGroupVersion.WithKind("Site"))
	takeControl := false
	newNamespaceRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newNamespaceRef)
	return ownerReferences
}
