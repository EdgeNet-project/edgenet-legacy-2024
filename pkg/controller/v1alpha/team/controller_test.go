package team

import (
	"context"
	"fmt"
	"testing"
	"time"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/util"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartController(t *testing.T) {
	g := TestGroup{}
	g.Init()
	// Run the controller in a goroutine
	go Start(g.client, g.edgenetClient)
	// Create a team
	g.edgenetClient.AppsV1alpha().Teams(g.teamObj.GetNamespace()).Create(context.TODO(), g.teamObj.DeepCopy(), metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	team, err := g.edgenetClient.AppsV1alpha().Teams(g.teamObj.GetNamespace()).Get(context.TODO(), g.teamObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	g.edgenetClient.AppsV1alpha().Users(g.userObj.GetNamespace()).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
	// Update a team
	team.Spec.Users = []apps_v1alpha.TeamUsers{
		{
			Authority: g.authorityObj.GetName(),
			Username:  g.userObj.GetName(),
		},
	}
	g.edgenetClient.AppsV1alpha().Teams(team.GetNamespace()).Update(context.TODO(), team, metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	childNamespaceStr := fmt.Sprintf("%s-team-%s", g.teamObj.GetNamespace(), g.teamObj.GetName())
	_, err = g.client.RbacV1().RoleBindings(childNamespaceStr).Get(context.TODO(), fmt.Sprintf("%s-%s-team-%s", g.userObj.GetNamespace(), g.userObj.GetName(), "user"), metav1.GetOptions{})
	util.OK(t, err)
	g.edgenetClient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Delete(context.TODO(), g.teamObj.Name, metav1.DeleteOptions{})
	time.Sleep(time.Millisecond * 500)
	_, err = g.client.CoreV1().Namespaces().Get(context.TODO(), fmt.Sprintf("%s-team-%s", g.teamObj.GetNamespace(), g.teamObj.GetName()), metav1.GetOptions{})
	util.Equals(t, true, errors.IsNotFound(err))
}
