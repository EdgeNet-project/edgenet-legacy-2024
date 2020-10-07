package totalresourcequota

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
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

// The main structure of test group
type TestGroup struct {
	TRQObj        apps_v1alpha.TotalResourceQuota
	claimObj      apps_v1alpha.TotalResourceDetails
	dropObj       apps_v1alpha.TotalResourceDetails
	authorityObj  apps_v1alpha.Authority
	teamObj       apps_v1alpha.Team
	sliceObj      apps_v1alpha.Slice
	nodeObj       corev1.Node
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

func (g *TestGroup) Init() {
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
		},
		Status: apps_v1alpha.TotalResourceQuotaStatus{
			Exceeded: false,
		},
	}
	claimObj := apps_v1alpha.TotalResourceDetails{
		Name:   "Default",
		CPU:    "12000m",
		Memory: "12Gi",
	}
	dropObj := apps_v1alpha.TotalResourceDetails{
		Name:   "Default",
		CPU:    "10000m",
		Memory: "10Gi",
	}
	authorityObj := apps_v1alpha.Authority{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Authority",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
			UID:  "edgenet",
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
	teamObj := apps_v1alpha.Team{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Team",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "team",
			Namespace: "authority-edgenet",
		},
		Spec: apps_v1alpha.TeamSpec{
			Users:       []apps_v1alpha.TeamUsers{},
			Description: "This is a description",
			Enabled:     true,
		},
		Status: apps_v1alpha.TeamStatus{
			State: success,
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
	nodeObj := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: "apps.edgenet.io/v1alpha",
					Kind:       "Authority",
					Name:       "edgenet",
					UID:        "edgenet"},
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourceMemory:           resource.MustParse("4Gi"),
				corev1.ResourceCPU:              resource.MustParse("2"),
				corev1.ResourceEphemeralStorage: resource.MustParse("51493088"),
				corev1.ResourcePods:             resource.MustParse("100"),
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceMemory:           resource.MustParse("4Gi"),
				corev1.ResourceCPU:              resource.MustParse("2"),
				corev1.ResourceEphemeralStorage: resource.MustParse("51493088"),
				corev1.ResourcePods:             resource.MustParse("100"),
			},
			Conditions: []corev1.NodeCondition{
				corev1.NodeCondition{
					Type:   "Ready",
					Status: "True",
				},
			},
		},
	}
	g.TRQObj = TRQObj
	g.claimObj = claimObj
	g.dropObj = dropObj
	g.authorityObj = authorityObj
	g.teamObj = teamObj
	g.sliceObj = sliceObj
	g.nodeObj = nodeObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetClient = edgenettestclient.NewSimpleClientset()
	// Imitate authority creation processes
	g.edgenetClient.AppsV1alpha().Authorities().Create(context.TODO(), g.authorityObj.DeepCopy(), metav1.CreateOptions{})
	namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("authority-%s", g.authorityObj.GetName())}}
	namespaceLabels := map[string]string{"owner": "authority", "owner-name": g.authorityObj.GetName(), "authority-name": g.authorityObj.GetName()}
	namespace.SetLabels(namespaceLabels)
	g.client.CoreV1().Namespaces().Create(context.TODO(), &namespace, metav1.CreateOptions{})
}

func TestHandlerInit(t *testing.T) {
	// Sync the test group
	g := TestGroup{}
	g.Init()
	// Initialize the handler
	g.handler.Init(g.client, g.edgenetClient)
	util.Equals(t, g.client, g.handler.clientset)
	util.Equals(t, g.edgenetClient, g.handler.edgenetClientset)
}

func TestCreate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)

	cases := map[string]struct {
		input    []time.Duration
		sleep    time.Duration
		expected int
	}{
		"without expiry date": {nil, 110, 2},
		"expiries soon":       {[]time.Duration{100}, 110, 0},
		"expired":             {[]time.Duration{-100}, 0, 0},
		"mix/1":               {[]time.Duration{100, 1000, -100}, 0, 4},
		"mix/2":               {[]time.Duration{100, 1000, -100}, 110, 2},
		"mix/3":               {[]time.Duration{100, 50, 1000, 1400, -10, -100}, 0, 8},
		"mix/4":               {[]time.Duration{100, 50, 1000, 1400, -10, -100}, 110, 4},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			TRQ := g.TRQObj
			claim := g.claimObj
			drop := g.dropObj
			if tc.input != nil {
				for _, input := range tc.input {
					claim.Expires = &metav1.Time{
						Time: time.Now().Add(input * time.Millisecond),
					}
					TRQ.Spec.Claim = append(TRQ.Spec.Claim, claim)
					drop.Expires = &metav1.Time{
						Time: time.Now().Add(input * time.Millisecond),
					}
					TRQ.Spec.Drop = append(TRQ.Spec.Drop, drop)
				}
			} else {
				TRQ.Spec.Claim = append(TRQ.Spec.Claim, claim)
				TRQ.Spec.Drop = append(TRQ.Spec.Drop, drop)
			}
			g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Create(context.TODO(), TRQ.DeepCopy(), metav1.CreateOptions{})
			g.handler.ObjectCreated(TRQ.DeepCopy())
			defer g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Delete(context.TODO(), TRQ.GetName(), metav1.DeleteOptions{})
			time.Sleep(tc.sleep * time.Millisecond)
			TRQCopy, err := g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), TRQ.GetName(), metav1.GetOptions{})
			util.OK(t, err)
			util.Equals(t, true, TRQCopy.Spec.Enabled)
			util.Equals(t, tc.expected, (len(TRQCopy.Spec.Claim) + len(TRQCopy.Spec.Drop)))
		})
	}

	t.Run("exceeded", func(t *testing.T) {
		g.edgenetClient.AppsV1alpha().Slices(g.sliceObj.GetNamespace()).Create(context.TODO(), g.sliceObj.DeepCopy(), metav1.CreateOptions{})
		childNamespace := fmt.Sprintf("%s-slice-%s", g.sliceObj.GetNamespace(), g.sliceObj.GetName())
		namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: childNamespace}}
		namespaceLabels := map[string]string{"owner": "slice", "owner-name": g.sliceObj.GetName(), "authority-name": g.authorityObj.GetName()}
		namespace.SetLabels(namespaceLabels)
		g.client.CoreV1().Namespaces().Create(context.TODO(), &namespace, metav1.CreateOptions{})
		quota := corev1.ResourceQuota{
			ObjectMeta: metav1.ObjectMeta{
				Name: "slice-high-quota",
			},
			Spec: corev1.ResourceQuotaSpec{
				Hard: map[corev1.ResourceName]resource.Quantity{
					"cpu":              resource.MustParse("8000m"),
					"memory":           resource.MustParse("8192Mi"),
					"requests.storage": resource.MustParse("8Gi"),
				},
			},
		}
		g.client.CoreV1().ResourceQuotas(childNamespace).Create(context.TODO(), quota.DeepCopy(), metav1.CreateOptions{})

		TRQ := g.TRQObj
		TRQ.Status.Exceeded = true
		g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Create(context.TODO(), TRQ.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(TRQ.DeepCopy())
		defer g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Delete(context.TODO(), TRQ.GetName(), metav1.DeleteOptions{})
		TRQCopy, err := g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), TRQ.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		util.Equals(t, true, TRQCopy.Spec.Enabled)

		_, err = g.edgenetClient.AppsV1alpha().Slices(g.sliceObj.GetNamespace()).Get(context.TODO(), g.sliceObj.GetName(), metav1.GetOptions{})
		util.Equals(t, true, errors.IsNotFound(err))

		TRQCopy, err = g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), TRQ.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		util.Equals(t, false, TRQCopy.Status.Exceeded)
	})
}

func TestUpdate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	cases := map[string]struct {
		input    []time.Duration
		sleep    time.Duration
		expected int
	}{
		"without expiry date": {nil, 110, 2},
		"expiries soon":       {[]time.Duration{100}, 110, 0},
		"expired":             {[]time.Duration{-100}, 0, 0},
		"mix/1":               {[]time.Duration{100, 1000, -100}, 0, 4},
		"mix/2":               {[]time.Duration{100, 1000, -100}, 110, 2},
		"mix/3":               {[]time.Duration{100, 50, 1000, 1400, -10, -100}, 0, 8},
		"mix/4":               {[]time.Duration{100, 50, 1000, 1400, -10, -100}, 110, 4},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			TRQ := g.TRQObj
			g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Create(context.TODO(), TRQ.DeepCopy(), metav1.CreateOptions{})
			g.handler.ObjectCreated(TRQ.DeepCopy())
			defer g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Delete(context.TODO(), TRQ.GetName(), metav1.DeleteOptions{})
			TRQCopy, err := g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), TRQ.GetName(), metav1.GetOptions{})
			util.OK(t, err)
			util.Equals(t, true, TRQCopy.Spec.Enabled)
			util.Equals(t, 0, (len(TRQCopy.Spec.Claim) + len(TRQCopy.Spec.Drop)))

			var field fields
			field.spec = true
			claim := g.claimObj
			drop := g.dropObj
			if tc.input != nil {
				for _, input := range tc.input {
					claim.Expires = &metav1.Time{
						Time: time.Now().Add(input * time.Millisecond),
					}
					TRQCopy.Spec.Claim = append(TRQCopy.Spec.Claim, claim)
					drop.Expires = &metav1.Time{
						Time: time.Now().Add(input * time.Millisecond),
					}
					TRQCopy.Spec.Drop = append(TRQCopy.Spec.Drop, drop)
				}
				field.expiry = true
			} else {
				TRQCopy.Spec.Claim = append(TRQCopy.Spec.Claim, claim)
				TRQCopy.Spec.Drop = append(TRQCopy.Spec.Drop, drop)
				field.expiry = false
			}

			TRQCopy, err = g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Update(context.TODO(), TRQCopy.DeepCopy(), metav1.UpdateOptions{})
			util.OK(t, err)
			g.handler.ObjectUpdated(TRQCopy.DeepCopy(), field)
			time.Sleep(tc.sleep * time.Millisecond)
			TRQCopy, err = g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), TRQCopy.GetName(), metav1.GetOptions{})

			util.OK(t, err)
			util.Equals(t, true, TRQCopy.Spec.Enabled)
			util.Equals(t, tc.expected, (len(TRQCopy.Spec.Claim) + len(TRQCopy.Spec.Drop)))
		})
	}
	t.Run("total quota", func(t *testing.T) {
		g.edgenetClient.AppsV1alpha().Teams(g.teamObj.GetNamespace()).Create(context.TODO(), g.teamObj.DeepCopy(), metav1.CreateOptions{})
		teamChildNamespace := fmt.Sprintf("%s-team-%s", g.teamObj.GetNamespace(), g.teamObj.GetName())
		namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: teamChildNamespace}}
		namespaceLabels := map[string]string{"owner": "team", "owner-name": g.teamObj.GetName(), "authority-name": g.authorityObj.GetName()}
		namespace.SetLabels(namespaceLabels)
		g.client.CoreV1().Namespaces().Create(context.TODO(), &namespace, metav1.CreateOptions{})
		defer g.client.CoreV1().Namespaces().Delete(context.TODO(), namespace.GetName(), metav1.DeleteOptions{})

		cases := map[string]struct {
			input    []apps_v1alpha.TotalResourceDetails
			expiry   []time.Duration
			kind     []string
			quota    string
			expected bool
		}{
			"claim/high":                                  {[]apps_v1alpha.TotalResourceDetails{g.claimObj}, nil, []string{"Claim"}, "High", false},
			"claim expires soon/high":                     {[]apps_v1alpha.TotalResourceDetails{g.claimObj}, []time.Duration{50}, []string{"Claim"}, "High", true},
			"claim-drop/low":                              {[]apps_v1alpha.TotalResourceDetails{g.claimObj, g.dropObj}, nil, []string{"Claim", "Drop"}, "Low", false},
			"claim-drop/high":                             {[]apps_v1alpha.TotalResourceDetails{g.claimObj, g.dropObj}, nil, []string{"Claim", "Drop"}, "High", true},
			"claim-drop expires soon/high":                {[]apps_v1alpha.TotalResourceDetails{g.claimObj, g.dropObj}, []time.Duration{1000, 80}, []string{"Claim", "Drop"}, "High", true},
			"claim-claim and then drop expires soon/high": {[]apps_v1alpha.TotalResourceDetails{g.claimObj, g.claimObj, g.dropObj}, []time.Duration{1000, 50, 90}, []string{"Claim", "Claim", "Drop"}, "High", false},
			"drop-claim and then drop expires soon/high":  {[]apps_v1alpha.TotalResourceDetails{g.dropObj, g.claimObj, g.dropObj}, []time.Duration{1000, 50, 90}, []string{"Drop", "Claim", "Drop"}, "High", true},
		}
		for k, tc := range cases {
			t.Run(k, func(t *testing.T) {
				TRQ := g.TRQObj
				TRQCopy, err := g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Create(context.TODO(), TRQ.DeepCopy(), metav1.CreateOptions{})
				util.OK(t, err)
				g.handler.ObjectCreated(TRQCopy.DeepCopy())
				defer g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Delete(context.TODO(), TRQCopy.GetName(), metav1.DeleteOptions{})

				slice := g.sliceObj
				slice.SetNamespace(teamChildNamespace)
				g.edgenetClient.AppsV1alpha().Slices(teamChildNamespace).Create(context.TODO(), slice.DeepCopy(), metav1.CreateOptions{})
				defer g.edgenetClient.AppsV1alpha().Slices(teamChildNamespace).Delete(context.TODO(), slice.GetName(), metav1.DeleteOptions{})
				childNamespace := fmt.Sprintf("%s-slice-%s", teamChildNamespace, slice.GetName())
				namespace = corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: childNamespace}}
				namespaceLabels = map[string]string{"owner": "slice", "owner-name": slice.GetName(), "authority-name": g.authorityObj.GetName()}
				namespace.SetLabels(namespaceLabels)
				g.client.CoreV1().Namespaces().Create(context.TODO(), &namespace, metav1.CreateOptions{})
				defer g.client.CoreV1().Namespaces().Delete(context.TODO(), namespace.GetName(), metav1.DeleteOptions{})

				var quota corev1.ResourceQuota
				if tc.quota == "High" {
					quota = corev1.ResourceQuota{
						ObjectMeta: metav1.ObjectMeta{
							Name: "slice-high-quota",
						},
						Spec: corev1.ResourceQuotaSpec{
							Hard: map[corev1.ResourceName]resource.Quantity{
								"cpu":              resource.MustParse("8000m"),
								"memory":           resource.MustParse("8192Mi"),
								"requests.storage": resource.MustParse("8Gi"),
							},
						},
					}
				} else if tc.quota == "Low" {
					quota = corev1.ResourceQuota{
						ObjectMeta: metav1.ObjectMeta{
							Name: "slice-low-quota",
						},
						Spec: corev1.ResourceQuotaSpec{
							Hard: map[corev1.ResourceName]resource.Quantity{
								"cpu":              resource.MustParse("2000m"),
								"memory":           resource.MustParse("2048Mi"),
								"requests.storage": resource.MustParse("500Mi"),
							},
						},
					}
				}
				g.client.CoreV1().ResourceQuotas(childNamespace).Create(context.TODO(), quota.DeepCopy(), metav1.CreateOptions{})
				defer g.client.CoreV1().ResourceQuotas(childNamespace).Delete(context.TODO(), quota.GetName(), metav1.DeleteOptions{})

				var field fields
				field.spec = true
				for i, input := range tc.input {
					if tc.kind[i] == "Claim" {
						claim := input
						if tc.expiry != nil {
							claim.Expires = &metav1.Time{
								Time: time.Now().Add(tc.expiry[i] * time.Millisecond),
							}
							field.expiry = true
						} else {
							field.expiry = false
						}

						TRQCopy.Spec.Claim = append(TRQCopy.Spec.Claim, claim)
					} else if tc.kind[i] == "Drop" {
						drop := input
						if tc.expiry != nil {
							drop.Expires = &metav1.Time{
								Time: time.Now().Add(tc.expiry[i] * time.Millisecond),
							}
							field.expiry = true
						} else {
							field.expiry = false
						}
						TRQCopy.Spec.Drop = append(TRQCopy.Spec.Drop, drop)
					}
				}
				TRQCopy, err = g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Update(context.TODO(), TRQCopy.DeepCopy(), metav1.UpdateOptions{})
				util.OK(t, err)
				g.handler.ObjectUpdated(TRQCopy.DeepCopy(), field)
				time.Sleep(100 * time.Millisecond)

				_, err = g.edgenetClient.AppsV1alpha().Slices(teamChildNamespace).Get(context.TODO(), g.sliceObj.GetName(), metav1.GetOptions{})
				util.Equals(t, tc.expected, errors.IsNotFound(err))
			})
		}
	})
}

func TestCreateTotalResourceQuota(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)

	_, err := g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), g.TRQObj.GetName(), metav1.GetOptions{})
	util.Equals(t, true, errors.IsNotFound(err))
	g.handler.Create(g.TRQObj.GetName())
	_, err = g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), g.TRQObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
}
