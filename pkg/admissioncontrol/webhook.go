package admissioncontrol

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"

	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/klog/v2"
)

type Webhook struct {
	CertFile string
	KeyFile  string
	Codecs   serializer.CodecFactory
}

func (wh *Webhook) RunServer() {
	cert, err := tls.LoadX509KeyPair(wh.CertFile, wh.KeyFile)
	if err != nil {
		klog.Fatalln(err.Error())
		os.Exit(1)
	}

	http.HandleFunc("/mutate/pod", wh.mutatePod)
	http.HandleFunc("/validate/pod", wh.validatePod)
	http.HandleFunc("/validate/tenant-request", wh.validateTenantRequest)
	http.HandleFunc("/validate/cluster-role-request", wh.validateClusterRoleRequest)
	//http.HandleFunc("/validate/role-request", wh.validateRoleRequest)
	//http.HandleFunc("/validate/subnamespace", wh.validateSubNamespace)
	//http.HandleFunc("/validate/slice", wh.validateSlice)

	server := http.Server{
		Addr: ":443",
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}

	if err := server.ListenAndServeTLS("", ""); err != nil {
		klog.Fatalln(err.Error())
		os.Exit(2)
	}
}

func (wh *Webhook) mutatePod(w http.ResponseWriter, r *http.Request) {
	klog.Infoln("Pod: message on mutate received")
	deserializer := wh.Codecs.UniversalDeserializer()
	admissionReviewRequest, err := admissionReviewFromRequest(r, deserializer)
	if err != nil {
		klog.Errorf("Pod admission review error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if admissionReviewRequest.Request.Resource != podResource {
		err := fmt.Errorf("pod wrong resource kind: %v", admissionReviewRequest.Request.Resource.Resource)
		klog.Error(err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	rawRequest := admissionReviewRequest.Request.Object.Raw
	pod := new(corev1.Pod)
	if _, _, err := deserializer.Decode(rawRequest, nil, pod); err != nil {
		klog.Errorf("pod decode error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	var ingressBandwidth resource.Quantity
	var egressBandwidth resource.Quantity
	for _, container := range pod.Spec.Containers {
		if quantity, ok := container.Resources.Requests["edge-net.io/ingress-bandwidth"]; !ok {
			ingressBandwidth.Add(quantity)
		}
		if quantity, ok := container.Resources.Requests["edge-net.io/egress-bandwidth"]; !ok {
			egressBandwidth.Add(quantity)
		}
	}
	if ingressBandwidth.IsZero() && egressBandwidth.IsZero() {
		klog.Infoln("Pod: no bandwidth requested")
		return
	}

	var patchElement []string
	if !ingressBandwidth.IsZero() {
		if actualIngressBandwidth, ok := pod.Annotations["kubernetes.io/ingress-bandwidth"]; !ok {
			patchElement = append(patchElement, fmt.Sprintf(`{"op":"add","path":"/metadata/annotations","value":{"kubernetes.io/ingress-bandwidth":%s}}`, ingressBandwidth.String()))
		} else {
			if actualQuantity, err := resource.ParseQuantity(actualIngressBandwidth); err != nil {
				patchElement = append(patchElement, fmt.Sprintf(`{"op":"replace","path":"/metadata/annotations","value":{"kubernetes.io/ingress-bandwidth":%s}}`, ingressBandwidth.String()))
			} else {
				if !ingressBandwidth.Equal(actualQuantity) {
					patchElement = append(patchElement, fmt.Sprintf(`{"op":"replace","path":"/metadata/annotations","value":{"kubernetes.io/ingress-bandwidth":%s}}`, ingressBandwidth.String()))
				}
			}
		}
	}
	if !egressBandwidth.IsZero() {
		if actualEgressBandwidth, ok := pod.Annotations["kubernetes.io/egress-bandwidth"]; !ok {
			patchElement = append(patchElement, fmt.Sprintf(`{"op":"add","path":"/metadata/annotations","value":{"kubernetes.io/egress-bandwidth":%s}}`, egressBandwidth.String()))
		} else {
			if actualQuantity, err := resource.ParseQuantity(actualEgressBandwidth); err != nil {
				patchElement = append(patchElement, fmt.Sprintf(`{"op":"replace","path":"/metadata/annotations","value":{"kubernetes.io/egress-bandwidth":%s}}`, egressBandwidth.String()))
			} else {
				if !egressBandwidth.Equal(actualQuantity) {
					patchElement = append(patchElement, fmt.Sprintf(`{"op":"replace","path":"/metadata/annotations","value":{"kubernetes.io/egress-bandwidth":%s}}`, egressBandwidth.String()))
				}
			}
		}
	}

	patch := fmt.Sprintf(`[%s]`, strings.Join(patchElement, ","))
	patchType := v1.PatchTypeJSONPatch
	admissionResponse := new(admissionv1.AdmissionResponse)
	admissionResponse.Allowed = true
	admissionResponse.PatchType = &patchType
	admissionResponse.Patch = []byte(patch)

	var admissionReviewResponse admissionv1.AdmissionReview
	admissionReviewResponse.Response = admissionResponse
	admissionReviewResponse.SetGroupVersionKind(admissionReviewRequest.GroupVersionKind())
	admissionReviewResponse.Response.UID = admissionReviewRequest.Request.UID

	resp, err := json.Marshal(admissionReviewResponse)
	if err != nil {
		klog.Errorf("pod decode error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func (wh *Webhook) validatePod(w http.ResponseWriter, r *http.Request) {
	klog.Infoln("Pod: message on validate received")
	deserializer := wh.Codecs.UniversalDeserializer()
	admissionReviewRequest, err := admissionReviewFromRequest(r, deserializer)
	if err != nil {
		klog.Errorf("Pod admission review error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if admissionReviewRequest.Request.Resource != podResource {
		err := fmt.Errorf("pod wrong resource kind: %v", admissionReviewRequest.Request.Resource.Resource)
		klog.Error(err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	rawRequest := admissionReviewRequest.Request.Object.Raw
	pod := new(corev1.Pod)
	if _, _, err := deserializer.Decode(rawRequest, nil, pod); err != nil {
		klog.Errorf("pod decode error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	var ingressBandwidth resource.Quantity
	var egressBandwidth resource.Quantity
	for _, container := range pod.Spec.Containers {
		if quantity, ok := container.Resources.Requests["edge-net.io/ingress-bandwidth"]; !ok {
			ingressBandwidth.Add(quantity)
		}
		if quantity, ok := container.Resources.Requests["edge-net.io/egress-bandwidth"]; !ok {
			egressBandwidth.Add(quantity)
		}
	}

	admissionResponse := new(admissionv1.AdmissionResponse)
	admissionResponse.Allowed = true

	if !ingressBandwidth.IsZero() {
		if actualIngressBandwidth, ok := pod.Annotations["kubernetes.io/ingress-bandwidth"]; !ok {
			admissionResponse.Allowed = false
			admissionResponse.Result = &metav1.Status{
				Message: "missing annotation ingress-bandwidth",
			}
		} else {
			if _, err := resource.ParseQuantity(actualIngressBandwidth); err != nil {
				admissionResponse.Allowed = false
				admissionResponse.Result = &metav1.Status{
					Message: "parse ingress-bandwidth failed",
				}
			}
		}
	}
	if !egressBandwidth.IsZero() {
		if actualEgressBandwidth, ok := pod.Annotations["kubernetes.io/egress-bandwidth"]; !ok {
			admissionResponse.Allowed = false
			admissionResponse.Result = &metav1.Status{
				Message: "missing annotation egress-bandwidth",
			}
		} else {
			if _, err := resource.ParseQuantity(actualEgressBandwidth); err != nil {
				admissionResponse.Allowed = false
				admissionResponse.Result = &metav1.Status{
					Message: "parse egress-bandwidth failed",
				}
			}
		}
	}

	var admissionReviewResponse admissionv1.AdmissionReview
	admissionReviewResponse.Response = admissionResponse
	admissionReviewResponse.SetGroupVersionKind(admissionReviewRequest.GroupVersionKind())
	admissionReviewResponse.Response.UID = admissionReviewRequest.Request.UID

	resp, err := json.Marshal(admissionReviewResponse)
	if err != nil {
		klog.Errorf("pod decode error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func (wh *Webhook) validateTenantRequest(w http.ResponseWriter, r *http.Request) {
	klog.Infoln("TenantRequest: message on validate received")
	deserializer := wh.Codecs.UniversalDeserializer()
	admissionReviewRequest, err := admissionReviewFromRequest(r, deserializer)
	if err != nil {
		klog.Errorf("TenantRequest admission review error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	tenantrequestResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "tenantrequests"}
	if admissionReviewRequest.Request.Resource != tenantrequestResource {
		err := fmt.Errorf("tenantrequest wrong resource kind: %v", admissionReviewRequest.Request.Resource.Resource)
		klog.Error(err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	rawRequest := admissionReviewRequest.Request.Object.Raw
	tenantrequest := new(registrationv1alpha.TenantRequest)
	if _, _, err := deserializer.Decode(rawRequest, nil, tenantrequest); err != nil {
		klog.Errorf("tenantrequest decode error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	admissionResponse := new(admissionv1.AdmissionResponse)
	admissionResponse.Allowed = true

	if admissionReviewRequest.Request.Operation == "CREATE" && tenantrequest.Spec.Approved {
		admissionResponse.Allowed = false
		admissionResponse.Result = &metav1.Status{
			Message: "tenant request cannot hold approved status at creation",
		}
	}

	var admissionReviewResponse admissionv1.AdmissionReview
	admissionReviewResponse.Response = admissionResponse
	admissionReviewResponse.SetGroupVersionKind(admissionReviewRequest.GroupVersionKind())
	admissionReviewResponse.Response.UID = admissionReviewRequest.Request.UID

	resp, err := json.Marshal(admissionReviewResponse)
	if err != nil {
		klog.Errorf("tenantrequest decode error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func (wh *Webhook) validateClusterRoleRequest(w http.ResponseWriter, r *http.Request) {
	klog.Infoln("ClusterRoleRequest: message on validate received")
	deserializer := wh.Codecs.UniversalDeserializer()
	admissionReviewRequest, err := admissionReviewFromRequest(r, deserializer)
	if err != nil {
		klog.Errorf("ClusterRoleRequest admission review error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	clusterrolerequestResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "clusterrolerequests"}
	if admissionReviewRequest.Request.Resource != clusterrolerequestResource {
		err := fmt.Errorf("clusterrolerequest wrong resource kind: %v", admissionReviewRequest.Request.Resource.Resource)
		klog.Error(err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	rawRequest := admissionReviewRequest.Request.Object.Raw
	clusterrolerequest := new(registrationv1alpha.ClusterRoleRequest)
	if _, _, err := deserializer.Decode(rawRequest, nil, clusterrolerequest); err != nil {
		klog.Errorf("clusterrolerequest decode error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	admissionResponse := new(admissionv1.AdmissionResponse)
	admissionResponse.Allowed = true

	if admissionReviewRequest.Request.Operation == "CREATE" && clusterrolerequest.Spec.Approved {
		admissionResponse.Allowed = false
		admissionResponse.Result = &metav1.Status{
			Message: "tenant request cannot hold approved status at creation",
		}
	}

	var admissionReviewResponse admissionv1.AdmissionReview
	admissionReviewResponse.Response = admissionResponse
	admissionReviewResponse.SetGroupVersionKind(admissionReviewRequest.GroupVersionKind())
	admissionReviewResponse.Response.UID = admissionReviewRequest.Request.UID

	resp, err := json.Marshal(admissionReviewResponse)
	if err != nil {
		klog.Errorf("clusterrolerequest decode error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func admissionReviewFromRequest(r *http.Request, deserializer runtime.Decoder) (*admissionv1.AdmissionReview, error) {
	if r.Header.Get("Content-Type") != "application/json" {
		return nil, errors.New("expected content-type is application/json")
	}
	if r.Body == nil {
		return nil, errors.New("request body is empty")
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	admissionReviewRequest := new(admissionv1.AdmissionReview)
	if _, _, err := deserializer.Decode(body, nil, admissionReviewRequest); err != nil {
		return nil, err
	}
	return admissionReviewRequest, nil
}
