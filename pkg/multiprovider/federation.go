package multiprovider

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	appsv1alpha2 "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha2"
	federationv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/federation/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

// SetupRemoteAccessCredentials creates a service account, a secret, and required permissions for the remote cluster to access the federation manager
func (m *Manager) SetupRemoteAccessCredentials(name, namespace, clusterRole string) error {
	// Below is to create a service account that will be used to access the home cluster remotely
	serviceAccount := new(corev1.ServiceAccount)
	serviceAccount.SetName(name)
	serviceAccount.SetNamespace(namespace)
	if _, err := m.kubeclientset.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), serviceAccount, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
		klog.Infoln(err)
		return err
	}
	// A secret that is tied to the service account creates a token to be consumed by the remote cluster
	authSecret := new(corev1.Secret)
	authSecret.Name = name
	authSecret.Namespace = namespace
	authSecret.Type = corev1.SecretTypeServiceAccountToken
	authSecret.Annotations = map[string]string{"kubernetes.io/service-account.name": serviceAccount.GetName()}
	if _, err := m.kubeclientset.CoreV1().Secrets(namespace).Create(context.TODO(), authSecret, metav1.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
		if secret, err := m.kubeclientset.CoreV1().Secrets(namespace).Get(context.TODO(), authSecret.GetName(), metav1.GetOptions{}); err == nil {
			if secret.Type != corev1.SecretTypeServiceAccountToken || secret.Annotations["kubernetes.io/service-account.name"] != serviceAccount.GetName() {
				secret.Type = authSecret.Type
				secret.Annotations = authSecret.Annotations
				if _, err := m.kubeclientset.CoreV1().Secrets(namespace).Update(context.TODO(), secret, metav1.UpdateOptions{}); err != nil {
					return err
				}
			}
		} else {
			return err
		}
	}
	// This part binds a ClusterRole to the service account to grant the predefined permissions to the serviceaccount
	roleRef := rbacv1.RoleRef{Kind: "ClusterRole", Name: clusterRole}
	rbSubjects := []rbacv1.Subject{{Kind: "ServiceAccount", Name: serviceAccount.GetName(), Namespace: serviceAccount.GetNamespace()}}
	roleBind := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-%s", clusterRole, name), Namespace: serviceAccount.GetNamespace()},
		Subjects: rbSubjects, RoleRef: roleRef}
	roleBindLabels := map[string]string{"edge-net.io/generated": "true"}
	roleBind.SetLabels(roleBindLabels)
	if _, err := m.kubeclientset.RbacV1().RoleBindings(namespace).Create(context.TODO(), roleBind, metav1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			if roleBinding, err := m.kubeclientset.RbacV1().RoleBindings(namespace).Get(context.TODO(), roleBind.GetName(), metav1.GetOptions{}); err == nil {
				roleBinding.RoleRef = roleBind.RoleRef
				roleBinding.Subjects = roleBind.Subjects
				roleBinding.SetLabels(roleBind.GetLabels())
				if _, err := m.kubeclientset.RbacV1().RoleBindings(namespace).Update(context.TODO(), roleBinding, metav1.UpdateOptions{}); err == nil {
					return nil
				}
			}
		}
		klog.Infoln(err)
		return err
	}
	return nil
}

// PrepareSecretForRemoteCluster prepares a secret from the secret of access credentials, which will be consumed by the remote cluster
func (m *Manager) PrepareSecretForRemoteCluster(name, namespace, clusterUID, remoteName, remoteNamespace string) (*corev1.Secret, bool, error) {
	// Get the secret of the access credentials that is already prepared
	authSecret, err := m.kubeclientset.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, false, err
	}
	// Check if the token is missing. If it is missing and the secret is older than a minute, return an error.
	// If it is missing and the secret is not older than a minute, return an error and enqueue message to try it again after a minute.
	if authSecret.Data["token"] == nil {
		if authSecret.GetCreationTimestamp().Add(1 * time.Minute).Before(time.Now()) {
			return nil, true, fmt.Errorf("token is missing")
		}
		return nil, false, fmt.Errorf("token is missing")
	}
	remoteSecret := new(corev1.Secret)
	remoteSecret.SetName(remoteName)
	remoteSecret.SetNamespace(remoteNamespace)
	remoteSecret.Data = make(map[string][]byte)
	remoteSecret.Data["token"] = authSecret.Data["token"]
	remoteSecret.Data["ca.crt"] = authSecret.Data["ca.crt"]
	remoteSecret.Data["namespace"] = []byte(namespace)
	var authentication string
	if authentication = strings.TrimSpace(os.Getenv("AUTHENTICATION_STRATEGY")); authentication != "kubeconfig" {
		authentication = "serviceaccount"
	}
	// TODO: This part needs to be changed to support multiple control plane nodes
	address, _ := m.GetClusterAddressWithLocation()
	remoteSecret.Data["server"] = []byte(address)
	// We need to add the cluster UID to the secret to be able to identify the parent cluster.
	// As we deploy this secret to the remote cluster, current cluster UID becomes remote to that cluster.
	remoteSecret.Data["remote-cluster-uid"] = []byte(clusterUID)
	return remoteSecret, false, nil
}

// GetServerAddress retrieves the server address of the cluster from a control plane node
func (m *Manager) GetClusterAddressWithLocation() (string, string) {
	// TODO: This part needs to be changed to support multiple control plane nodes
	var address string
	var location string
	nodeRaw, _ := m.kubeclientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: "node-role.kubernetes.io/control-plane"})
	for _, node := range nodeRaw.Items {
		if internal, external := GetNodeIPAddresses(node.DeepCopy()); external == "" && internal == "" {
			continue
		} else if external != "" {
			address = external + ":8443"
		} else {
			address = internal + ":8443"
		}
		labels := node.GetLabels()
		location = fmt.Sprintf("%s/%s", labels["edge-net.io/country"], labels["edge-net.io/city"])
		break
	}
	return address, location
}

// DeploySecret deploys a secret to the remote cluster
func (m *Manager) DeploySecret(secret *corev1.Secret) error {
	// Using the remote kube clientset, create/update the secret in the remote cluster
	if _, err := m.remotekubeclientset.CoreV1().Secrets(secret.GetNamespace()).Create(context.TODO(), secret, metav1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			klog.Infoln(err)
			if currentSecret, err := m.remotekubeclientset.CoreV1().Secrets(secret.GetNamespace()).Get(context.TODO(), secret.GetName(), metav1.GetOptions{}); err == nil {
				if reflect.DeepEqual(currentSecret.Data, secret.Data) {
					return nil
				}
				currentSecret.Data = secret.Data
				if _, err = m.remotekubeclientset.CoreV1().Secrets(secret.GetNamespace()).Update(context.TODO(), currentSecret, metav1.UpdateOptions{}); err == nil {
					return nil
				} else {
					klog.Infoln(err)
				}
			}
		}
		klog.Infoln(err)
		return err
	}
	return nil
}

// AnchorSelectiveDeployment creates a selective deployment anchor in its federation manager
func (m *Manager) AnchorSelectiveDeployment(selectivedeployment *appsv1alpha2.SelectiveDeployment, propagationNamespace, secretName string) error {
	selectivedeploymentanchor := new(federationv1alpha1.SelectiveDeploymentAnchor)
	selectivedeploymentanchor.SetName(string(selectivedeployment.GetUID()))
	selectivedeploymentanchor.SetNamespace(propagationNamespace)
	selectivedeploymentanchor.Spec.ClusterAffinity = selectivedeployment.Spec.ClusterAffinity
	selectivedeploymentanchor.Spec.ClusterReplicas = selectivedeployment.Spec.ClusterReplicas
	selectivedeploymentanchor.Spec.OriginRef.Name = selectivedeployment.GetName()
	selectivedeploymentanchor.Spec.OriginRef.Namespace = selectivedeployment.GetNamespace()
	selectivedeploymentanchor.Spec.OriginRef.UID = string(selectivedeployment.GetUID())
	selectivedeploymentanchor.Spec.SecretName = secretName
	if _, err := m.remoteedgeclientset.FederationV1alpha1().SelectiveDeploymentAnchors(selectivedeploymentanchor.GetNamespace()).Create(context.TODO(), selectivedeploymentanchor, metav1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			klog.Infoln(err)
			if currentSelectiveDeploymentAnchor, err := m.remoteedgeclientset.FederationV1alpha1().SelectiveDeploymentAnchors(selectivedeploymentanchor.GetNamespace()).Get(context.TODO(), selectivedeploymentanchor.GetName(), metav1.GetOptions{}); err == nil {
				if reflect.DeepEqual(currentSelectiveDeploymentAnchor.Spec, selectivedeploymentanchor.Spec) {
					return nil
				}
				currentSelectiveDeploymentAnchor.Spec = selectivedeploymentanchor.Spec
				if _, err := m.remoteedgeclientset.FederationV1alpha1().SelectiveDeploymentAnchors(selectivedeploymentanchor.GetNamespace()).Update(context.TODO(), currentSelectiveDeploymentAnchor, metav1.UpdateOptions{}); err == nil {
					return nil
				} else {
					klog.Infoln(err)
				}
			}
		}
		klog.Infoln(err)
		return err
	}
	return nil
}

// UpdateSelectiveDeploymentClusterStatus updates the status of originating selective deployment in its originating cluster
func (m *Manager) UpdateSelectiveDeploymentClusterStatus(name, namespace, clusterUID string, status appsv1alpha2.WorkloadClusterStatus) bool {
	originatingSelectivedeployment, err := m.remoteedgeclientset.AppsV1alpha2().SelectiveDeployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		klog.Infoln(err)
		return false
	}
	updateStatus := true
	if originatingSelectivedeployment.Status.Clusters != nil {
		if originatingStatus, elementExists := originatingSelectivedeployment.Status.Clusters[clusterUID]; elementExists {
			if reflect.DeepEqual(originatingStatus, status) {
				updateStatus = false
			}
		}
	} else {
		originatingSelectivedeployment.Status.Clusters = make(map[string]appsv1alpha2.WorkloadClusterStatus)
	}
	if updateStatus {
		originatingSelectivedeployment.Status.Clusters[clusterUID] = status
		if _, err := m.remoteedgeclientset.AppsV1alpha2().SelectiveDeployments(namespace).UpdateStatus(context.TODO(), originatingSelectivedeployment, metav1.UpdateOptions{}); err != nil {
			klog.Infoln(err)
			return false
		}
	}
	return true
}

// CreateManagerCache creates a manager cache in the remote cluster
func (m *Manager) CreateManagerCache(managerCache *federationv1alpha1.ManagerCache) error {
	if _, err := m.remoteedgeclientset.FederationV1alpha1().ManagerCaches().Create(context.TODO(), managerCache, metav1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			if currentRemoteManagerCache, err := m.remoteedgeclientset.FederationV1alpha1().ManagerCaches().Get(context.TODO(), managerCache.GetName(), metav1.GetOptions{}); err == nil {
				currentRemoteManagerCache.Spec = managerCache.Spec
				if _, err := m.remoteedgeclientset.FederationV1alpha1().ManagerCaches().Update(context.TODO(), currentRemoteManagerCache, metav1.UpdateOptions{}); err == nil {
					return nil
				}
			}
		}
		return err
	}
	return nil
}

// DisableChildrenManagers closes the children managers to the federation
func (m *Manager) DisableChildrenManagers() error {
	if clusterRaw, err := m.remoteedgeclientset.FederationV1alpha1().Clusters("").List(context.TODO(), metav1.ListOptions{}); err == nil {
		for _, clusterRow := range clusterRaw.Items {
			if clusterRow.Spec.Role == federationv1alpha1.FederationManagerRole && !clusterRow.Spec.Enabled {
				clusterRow.Spec.Enabled = false
				// TODO: Error handling
				m.remoteedgeclientset.FederationV1alpha1().Clusters(clusterRow.GetNamespace()).Update(context.TODO(), &clusterRow, metav1.UpdateOptions{})
			}
		}
	} else {
		return err
	}
	return nil
}
