package selectivedeployment

import (
	"context"
	"testing"
	"time"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartController(t *testing.T) {
	g := TestGroup{}
	g.Init()
	// Run the controller in a goroutine
	go Start(g.client, g.edgenetClient)
	// Creating nodes
	nodeParis := g.nodeObj
	nodeParis.SetName("edgenet.planet-lab.eu")
	nodeParis.ObjectMeta.Labels = map[string]string{
		"kubernetes.io/hostname":  "edgenet.planet-lab.eu",
		"edge-net.io/city":        "Paris",
		"edge-net.io/country-iso": "FR",
		"edge-net.io/state-iso":   "IDF",
		"edge-net.io/continent":   "Europe",
		"edge-net.io/lon":         "e2.34",
		"edge-net.io/lat":         "n48.86",
	}
	g.client.CoreV1().Nodes().Create(context.TODO(), nodeParis.DeepCopy(), metav1.CreateOptions{})
	g.client.AppsV1().Deployments("").Create(context.TODO(), g.deploymentObj.DeepCopy(), metav1.CreateOptions{})
	g.client.AppsV1().DaemonSets("").Create(context.TODO(), g.daemonsetObj.DeepCopy(), metav1.CreateOptions{})
	g.client.AppsV1().StatefulSets("").Create(context.TODO(), g.statefulsetObj.DeepCopy(), metav1.CreateOptions{})
	g.client.BatchV1().Jobs("").Create(context.TODO(), g.jobObj.DeepCopy(), metav1.CreateOptions{})
	g.client.BatchV1beta1().CronJobs("").Create(context.TODO(), g.cronjobObj.DeepCopy(), metav1.CreateOptions{})
	// Invoking the create function
	sdObj := g.sdObj.DeepCopy()
	g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), sdObj.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)
	sdCopy, err := g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), sdObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, success, sdCopy.Status.State)
	util.Equals(t, "5/5", sdCopy.Status.Ready)

	g.client.AppsV1().Deployments("").Delete(context.TODO(), g.deploymentObj.GetName(), metav1.DeleteOptions{})
	g.client.AppsV1().DaemonSets("").Delete(context.TODO(), g.daemonsetObj.GetName(), metav1.DeleteOptions{})
	g.client.AppsV1().StatefulSets("").Delete(context.TODO(), g.statefulsetObj.GetName(), metav1.DeleteOptions{})
	g.client.BatchV1().Jobs("").Delete(context.TODO(), g.jobObj.GetName(), metav1.DeleteOptions{})
	g.client.BatchV1beta1().CronJobs("").Delete(context.TODO(), g.cronjobObj.GetName(), metav1.DeleteOptions{})
	time.Sleep(time.Millisecond * 500)
	sdCopy, err = g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), sdObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, success, sdCopy.Status.State)
	util.Equals(t, "5/5", sdCopy.Status.Ready)
	_, err = g.client.AppsV1().Deployments("").Get(context.TODO(), g.deploymentObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	_, err = g.client.AppsV1().DaemonSets("").Get(context.TODO(), g.daemonsetObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	_, err = g.client.AppsV1().StatefulSets("").Get(context.TODO(), g.statefulsetObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	_, err = g.client.BatchV1().Jobs("").Get(context.TODO(), g.jobObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	_, err = g.client.BatchV1beta1().CronJobs("").Get(context.TODO(), g.cronjobObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)

	useu := g.selector
	useu.Value = []string{"US", "FR"}
	useu.Quantity = 2
	useu.Name = "Country"
	countryUSEU := []apps_v1alpha.Selector{useu}
	sdCopy.Spec.Selector = countryUSEU
	g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Update(context.TODO(), sdCopy.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	sdCopy, err = g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), sdCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, failure, sdCopy.Status.State)
	util.Equals(t, "0/5", sdCopy.Status.Ready)

	nodeRichardson := g.nodeObj
	nodeRichardson.SetName("utdallas-1.edge-net.io")
	nodeRichardson.ObjectMeta.Labels = map[string]string{
		"kubernetes.io/hostname":  "utdallas-1.edge-net.io",
		"edge-net.io/city":        "Richardson",
		"edge-net.io/country-iso": "US",
		"edge-net.io/state-iso":   "TX",
		"edge-net.io/continent":   "North America",
		"edge-net.io/lon":         "w-96.78",
		"edge-net.io/lat":         "n32.77",
	}
	g.client.CoreV1().Nodes().Create(context.TODO(), nodeRichardson.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)
	sdCopy, err = g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), sdCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, failure, sdCopy.Status.State)
	util.Equals(t, "0/5", sdCopy.Status.Ready)

	sdCopy.Spec.Recovery = true
	_, err = g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Update(context.TODO(), sdCopy, metav1.UpdateOptions{})
	util.OK(t, err)
	time.Sleep(time.Millisecond * 500)
	sdCopy, err = g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), sdCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, success, sdCopy.Status.State)
	util.Equals(t, "5/5", sdCopy.Status.Ready)

	nodeSeaside := g.nodeObj
	nodeSeaside.SetName("nps-1.edge-net.io")
	nodeSeaside.ObjectMeta.Labels = map[string]string{
		"kubernetes.io/hostname":  "nps-1.edge-net.io",
		"edge-net.io/city":        "Seaside",
		"edge-net.io/country-iso": "US",
		"edge-net.io/state-iso":   "CA",
		"edge-net.io/continent":   "North America",
		"edge-net.io/lon":         "w-121.79",
		"edge-net.io/lat":         "n36.62",
	}
	nodeSeaside.Status.Conditions[0].Type = "NotReady"
	g.client.CoreV1().Nodes().Create(context.TODO(), nodeSeaside.DeepCopy(), metav1.CreateOptions{})

	useu = g.selector
	useu.Value = []string{"US", "FR"}
	useu.Quantity = 3
	useu.Name = "Country"
	countryUSEU = []apps_v1alpha.Selector{useu}
	sdCopy.Spec.Selector = countryUSEU
	g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Update(context.TODO(), sdCopy.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	sdCopy, err = g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), sdCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, failure, sdCopy.Status.State)
	util.Equals(t, "0/5", sdCopy.Status.Ready)

	nodeSeaside.Status.Conditions[0].Type = "Ready"
	g.client.CoreV1().Nodes().Update(context.TODO(), nodeSeaside.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	sdCopy, err = g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), sdCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, success, sdCopy.Status.State)
	util.Equals(t, "5/5", sdCopy.Status.Ready)

	nodeCopy, _ := g.client.CoreV1().Nodes().Get(context.TODO(), nodeSeaside.GetName(), metav1.GetOptions{})
	nodeCopy.Status.Conditions[0].Type = "NotReady"
	g.client.CoreV1().Nodes().Update(context.TODO(), nodeCopy.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	sdCopy, err = g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), sdCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, failure, sdCopy.Status.State)
	util.Equals(t, "0/5", sdCopy.Status.Ready)

	nodeCollegePark := g.nodeObj
	nodeCollegePark.SetName("maxgigapop-1.edge-net.io")
	nodeCollegePark.ObjectMeta.Labels = map[string]string{
		"kubernetes.io/hostname":  "maxgigapop-1.edge-net.io",
		"edge-net.io/city":        "College Park",
		"edge-net.io/country-iso": "US",
		"edge-net.io/state-iso":   "MD",
		"edge-net.io/continent":   "North America",
		"edge-net.io/lon":         "w-76.94",
		"edge-net.io/lat":         "n38.99",
	}
	g.client.CoreV1().Nodes().Create(context.TODO(), nodeCollegePark.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)
	sdCopy, err = g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), sdCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, success, sdCopy.Status.State)
	util.Equals(t, "5/5", sdCopy.Status.Ready)

	g.client.CoreV1().Nodes().Delete(context.TODO(), nodeCollegePark.GetName(), metav1.DeleteOptions{})
	time.Sleep(time.Millisecond * 1500)
	sdCopy, err = g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), sdCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, failure, sdCopy.Status.State)
	util.Equals(t, "0/5", sdCopy.Status.Ready)
}
