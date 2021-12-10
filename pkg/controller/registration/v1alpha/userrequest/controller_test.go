package userrequest

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/signals"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog"
)

type TestGroup struct {
	tenantObj      corev1alpha.Tenant
	userRequestObj registrationv1alpha.UserRequest
}

var controller *Controller
var kubeclientset kubernetes.Interface = testclient.NewSimpleClientset()
var edgenetclientset versioned.Interface = edgenettestclient.NewSimpleClientset()

func TestMain(m *testing.M) {
	klog.SetOutput(ioutil.Discard)
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)

	flag.String("dir", "../../../../..", "Override the directory.")
	flag.String("smtp-path", "../../../../../configs/smtp_test.yaml", "Set SMTP path.")
	flag.Parse()

	stopCh := signals.SetupSignalHandler()

	go func() {

		edgenetInformerFactory := informers.NewSharedInformerFactory(edgenetclientset, time.Second*30)

		newController := NewController(kubeclientset,
			edgenetclientset,
			edgenetInformerFactory.Registration().V1alpha().UserRequests())

		edgenetInformerFactory.Start(stopCh)
		controller = newController
		if err := controller.Run(2, stopCh); err != nil {
			klog.Fatalf("Error running controller: %s", err.Error())
		}
	}()

	os.Exit(m.Run())
	<-stopCh
}

// Init syncs the test group
func (g *TestGroup) Init() {
	tenantObj := corev1alpha.Tenant{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Tenant",
			APIVersion: "apps.edgenet.io/v1alpha",
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
				Email:     "joe.public@edge-net.org",
				FirstName: "Joe",
				LastName:  "Public",
				Phone:     "+33NUMBER",
				Username:  "joepublic",
			},
			Enabled: true,
		},
	}
	userRequestObj := registrationv1alpha.UserRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "UserRequest",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "johnsmith",
		},
		Spec: registrationv1alpha.UserRequestSpec{
			Tenant:    "edgenet",
			FirstName: "John",
			LastName:  "Smith",
			Email:     "john.smith@edge-net.org",
		},
	}
	g.tenantObj = tenantObj
	g.userRequestObj = userRequestObj
	edgenetclientset.CoreV1alpha().Tenants().Create(context.TODO(), g.tenantObj.DeepCopy(), metav1.CreateOptions{})
}

func TestStartController(t *testing.T) {
	g := TestGroup{}
	g.Init()
	userRequestTest := g.userRequestObj.DeepCopy()
	userRequestTest.SetName("user-request-controller-test")

	// Create a user registration object
	edgenetclientset.RegistrationV1alpha().UserRequests().Create(context.TODO(), userRequestTest, metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 1000)
	// Get the object and check the status
	userRequest, err := edgenetclientset.RegistrationV1alpha().UserRequests().Get(context.TODO(), userRequestTest.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	log.Println(userRequest)
	expected := metav1.Time{
		Time: time.Now().Add(72 * time.Hour),
	}
	util.Equals(t, expected.Day(), userRequest.Status.Expiry.Day())
	util.Equals(t, expected.Month(), userRequest.Status.Expiry.Month())
	util.Equals(t, expected.Year(), userRequest.Status.Expiry.Year())
	util.EqualsMultipleExp(t, []string{statusDict["email-ok"], statusDict["email-fail"]}, userRequest.Status.Message[0])
	// Update a user request
	userRequest.Spec.Email = "different-email@edge-net.org"
	userRequest, _ = edgenetclientset.RegistrationV1alpha().UserRequests().Update(context.TODO(), userRequest, metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	// Update a user request
	userRequest.Spec.Approved = true
	edgenetclientset.RegistrationV1alpha().UserRequests().Update(context.TODO(), userRequest, metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	// Checking if user registration transitioned to user after update
	//_, err := edgenetclientset.AppsV1alpha().Users().Get(context.TODO(), userRequest.GetName(), metav1.GetOptions{})
	//util.OK(t, err)
}

func TestTimeout(t *testing.T) {
	g := TestGroup{}
	g.Init()
	userRequestTest := g.userRequestObj.DeepCopy()
	userRequestTest.SetName("user-request-timeout-test")
	edgenetclientset.RegistrationV1alpha().UserRequests().Create(context.TODO(), userRequestTest, metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)

	t.Run("set expiry date", func(t *testing.T) {
		userRequest, _ := edgenetclientset.RegistrationV1alpha().UserRequests().Get(context.TODO(), userRequestTest.GetName(), metav1.GetOptions{})
		expected := metav1.Time{
			Time: time.Now().Add(72 * time.Hour),
		}
		util.Equals(t, expected.Day(), userRequest.Status.Expiry.Day())
		util.Equals(t, expected.Month(), userRequest.Status.Expiry.Month())
		util.Equals(t, expected.Year(), userRequest.Status.Expiry.Year())
	})
	t.Run("timeout", func(t *testing.T) {
		userRequest, _ := edgenetclientset.RegistrationV1alpha().UserRequests().Get(context.TODO(), userRequestTest.GetName(), metav1.GetOptions{})
		userRequest.Status.Expiry = &metav1.Time{
			Time: time.Now().Add(10 * time.Millisecond),
		}
		edgenetclientset.RegistrationV1alpha().UserRequests().UpdateStatus(context.TODO(), userRequest, metav1.UpdateOptions{})
		time.Sleep(100 * time.Millisecond)
		_, err := edgenetclientset.RegistrationV1alpha().UserRequests().Get(context.TODO(), userRequest.GetName(), metav1.GetOptions{})
		util.Equals(t, true, errors.IsNotFound(err))
	})
}
