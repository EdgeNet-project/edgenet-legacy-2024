package acceptableusepolicy

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartController(t *testing.T) {
	g := AUPTestGroup{}
	g.Init()
	// Run the controller in a goroutine
	go Start(g.client, g.edgenetclient)
	// Create an AUP
	g.edgenetclient.AppsV1alpha().AcceptableUsePolicies(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.AUPObj.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)
	AUP, _ := g.edgenetclient.AppsV1alpha().AcceptableUsePolicies(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.AUPObj.GetName(), metav1.GetOptions{})
	// Check state
	if AUP.Status.State != success && AUP.Status.Expires != nil {
		t.Errorf(errorDict["add-func"])
	}

	// Update an AUP
	g.AUPObj.Spec.Accepted, g.AUPObj.Spec.Renew = true, true
	// Requesting server to Update internal representation of AUP
	g.edgenetclient.AppsV1alpha().AcceptableUsePolicies(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Update(context.TODO(), g.AUPObj.DeepCopy(), metav1.UpdateOptions{})
	AUP, _ = g.edgenetclient.AppsV1alpha().AcceptableUsePolicies(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.AUPObj.GetName(), metav1.GetOptions{})
	// Check state
	if AUP.Status.State != success && AUP.Status.Expires != nil && strings.Contains(AUP.Status.Message[0], "Agreed and Renewed") {
		t.Errorf(errorDict["add-func"])
	}
}
