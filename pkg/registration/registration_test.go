package registration

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	"github.com/EdgeNet-project/edgenet/pkg/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

type TestGroup struct {
	authorityObj  apps_v1alpha.Authority
	userObj       apps_v1alpha.User
	client        kubernetes.Interface
	edgenetclient versioned.Interface
}

func TestMain(m *testing.M) {
	//log.SetOutput(ioutil.Discard)
	//logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

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
				Phone:     "+333333333",
				Username:  "johndoe",
			},
			Enabled: true,
		},
	}
	userObj := apps_v1alpha.User{
		TypeMeta: metav1.TypeMeta{
			Kind:       "User",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "johndoe",
			Namespace: "authority-edgenet",
		},
		Spec: apps_v1alpha.UserSpec{
			FirstName: "EdgeNet",
			LastName:  "EdgeNet",
			Email:     "john.doe@edge-net.org",
			Active:    true,
		},
		Status: apps_v1alpha.UserStatus{
			Type: "admin",
		},
	}
	g.authorityObj = authorityObj
	g.userObj = userObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
	Clientset = g.client
}

func TestKubeconfigWithUser(t *testing.T) {
	g := TestGroup{}
	g.Init()

	t.Run("create user with client certificates", func(t *testing.T) {
		// Mock the signer
		go func() {
			timeout := time.After(10 * time.Second)
			ticker := time.Tick(1 * time.Second)
		check:
			for {
				select {
				case <-timeout:
					break check
				case <-ticker:
					CSRObj, getErr := Clientset.CertificatesV1().CertificateSigningRequests().Get(context.TODO(), fmt.Sprintf("%s-%s", g.authorityObj.GetName(), g.userObj.GetName()), metav1.GetOptions{})
					if getErr == nil {
						CSRObj.Status.Certificate = CSRObj.Spec.Request
						_, updateErr := Clientset.CertificatesV1().CertificateSigningRequests().UpdateStatus(context.TODO(), CSRObj, metav1.UpdateOptions{})
						if updateErr == nil {
							break check
						}
					}
				}
			}
		}()

		cert, key, err := MakeUser(g.authorityObj.GetName(), g.userObj.GetName(), g.userObj.Spec.Email)
		util.OK(t, err)

		t.Run("generate config", func(t *testing.T) {
			err = MakeConfig(g.authorityObj.GetName(), g.userObj.GetName(), g.userObj.Spec.Email, cert, key)
			util.OK(t, err)
		})
	})
}

func TestKubeconfigWithServiceAccount(t *testing.T) {
	g := TestGroup{}
	g.Init()
	t.Run("create service account", func(t *testing.T) {
		serviceAccount, err := CreateServiceAccount(g.userObj.DeepCopy(), "User", []metav1.OwnerReference{})

		util.OK(t, err)
		t.Run("generate config without secret", func(t *testing.T) {
			output := CreateConfig(serviceAccount)
			util.Equals(t, fmt.Sprintf("Serviceaccount %s doesn't have a token", g.userObj.GetName()), output)
		})
	})

	t.Run("generate config with service account containing token", func(t *testing.T) {
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-token-1234",
				Namespace: g.userObj.GetNamespace(),
			},
		}
		secret.Data = make(map[string][]byte)
		secret.Data["token"] = []byte("test1234token")
		serviceAccount := corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      g.userObj.GetName(),
				Namespace: g.userObj.GetNamespace(),
			},
			Secrets: []corev1.ObjectReference{
				corev1.ObjectReference{
					Name:      "test-token-1234",
					Namespace: g.userObj.GetNamespace(),
				},
			},
		}
		_, err := g.client.CoreV1().Secrets(secret.Namespace).Create(context.TODO(), &secret, metav1.CreateOptions{})
		util.OK(t, err)
		output := CreateConfig(&serviceAccount)

		list := []string{
			"certificate-authority-data",
			"clusters",
			"cluster",
			"server",
			"contexts",
			"context",
			"current-context",
			"namespace",
			secret.Namespace,
			"user",
			g.userObj.GetName(),
			string(secret.Data["token"]),
			"kind",
			"Config",
			"apiVersion",
		}
		for _, expected := range list {
			if !strings.Contains(output, expected) {
				t.Errorf("Config malformed. Expected \"%s\" in the config not found", expected)
			}
		}
	})
}
