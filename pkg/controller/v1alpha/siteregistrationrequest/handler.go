package siteregistrationrequest

import (
	"fmt"
	"time"

	apps_v1alpha "headnode/pkg/apis/apps/v1alpha"
	"headnode/pkg/authorization"
	"headnode/pkg/client/clientset/versioned"

	log "github.com/Sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// HandlerInterface interface contains the methods that are required
type HandlerInterface interface {
	Init() error
	ObjectCreated(obj interface{})
	ObjectUpdated(obj interface{})
	ObjectDeleted(obj interface{})
}

// Handler implementation
type Handler struct {
	clientset        *kubernetes.Clientset
	edgenetClientset *versioned.Clientset
}

// Init handles any handler initialization
func (t *Handler) Init() error {
	log.Info("SRRHandler.Init")
	var err error
	t.clientset, err = authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	t.edgenetClientset, err = authorization.CreateEdgeNetClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	return err
}

// ObjectCreated is called when an object is created
func (t *Handler) ObjectCreated(obj interface{}) {
	log.Info("SRRHandler.ObjectCreated")
	// Create a copy of the site object to make changes on it
	SRRCopy := obj.(*apps_v1alpha.SiteRegistrationRequest).DeepCopy()
	defer t.edgenetClientset.AppsV1alpha().SiteRegistrationRequests().UpdateStatus(SRRCopy)
	SRRCopy.Status.Approved = false

	if SRRCopy.Status.Expires == nil {
		go t.setApprovalTimeout(SRRCopy)
		SRRCopy.Status.Expires = &metav1.Time{
			Time: time.Now().Add(72 * time.Hour),
		}
	} else {
		go t.setApprovalTimeout(SRRCopy)
	}
	// Send en email to inform admins of cluster, TBD
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj interface{}) {
	log.Info("SRRHandler.ObjectUpdated")
	// Create a copy of the site object to make changes on it
	SRRCopy := obj.(*apps_v1alpha.SiteRegistrationRequest).DeepCopy()

	// Check whether the request for site registration approved
	if SRRCopy.Status.Approved {
		site := apps_v1alpha.Site{}
		site.SetName(SRRCopy.GetName())
		site.Spec.Address = SRRCopy.Spec.Address
		site.Spec.Contact = SRRCopy.Spec.Contact
		site.Spec.FullName = SRRCopy.Spec.FullName
		site.Spec.ShortName = SRRCopy.Spec.ShortName
		site.Spec.URL = SRRCopy.Spec.URL

		t.edgenetClientset.AppsV1alpha().Sites().Create(site.DeepCopy())
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("SRRHandler.ObjectDeleted")
	// Mail notification, TBD
}

// setApprovalTimeout puts a procedure in place to remove requests by approval or timeout
func (t *Handler) setApprovalTimeout(SRRCopy *apps_v1alpha.SiteRegistrationRequest) {
	registrationApproved := make(chan bool, 1)
	timeoutRenewed := make(chan bool, 1)
	terminated := make(chan bool, 1)
	var timeout <-chan time.Time
	closeChannels := func() {
		close(registrationApproved)
		close(timeoutRenewed)
		close(terminated)
	}

	// Watch the events of site registration request object
	watchSRR, err := t.edgenetClientset.AppsV1alpha().SiteRegistrationRequests().Watch(metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", SRRCopy.GetName())})
	if err == nil {
		go func() {
			// Get events from watch interface
			for SRREvent := range watchSRR.ResultChan() {
				// Get updated site registration request object
				updatedSRR, status := SRREvent.Object.(*apps_v1alpha.SiteRegistrationRequest)
				if status {
					if SRREvent.Type == "DELETED" {
						terminated <- true
						continue
					}

					if updatedSRR.Status.Approved == true {
						registrationApproved <- true
						break
					} else if updatedSRR.Status.Expires != nil {
						timeout = time.After(time.Until(updatedSRR.Status.Expires.Time))
						// Check whether expire date updated
						if SRRCopy.Status.Expires != nil {
							if SRRCopy.Status.Expires.Time != updatedSRR.Status.Expires.Time {
								timeoutRenewed <- true
							}
						} else {
							timeoutRenewed <- true
						}
					}
				}
			}
		}()
	} else {
		// In case of any malfunction of watching siteregistrationrequest resources,
		// there is a timeout at 72 hours
		timeout = time.After(72 * time.Hour)
	}

	// Infinite loop
timeoutLoop:
	for {
		// Wait on multiple channel operations
	timeoutOptions:
		select {
		case <-registrationApproved:
			watchSRR.Stop()
			closeChannels()
			t.edgenetClientset.AppsV1alpha().SiteRegistrationRequests().Delete(SRRCopy.GetName(), &metav1.DeleteOptions{})
			break timeoutLoop
		case <-timeoutRenewed:
			break timeoutOptions
		case <-timeout:
			watchSRR.Stop()
			closeChannels()
			t.edgenetClientset.AppsV1alpha().SiteRegistrationRequests().Delete(SRRCopy.GetName(), &metav1.DeleteOptions{})
			break timeoutLoop
		case <-terminated:
			watchSRR.Stop()
			closeChannels()
			break timeoutLoop
		}
	}
}
