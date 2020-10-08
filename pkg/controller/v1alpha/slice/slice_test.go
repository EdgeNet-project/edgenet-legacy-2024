package slice

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// Constant variables for events
const success = "Successful"

// The main structure of test group
type TestGroup struct {
	authorityObj  apps_v1alpha.Authority
	TRQObj        apps_v1alpha.TotalResourceQuota
	userObj       apps_v1alpha.User
	sliceObj      apps_v1alpha.Slice
	client        kubernetes.Interface
	edgenetClient versioned.Interface
	handler       Handler
}

func TestMain(m *testing.M) {
	flag.String("dir", "../../../..", "Override the directory.")
	flag.String("smtp-path", "../../../../configs/smtp_test.yaml", "Set SMTP path.")
	flag.Parse()

	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

// Init syncs the test group
func (g *TestGroup) Init() {
	authorityObj := apps_v1alpha.Authority{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Authority",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
		},
		Spec: apps_v1alpha.AuthoritySpec{
			FullName:  "EdgeNet",
			ShortName: "EdgeNet",
			URL:       "https://www.edge-net.org",
			Address: apps_v1alpha.Address{
				City:    "Paris - NY - CA",
				Country: "France - US",
				Street:  "4 place Jussieu, boite 169",
				ZIP:     "75005",
			},
			Contact: apps_v1alpha.Contact{
				Email:     "john.doe@edge-net.org",
				FirstName: "John",
				LastName:  "Doe",
				Phone:     "+33NUMBER",
				Username:  "johndoe",
			},
			Enabled: true,
		},
	}
	TRQObj := apps_v1alpha.TotalResourceQuota{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TotalResourceQuota",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
		},
		Spec: apps_v1alpha.TotalResourceQuotaSpec{
			Enabled: true,
			Claim: []apps_v1alpha.TotalResourceDetails{
				apps_v1alpha.TotalResourceDetails{
					Name:   "Default",
					CPU:    "12000m",
					Memory: "12Gi",
				},
			},
		},
		Status: apps_v1alpha.TotalResourceQuotaStatus{
			Exceeded: false,
		},
	}
	userObj := apps_v1alpha.User{
		TypeMeta: metav1.TypeMeta{
			Kind:       "User",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "joepublic",
			Namespace:  "authority-edgenet",
			UID:        "TestUID",
			Generation: 1,
		},
		Spec: apps_v1alpha.UserSpec{
			FirstName: "Joe",
			LastName:  "Public",
			Email:     "joe.public@edge-net.org",
			Active:    true,
		},
		Status: apps_v1alpha.UserStatus{
			Type: "user",
			AUP:  true,
		},
	}
	sliceObj := apps_v1alpha.Slice{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Slice",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "slice",
			Namespace: "authority-edgenet",
		},
		Spec: apps_v1alpha.SliceSpec{
			Profile:     "High",
			Users:       []apps_v1alpha.SliceUsers{},
			Description: "This is a description",
		},
		Status: apps_v1alpha.SliceStatus{
			Expires: nil,
		},
	}
	g.authorityObj = authorityObj
	g.TRQObj = TRQObj
	g.userObj = userObj
	g.sliceObj = sliceObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetClient = edgenettestclient.NewSimpleClientset()
	// Imitate authority creation processes
	g.edgenetClient.AppsV1alpha().Authorities().Create(context.TODO(), g.authorityObj.DeepCopy(), metav1.CreateOptions{})
	namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("authority-%s", g.authorityObj.GetName())}}
	namespaceLabels := map[string]string{"owner": "authority", "owner-name": g.authorityObj.GetName(), "authority-name": g.authorityObj.GetName()}
	namespace.SetLabels(namespaceLabels)
	g.client.CoreV1().Namespaces().Create(context.TODO(), &namespace, metav1.CreateOptions{})
	g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Create(context.TODO(), g.TRQObj.DeepCopy(), metav1.CreateOptions{})
	// Create a user as admin on authority
	user := apps_v1alpha.User{}
	user.SetName(strings.ToLower(g.authorityObj.Spec.Contact.Username))
	user.Spec.Email = g.authorityObj.Spec.Contact.Email
	user.Spec.FirstName = g.authorityObj.Spec.Contact.FirstName
	user.Spec.LastName = g.authorityObj.Spec.Contact.LastName
	user.Spec.Active = true
	user.Status.AUP = true
	user.Status.Type = "admin"
	g.edgenetClient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), user.DeepCopy(), metav1.CreateOptions{})
}

func TestHandlerInit(t *testing.T) {
	// Sync the test group
	g := TestGroup{}
	g.Init()
	// Initialize the handler
	g.handler.Init(g.client, g.edgenetClient)
	util.Equals(t, g.client, g.handler.clientset)
	util.Equals(t, g.edgenetClient, g.handler.edgenetClientset)
	util.Equals(t, "slice-low-quota", g.handler.lowResourceQuota.Name)
	util.Equals(t, "slice-medium-quota", g.handler.medResourceQuota.Name)
	util.Equals(t, "slice-high-quota", g.handler.highResourceQuota.Name)
	util.NotEquals(t, nil, g.handler.lowResourceQuota.Spec.Hard)
	util.NotEquals(t, nil, g.handler.medResourceQuota.Spec.Hard)
	util.NotEquals(t, nil, g.handler.highResourceQuota.Spec.Hard)
}

func TestSlice(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	// Create a slice
	g.edgenetClient.AppsV1alpha().Slices(g.sliceObj.GetNamespace()).Create(context.TODO(), g.sliceObj.DeepCopy(), metav1.CreateOptions{})
	g.handler.ObjectCreated(g.sliceObj.DeepCopy())
	sliceCopy, err := g.edgenetClient.AppsV1alpha().Slices(g.sliceObj.GetNamespace()).Get(context.TODO(), g.sliceObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	childNamespaceStr := fmt.Sprintf("%s-slice-%s", g.sliceObj.GetNamespace(), g.sliceObj.GetName())
	TRQCopy, _ := g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), g.TRQObj.GetName(), metav1.GetOptions{})
	memoryRes := resource.MustParse(TRQCopy.Spec.Claim[0].Memory)
	memory := memoryRes.Value()
	CPURes := resource.MustParse(TRQCopy.Spec.Claim[0].CPU)
	cpu := CPURes.Value()
	t.Run("namespace", func(t *testing.T) {
		_, err := g.handler.clientset.CoreV1().Namespaces().Get(context.TODO(), childNamespaceStr, metav1.GetOptions{})
		util.OK(t, err)
	})
	t.Run("resource quota", func(t *testing.T) {
		_, err = g.client.CoreV1().ResourceQuotas(childNamespaceStr).List(context.TODO(), metav1.ListOptions{})
		util.OK(t, err)
	})
	t.Run("role bindings", func(t *testing.T) {
		_, err := g.client.RbacV1().RoleBindings(childNamespaceStr).Get(context.TODO(), fmt.Sprintf("authority-%s-%s-slice-%s", g.authorityObj.GetName(), g.authorityObj.Spec.Contact.Username, "admin"), metav1.GetOptions{})
		util.OK(t, err)
	})
	t.Run("set expiry date", func(t *testing.T) {
		expected := metav1.Time{
			Time: time.Now().Add(336 * time.Hour),
		}
		util.Equals(t, expected.Day(), sliceCopy.Status.Expires.Day())
		util.Equals(t, expected.Month(), sliceCopy.Status.Expires.Month())
		util.Equals(t, expected.Year(), sliceCopy.Status.Expires.Year())
	})
	t.Run("consumed quota", func(t *testing.T) {
		TRQCopy, _ := g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), g.TRQObj.GetName(), metav1.GetOptions{})
		CPUPercentage := float64(g.handler.highResourceQuota.Spec.Hard.Cpu().Value()) / float64(cpu) * 100
		memoryPercentage := float64(g.handler.highResourceQuota.Spec.Hard.Memory().Value()) / float64(memory) * 100
		util.Equals(t, CPUPercentage, TRQCopy.Status.Used.CPU)
		util.Equals(t, memoryPercentage, TRQCopy.Status.Used.Memory)
	})
	t.Run("total quota exceeded", func(t *testing.T) {
		slice := g.sliceObj
		slice.SetName("exceed")
		g.edgenetClient.AppsV1alpha().Slices(g.sliceObj.GetNamespace()).Create(context.TODO(), slice.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(slice.DeepCopy())
		_, err := g.edgenetClient.AppsV1alpha().Slices(slice.GetNamespace()).Get(context.TODO(), slice.GetName(), metav1.GetOptions{})
		util.Equals(t, true, errors.IsNotFound(err))
		t.Run("consumed quota", func(t *testing.T) {
			TRQCopy, _ := g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), g.TRQObj.GetName(), metav1.GetOptions{})
			CPUPercentage := float64(g.handler.highResourceQuota.Spec.Hard.Cpu().Value()) / float64(cpu) * 100
			memoryPercentage := float64(g.handler.highResourceQuota.Spec.Hard.Memory().Value()) / float64(memory) * 100
			util.Equals(t, CPUPercentage, TRQCopy.Status.Used.CPU)
			util.Equals(t, memoryPercentage, TRQCopy.Status.Used.Memory)
		})
	})
	t.Run("timeout", func(t *testing.T) {
		go g.handler.runTimeout(sliceCopy)
		sliceCopy.Status.Expires = &metav1.Time{
			Time: time.Now().Add(10 * time.Millisecond),
		}
		g.edgenetClient.AppsV1alpha().Slices(sliceCopy.GetNamespace()).Update(context.TODO(), sliceCopy, metav1.UpdateOptions{})
		time.Sleep(100 * time.Millisecond)
		t.Run("delete slice", func(t *testing.T) {
			_, err := g.edgenetClient.AppsV1alpha().Slices(g.sliceObj.GetNamespace()).Get(context.TODO(), g.sliceObj.GetName(), metav1.GetOptions{})
			util.Equals(t, true, errors.IsNotFound(err))
		})
		t.Run("delete namespace", func(t *testing.T) {
			_, err := g.handler.clientset.CoreV1().Namespaces().Get(context.TODO(), childNamespaceStr, metav1.GetOptions{})
			util.Equals(t, true, errors.IsNotFound(err))
		})
		t.Run("consumed quota", func(t *testing.T) {
			TRQCopy, _ := g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), g.TRQObj.GetName(), metav1.GetOptions{})
			util.Equals(t, float64(0), TRQCopy.Status.Used.CPU)
			util.Equals(t, float64(0), TRQCopy.Status.Used.Memory)
		})
	})
}

func TestUpdate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	g.edgenetClient.AppsV1alpha().Users(g.userObj.GetNamespace()).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
	// Create a slice
	sliceCopy, err := g.edgenetClient.AppsV1alpha().Slices(g.sliceObj.GetNamespace()).Create(context.TODO(), g.sliceObj.DeepCopy(), metav1.CreateOptions{})
	util.OK(t, err)
	g.handler.ObjectCreated(g.sliceObj.DeepCopy())
	childNamespaceStr := fmt.Sprintf("%s-slice-%s", g.sliceObj.GetNamespace(), g.sliceObj.GetName())
	TRQCopy, _ := g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), g.TRQObj.GetName(), metav1.GetOptions{})
	memoryRes := resource.MustParse(TRQCopy.Spec.Claim[0].Memory)
	memory := memoryRes.Value()
	CPURes := resource.MustParse(TRQCopy.Spec.Claim[0].CPU)
	cpu := CPURes.Value()
	// Add new users to slice
	t.Run("add user", func(t *testing.T) {
		sliceCopy.Spec.Users = []apps_v1alpha.SliceUsers{
			{
				Authority: g.authorityObj.GetName(),
				Username:  g.userObj.GetName(),
			},
		}
		_, err := g.client.RbacV1().RoleBindings(childNamespaceStr).Get(context.TODO(), fmt.Sprintf("%s-%s-slice-%s", g.userObj.GetNamespace(), g.userObj.GetName(), "user"), metav1.GetOptions{})
		// Verifying the user is not involved in the beginning
		util.Equals(t, true, errors.IsNotFound(err))
		// Building field parameter
		var field fields
		field.users.status = true
		field.users.added = fmt.Sprintf("`[{\"Authority\": \"%s\", \"Username\": \"%s\" }]`", g.authorityObj.GetName(), g.userObj.GetName())
		// Requesting server to update internal representation of slice
		_, err = g.edgenetClient.AppsV1alpha().Slices(sliceCopy.GetNamespace()).Update(context.TODO(), sliceCopy, metav1.UpdateOptions{})
		util.OK(t, err)
		// Invoking ObjectUpdated to send emails to users removed or added to slice
		g.handler.ObjectUpdated(sliceCopy, field)
		// Check user rolebinding in slice child namespace
		_, err = g.client.RbacV1().RoleBindings(childNamespaceStr).Get(context.TODO(), fmt.Sprintf("%s-%s-slice-%s", g.userObj.GetNamespace(), g.userObj.GetName(), "user"), metav1.GetOptions{})
		// Verifying server created rolebinding for new user in slice's child namespace
		util.OK(t, err)
	})
	t.Run("renew", func(t *testing.T) {
		go g.handler.runTimeout(sliceCopy)
		sliceCopy.Status.Expires = &metav1.Time{
			Time: time.Now().Add(50 * time.Millisecond),
		}
		g.edgenetClient.AppsV1alpha().Slices(sliceCopy.GetNamespace()).Update(context.TODO(), sliceCopy, metav1.UpdateOptions{})
		time.Sleep(10 * time.Millisecond)
		sliceCopy.Spec.Renew = true
		_, err = g.edgenetClient.AppsV1alpha().Slices(sliceCopy.GetNamespace()).Update(context.TODO(), sliceCopy, metav1.UpdateOptions{})
		util.OK(t, err)
		var field fields
		g.handler.ObjectUpdated(sliceCopy, field)
		time.Sleep(100 * time.Millisecond)

		t.Run("save slice", func(t *testing.T) {
			_, err := g.edgenetClient.AppsV1alpha().Slices(g.sliceObj.GetNamespace()).Get(context.TODO(), g.sliceObj.GetName(), metav1.GetOptions{})
			util.Equals(t, false, errors.IsNotFound(err))
		})
		t.Run("save namespace", func(t *testing.T) {
			_, err := g.handler.clientset.CoreV1().Namespaces().Get(context.TODO(), childNamespaceStr, metav1.GetOptions{})
			util.Equals(t, false, errors.IsNotFound(err))
		})
		t.Run("save consumed quota", func(t *testing.T) {
			TRQCopy, _ := g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), g.TRQObj.GetName(), metav1.GetOptions{})
			CPUPercentage := float64(g.handler.highResourceQuota.Spec.Hard.Cpu().Value()) / float64(cpu) * 100
			memoryPercentage := float64(g.handler.highResourceQuota.Spec.Hard.Memory().Value()) / float64(memory) * 100
			util.Equals(t, CPUPercentage, TRQCopy.Status.Used.CPU)
			util.Equals(t, memoryPercentage, TRQCopy.Status.Used.Memory)
		})
	})

	t.Run("change profile", func(t *testing.T) {
		sliceCopy.Spec.Profile = "Low"
		g.edgenetClient.AppsV1alpha().Slices(sliceCopy.GetNamespace()).Update(context.TODO(), sliceCopy, metav1.UpdateOptions{})
		var field fields
		field.profile.old = "High"
		field.profile.status = true
		err := g.client.CoreV1().ResourceQuotas(childNamespaceStr).Delete(context.TODO(), g.handler.highResourceQuota.GetName(), metav1.DeleteOptions{})
		util.OK(t, err)
		g.handler.ObjectUpdated(sliceCopy, field)
		sliceCopy, err := g.edgenetClient.AppsV1alpha().Slices(g.sliceObj.GetNamespace()).Get(context.TODO(), g.sliceObj.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		t.Run("set expiry date", func(t *testing.T) {
			expected := metav1.Time{
				Time: time.Now().Add(1344 * time.Hour),
			}
			util.Equals(t, expected.Day(), sliceCopy.Status.Expires.Day())
			util.Equals(t, expected.Month(), sliceCopy.Status.Expires.Month())
			util.Equals(t, expected.Year(), sliceCopy.Status.Expires.Year())
		})
		t.Run("consumed quota", func(t *testing.T) {
			TRQCopy, _ := g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), g.TRQObj.GetName(), metav1.GetOptions{})
			CPUPercentage := float64(g.handler.lowResourceQuota.Spec.Hard.Cpu().Value()) / float64(cpu) * 100
			memoryPercentage := float64(g.handler.lowResourceQuota.Spec.Hard.Memory().Value()) / float64(memory) * 100
			util.Equals(t, CPUPercentage, TRQCopy.Status.Used.CPU)
			util.Equals(t, memoryPercentage, TRQCopy.Status.Used.Memory)
		})
	})
}

func TestOperations(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	// Create a slice
	sliceCopy, err := g.edgenetClient.AppsV1alpha().Slices(g.sliceObj.GetNamespace()).Create(context.TODO(), g.sliceObj.DeepCopy(), metav1.CreateOptions{})
	util.OK(t, err)
	g.handler.ObjectCreated(g.sliceObj.DeepCopy())
	childNamespaceStr := fmt.Sprintf("%s-slice-%s", g.sliceObj.GetNamespace(), g.sliceObj.GetName())
	TRQCopy, _ := g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), g.TRQObj.GetName(), metav1.GetOptions{})
	memoryRes := resource.MustParse(TRQCopy.Spec.Claim[0].Memory)
	memory := memoryRes.Value()
	CPURes := resource.MustParse(TRQCopy.Spec.Claim[0].CPU)
	cpu := CPURes.Value()

	var changeProfile = func(profile string, oldQuota *corev1.ResourceQuota, expectedDuration time.Duration, expectedQuota *corev1.ResourceQuota) {
		err := g.client.CoreV1().ResourceQuotas(childNamespaceStr).Delete(context.TODO(), oldQuota.GetName(), metav1.DeleteOptions{})
		util.OK(t, err)
		sliceCopy.Spec.Profile = profile
		g.handler.checkResourcesAvailabilityForSlice(sliceCopy, g.authorityObj.GetName())
		sliceCopy := g.handler.setConstrainsByProfile(childNamespaceStr, sliceCopy)
		t.Run("set expiry date", func(t *testing.T) {
			expected := metav1.Time{
				Time: time.Now().Add(expectedDuration),
			}
			util.Equals(t, expected.Day(), sliceCopy.Status.Expires.Day())
			util.Equals(t, expected.Month(), sliceCopy.Status.Expires.Month())
			util.Equals(t, expected.Year(), sliceCopy.Status.Expires.Year())
		})
		t.Run("consumed quota", func(t *testing.T) {
			TRQCopy, _ := g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), g.TRQObj.GetName(), metav1.GetOptions{})
			CPUPercentage := float64(expectedQuota.Spec.Hard.Cpu().Value()) / float64(cpu) * 100
			memoryPercentage := float64(expectedQuota.Spec.Hard.Memory().Value()) / float64(memory) * 100
			util.Equals(t, CPUPercentage, TRQCopy.Status.Used.CPU)
			util.Equals(t, memoryPercentage, TRQCopy.Status.Used.Memory)
		})
	}

	changeProfile("Low", g.handler.highResourceQuota, (1344 * time.Hour), g.handler.lowResourceQuota)
	changeProfile("Medium", g.handler.lowResourceQuota, (672 * time.Hour), g.handler.medResourceQuota)
	changeProfile("High", g.handler.medResourceQuota, (336 * time.Hour), g.handler.highResourceQuota)
	changeProfile("Medium", g.handler.highResourceQuota, (672 * time.Hour), g.handler.medResourceQuota)
	changeProfile("Low", g.handler.medResourceQuota, (1344 * time.Hour), g.handler.lowResourceQuota)
	changeProfile("High", g.handler.lowResourceQuota, (336 * time.Hour), g.handler.highResourceQuota)
}
