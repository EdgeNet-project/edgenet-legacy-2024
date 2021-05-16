package emailverification

import (
	"context"
	"testing"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/util"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartController(t *testing.T) {
	g := TestGroup{}
	g.Init()
	// Run the controller in a goroutine
	go Start(g.client, g.edgenetClient)
	// Create a emailVerification object
	g.edgenetClient.RegistrationV1alpha().EmailVerifications().Create(context.TODO(), g.emailVerification.DeepCopy(), metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	emailVerification, _ := g.edgenetClient.RegistrationV1alpha().EmailVerifications().Get(context.TODO(), g.emailVerification.GetName(), metav1.GetOptions{})
	util.NotEquals(t, nil, emailVerification.Status.Expiry)
	// Update an emailVerification
	g.emailVerification.Spec.Verified = true
	g.edgenetClient.RegistrationV1alpha().EmailVerifications().Update(context.TODO(), g.emailVerification.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	_, err := g.edgenetClient.RegistrationV1alpha().EmailVerifications().Get(context.TODO(), g.emailVerification.GetName(), metav1.GetOptions{})
	util.Equals(t, false, errors.IsNotFound(err))
	// TODO: Check the status of the relevant object
}
