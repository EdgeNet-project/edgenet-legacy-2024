package team

import (
	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartController(t *testing.T) {
	g := TeamTestGroup{}
	g.Init()
	// Run the controller in a goroutine
	go Start(g.client, g.edgenetclient)
	// Create a team
	g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.teamObj.DeepCopy())
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	// authority, _ := g.edgenetclient.AppsV1alpha().Authorities().Get(g.authorityObj.GetName(), metav1.GetOptions{})
	// if !authority.Status.Enabled {
	// 	t.Error("Add func of event handler authority doesn't work properly")
	// }
	team, _ := g.edgenetclient.AppsV1alpha().Teams(g.authorityObj.GetNamespace()).Get(g.teamObj.GetName(), metav1.GetOptions{})
	if !team.Status.Enabled {
		t.Error("Add func of event handler authority doesn't work properly")
	}
	// Update a team
	g.teamObj.Spec.Users = []apps_v1alpha.TeamUsers{
		apps_v1alpha.TeamUsers{
			Authority: g.authorityObj.GetName(),
			Username:  "user1",
		},
	}
	// Creating User before updating requesting server to update internal representation of team
	g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.userObj.DeepCopy())
	// Requesting server to update internal representation of team
	g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Update(g.teamObj.DeepCopy())
	team, _ = g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.teamObj.GetName(), metav1.GetOptions{})
	if len(team.Spec.Users) != 1 {
		t.Error("Failed to add user to team")
	}
	// Delete a user
	// Requesting server to delete internal representation of team
	g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Delete(g.teamObj.Name, &metav1.DeleteOptions{})
	team, _ = g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.teamObj.GetName(), metav1.GetOptions{})
	if team != nil {
		t.Error("Failed to delete new test team")
	}

}
