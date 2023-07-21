package admissioncontrol

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"

	corev1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha1"
	registrationv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha1"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/klog/v2"
)

const (
	reserved = "Reserved"
	bound    = "Bound"
)

type Webhook struct {
	CertFile string
	KeyFile  string
	Codecs   serializer.CodecFactory
	Runtime  string
	Port     string
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
	http.HandleFunc("/validate/role-request", wh.validateRoleRequest)
	http.HandleFunc("/validate/subnamespace", wh.validateSubNamespace)
	http.HandleFunc("/validate/slice", wh.validateSlice)
	http.HandleFunc("/validate/slice-claim", wh.validateSliceClaim)

	server := http.Server{
		Addr: ":8080",
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
		ingressBandwidth.Add(*container.Resources.Limits.Name("edge-net.io/ingress-bandwidth", resource.BinarySI))
		egressBandwidth.Add(*container.Resources.Limits.Name("edge-net.io/egress-bandwidth", resource.BinarySI))
	}
	patchOperation := map[string]string{}
	if !ingressBandwidth.IsZero() {
		if actualIngressBandwidth, ok := pod.Annotations["kubernetes.io/ingress-bandwidth"]; !ok {
			patchOperation["ingress"] = "add"
		} else {
			if actualQuantity, err := resource.ParseQuantity(actualIngressBandwidth); err != nil {
				patchOperation["ingress"] = "replace"
			} else {
				if !ingressBandwidth.Equal(actualQuantity) {
					patchOperation["ingress"] = "replace"
				}
			}
		}
	}
	if !egressBandwidth.IsZero() {
		if actualEgressBandwidth, ok := pod.Annotations["kubernetes.io/egress-bandwidth"]; !ok {
			patchOperation["egress"] = "add"
		} else {
			if actualQuantity, err := resource.ParseQuantity(actualEgressBandwidth); err != nil {
				patchOperation["egress"] = "replace"
			} else {
				if !egressBandwidth.Equal(actualQuantity) {
					patchOperation["egress"] = "replace"
				}
			}
		}
	}

	if slice, sliceExists := pod.Spec.NodeSelector["edge-net.io/slice"]; sliceExists && slice == "none" {
		if pod.Spec.RuntimeClassName == nil {
			patchOperation["runtime"] = "add"
		} else {
			patchOperation["runtime"] = "replace"
		}
	}

	admissionResponse := new(admissionv1.AdmissionResponse)
	admissionResponse.Allowed = true

	_, ingressExists := patchOperation["ingress"]
	_, egressExists := patchOperation["egress"]
	_, runtimeExists := patchOperation["runtime"]

	var patchItems []string
	if ingressExists {
		ingress := fmt.Sprintf(`{"op":"%s","path":"/metadata/annotations","value":{"kubernetes.io/ingress-bandwidth":"%s"}}`, patchOperation["ingress"], ingressBandwidth.String())
		patchItems = append(patchItems, ingress)
	}
	if egressExists {
		egress := fmt.Sprintf(`{"op":"%s","path":"/metadata/annotations","value":{"kubernetes.io/egress-bandwidth":"%s"}}`, patchOperation["egress"], egressBandwidth.String())
		patchItems = append(patchItems, egress)
	}
	if runtimeExists {
		runtime := fmt.Sprintf(`{"op":"%s","path":"/spec/runtimeClassName","value":"%s"}`, patchOperation["runtime"], wh.Runtime)
		patchItems = append(patchItems, runtime)
	}
	patch := fmt.Sprintf(`[%s]`, strings.Join(patchItems, ","))
	patchType := admissionv1.PatchTypeJSONPatch
	admissionResponse.PatchType = &patchType
	admissionResponse.Patch = []byte(patch)

	/*if !ingressBandwidth.IsZero() || !egressBandwidth.IsZero() {
		var patch string
		if patchOperation["ingress"] == patchOperation["egress"] {
			patch = fmt.Sprintf(`[{"op":"%s","path":"/metadata/annotations","value":{"kubernetes.io/ingress-bandwidth":"%s", "kubernetes.io/egress-bandwidth":"%s"}}]`, patchOperation["ingress"], ingressBandwidth.String(), egressBandwidth.String())
		} else {
			ingress := fmt.Sprintf(`{"op":"%s","path":"/metadata/annotations","value":{"kubernetes.io/ingress-bandwidth":"%s"}}`, patchOperation["ingress"], ingressBandwidth.String())
			egress := fmt.Sprintf(`{"op":"%s","path":"/metadata/annotations","value":{"kubernetes.io/egress-bandwidth":"%s"}}`, patchOperation["egress"], egressBandwidth.String())
			patch = fmt.Sprintf(`[%s]`, strings.Join([]string{ingress, egress}, ","))
		}
		klog.Infoln(patch)
		patchType := v1.PatchTypeJSONPatch
		admissionResponse.PatchType = &patchType
		admissionResponse.Patch = []byte(patch)
	} else {
		klog.Infoln("Pod: no bandwidth requested")
	}*/

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
		ingressBandwidth.Add(*container.Resources.Limits.Name("edge-net.io/ingress-bandwidth", resource.BinarySI))
		egressBandwidth.Add(*container.Resources.Limits.Name("edge-net.io/egress-bandwidth", resource.BinarySI))
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

	tenantrequestResource := metav1.GroupVersionResource{Group: "registration.edgenet.io", Version: "v1alpha1", Resource: "tenantrequests"}
	if admissionReviewRequest.Request.Resource != tenantrequestResource {
		err := fmt.Errorf("tenantrequest wrong resource kind: %v", admissionReviewRequest.Request.Resource.Resource)
		klog.Error(err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	rawRequest := admissionReviewRequest.Request.Object.Raw
	tenantrequest := new(registrationv1alpha1.TenantRequest)
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
			Message: "tenant request cannot be approved at creation",
		}
	}

	if admissionReviewRequest.Request.UserInfo.Username != tenantrequest.Spec.Contact.Email {
		admissionResponse.Allowed = false
		admissionResponse.Result = &metav1.Status{
			Message: "username, which is an email address, and contact email address must be the same",
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

	clusterrolerequestResource := metav1.GroupVersionResource{Group: "registration.edgenet.io", Version: "v1alpha1", Resource: "clusterrolerequests"}
	if admissionReviewRequest.Request.Resource != clusterrolerequestResource {
		err := fmt.Errorf("clusterrolerequest wrong resource kind: %v", admissionReviewRequest.Request.Resource.Resource)
		klog.Error(err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	rawRequest := admissionReviewRequest.Request.Object.Raw
	clusterrolerequest := new(registrationv1alpha1.ClusterRoleRequest)
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
			Message: "cluster role request cannot be approved at creation",
		}
	}

	if admissionReviewRequest.Request.UserInfo.Username != clusterrolerequest.Spec.Email {
		admissionResponse.Allowed = false
		admissionResponse.Result = &metav1.Status{
			Message: "username, which is an email address, and email address must be the same",
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

func (wh *Webhook) validateRoleRequest(w http.ResponseWriter, r *http.Request) {
	klog.Infoln("RoleRequest: message on validate received")
	deserializer := wh.Codecs.UniversalDeserializer()
	admissionReviewRequest, err := admissionReviewFromRequest(r, deserializer)
	if err != nil {
		klog.Errorf("RoleRequest admission review error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	rolerequestResource := metav1.GroupVersionResource{Group: "registration.edgenet.io", Version: "v1alpha1", Resource: "rolerequests"}
	if admissionReviewRequest.Request.Resource != rolerequestResource {
		err := fmt.Errorf("rolerequest wrong resource kind: %v", admissionReviewRequest.Request.Resource.Resource)
		klog.Error(err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	rawRequest := admissionReviewRequest.Request.Object.Raw
	rolerequest := new(registrationv1alpha1.RoleRequest)
	if _, _, err := deserializer.Decode(rawRequest, nil, rolerequest); err != nil {
		klog.Errorf("rolerequest decode error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	admissionResponse := new(admissionv1.AdmissionResponse)
	admissionResponse.Allowed = true

	if admissionReviewRequest.Request.Operation == "CREATE" && rolerequest.Spec.Approved {
		admissionResponse.Allowed = false
		admissionResponse.Result = &metav1.Status{
			Message: "role request cannot be approved at creation",
		}
	}

	if admissionReviewRequest.Request.UserInfo.Username != rolerequest.Spec.Email {
		admissionResponse.Allowed = false
		admissionResponse.Result = &metav1.Status{
			Message: "username, which is an email address, and email address must be the same",
		}
	}

	var admissionReviewResponse admissionv1.AdmissionReview
	admissionReviewResponse.Response = admissionResponse
	admissionReviewResponse.SetGroupVersionKind(admissionReviewRequest.GroupVersionKind())
	admissionReviewResponse.Response.UID = admissionReviewRequest.Request.UID

	resp, err := json.Marshal(admissionReviewResponse)
	if err != nil {
		klog.Errorf("rolerequest decode error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func (wh *Webhook) validateSubNamespace(w http.ResponseWriter, r *http.Request) {
	klog.Infoln("SubNamespace: message on validate received")
	deserializer := wh.Codecs.UniversalDeserializer()
	admissionReviewRequest, err := admissionReviewFromRequest(r, deserializer)
	if err != nil {
		klog.Errorf("SubNamespace admission review error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	subnamespaceResource := metav1.GroupVersionResource{Group: "core.edgenet.io", Version: "v1alpha1", Resource: "subnamespaces"}
	if admissionReviewRequest.Request.Resource != subnamespaceResource {
		err := fmt.Errorf("subnamespace wrong resource kind: %v", admissionReviewRequest.Request.Resource.Resource)
		klog.Error(err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	rawRequest := admissionReviewRequest.Request.Object.Raw
	subnamespace := new(corev1alpha1.SubNamespace)
	if _, _, err := deserializer.Decode(rawRequest, nil, subnamespace); err != nil {
		klog.Errorf("subnamespace decode error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	admissionResponse := new(admissionv1.AdmissionResponse)
	admissionResponse.Allowed = true
	if admissionReviewRequest.Request.Operation == "CREATE" {
		if subnamespace.GetSliceClaim() != nil && subnamespace.GetResourceAllocation() != nil {
			admissionResponse.Allowed = false
			admissionResponse.Result = &metav1.Status{
				Message: "subsidiary namespace slice and resource allocation cannot be set at creation",
			}
		}
	}

	if admissionReviewRequest.Request.Operation == "UPDATE" || admissionReviewRequest.Request.Operation == "PATCH" {
		oldObjectRaw := admissionReviewRequest.Request.OldObject.Raw
		oldSubnamespace := new(corev1alpha1.SubNamespace)
		if _, _, err := deserializer.Decode(oldObjectRaw, nil, oldSubnamespace); err != nil {
			klog.Errorf("old subnamespace decode error: %v", err)
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
			return
		}

		if subnamespace.Spec.Workspace != nil && oldSubnamespace.Spec.Workspace.Scope != subnamespace.Spec.Workspace.Scope {
			admissionResponse.Allowed = false
			admissionResponse.Result = &metav1.Status{
				Message: "subsidiary namespace scope cannot be changed after creation",
			}
		}
		if (oldSubnamespace.Spec.Subtenant == nil && subnamespace.Spec.Subtenant != nil) || (oldSubnamespace.Spec.Workspace == nil && subnamespace.Spec.Workspace != nil) {
			admissionResponse.Allowed = false
			admissionResponse.Result = &metav1.Status{
				Message: "subsidiary namespace mode cannot be changed after creation",
			}
		}

		if oldSubnamespace.GetSliceClaim() != nil && subnamespace.GetSliceClaim() != nil {
			if *oldSubnamespace.GetSliceClaim() != *subnamespace.GetSliceClaim() {
				admissionResponse.Allowed = false
				admissionResponse.Result = &metav1.Status{
					Message: "subsidiary namespace slice cannot be set after creation",
				}
			}
		}

		if subnamespace.GetSliceClaim() != nil && !reflect.DeepEqual(oldSubnamespace.GetResourceAllocation(), subnamespace.GetResourceAllocation()) && admissionReviewRequest.Request.UserInfo.Username != "system:serviceaccount:edgenet:subnamespace" {
			klog.Infoln(admissionReviewRequest.Request.UserInfo.Username)
			admissionResponse.Allowed = false
			admissionResponse.Result = &metav1.Status{
				Message: "subsidiary namespace resource allocation cannot be updated when a slice is applied",
			}
		}
	}

	var admissionReviewResponse admissionv1.AdmissionReview
	admissionReviewResponse.Response = admissionResponse
	admissionReviewResponse.SetGroupVersionKind(admissionReviewRequest.GroupVersionKind())
	admissionReviewResponse.Response.UID = admissionReviewRequest.Request.UID

	resp, err := json.Marshal(admissionReviewResponse)
	if err != nil {
		klog.Errorf("subnamespace decode error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func (wh *Webhook) validateSlice(w http.ResponseWriter, r *http.Request) {
	klog.Infoln("Slice: message on validate received")
	deserializer := wh.Codecs.UniversalDeserializer()
	admissionReviewRequest, err := admissionReviewFromRequest(r, deserializer)
	if err != nil {
		klog.Errorf("Slice admission review error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	sliceResource := metav1.GroupVersionResource{Group: "core.edgenet.io", Version: "v1alpha1", Resource: "slices"}
	if admissionReviewRequest.Request.Resource != sliceResource {
		err := fmt.Errorf("slice wrong resource kind: %v", admissionReviewRequest.Request.Resource.Resource)
		klog.Error(err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	rawRequest := admissionReviewRequest.Request.Object.Raw
	slice := new(corev1alpha1.Slice)
	if _, _, err := deserializer.Decode(rawRequest, nil, slice); err != nil {
		klog.Errorf("slice decode error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	admissionResponse := new(admissionv1.AdmissionResponse)
	admissionResponse.Allowed = true

	if admissionReviewRequest.Request.Operation == "UPDATE" || admissionReviewRequest.Request.Operation == "PATCH" {
		oldObjectRaw := admissionReviewRequest.Request.OldObject.Raw
		oldSlice := new(corev1alpha1.Slice)
		if _, _, err := deserializer.Decode(oldObjectRaw, nil, oldSlice); err != nil {
			klog.Errorf("old slice decode error: %v", err)
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
			return
		}
		if oldSlice.Status.State == reserved || oldSlice.Status.State == bound {
			if oldSlice.Spec.SliceClassName != slice.Spec.SliceClassName {
				admissionResponse.Allowed = false
				admissionResponse.Result = &metav1.Status{
					Message: "slice class name cannot be changed after nodes are reserved",
				}
			}
			if !reflect.DeepEqual(oldSlice.Spec.NodeSelector, slice.Spec.NodeSelector) {
				admissionResponse.Allowed = false
				admissionResponse.Result = &metav1.Status{
					Message: "node selector cannot be changed after nodes are reserved",
				}
			}
			if oldSlice.Spec.ClaimRef != slice.Spec.ClaimRef && oldSlice.Status.State == bound {
				admissionResponse.Allowed = false
				admissionResponse.Result = &metav1.Status{
					Message: "slice claim cannot be changed after slice is bound",
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
		klog.Errorf("slice decode error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func (wh *Webhook) validateSliceClaim(w http.ResponseWriter, r *http.Request) {
	klog.Infoln("SliceClaim: message on validate received")
	deserializer := wh.Codecs.UniversalDeserializer()
	admissionReviewRequest, err := admissionReviewFromRequest(r, deserializer)
	if err != nil {
		klog.Errorf("SliceClaim admission review error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	sliceclaimResource := metav1.GroupVersionResource{Group: "core.edgenet.io", Version: "v1alpha1", Resource: "sliceclaims"}
	if admissionReviewRequest.Request.Resource != sliceclaimResource {
		err := fmt.Errorf("sliceclaim wrong resource kind: %v", admissionReviewRequest.Request.Resource.Resource)
		klog.Error(err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	rawRequest := admissionReviewRequest.Request.Object.Raw
	sliceclaim := new(corev1alpha1.SliceClaim)
	if _, _, err := deserializer.Decode(rawRequest, nil, sliceclaim); err != nil {
		klog.Errorf("sliceclaim decode error: %v", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	admissionResponse := new(admissionv1.AdmissionResponse)
	admissionResponse.Allowed = true

	if admissionReviewRequest.Request.Operation == "UPDATE" || admissionReviewRequest.Request.Operation == "PATCH" {
		oldObjectRaw := admissionReviewRequest.Request.OldObject.Raw
		oldSliceClaim := new(corev1alpha1.SliceClaim)
		if _, _, err := deserializer.Decode(oldObjectRaw, nil, oldSliceClaim); err != nil {
			klog.Errorf("old sliceclaim decode error: %v", err)
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
			return
		}

		if oldSliceClaim.Spec.SliceClassName != sliceclaim.Spec.SliceClassName {
			admissionResponse.Allowed = false
			admissionResponse.Result = &metav1.Status{
				Message: "slice class name cannot be changed after creation",
			}
		}
		if !reflect.DeepEqual(oldSliceClaim.Spec.NodeSelector, sliceclaim.Spec.NodeSelector) {
			admissionResponse.Allowed = false
			admissionResponse.Result = &metav1.Status{
				Message: "node selector cannot be changed after creation",
			}
		}
		if oldSliceClaim.Spec.SliceName != sliceclaim.Spec.SliceName {
			admissionResponse.Allowed = false
			admissionResponse.Result = &metav1.Status{
				Message: "slice name cannot be changed after creation",
			}
		}
	}

	var admissionReviewResponse admissionv1.AdmissionReview
	admissionReviewResponse.Response = admissionResponse
	admissionReviewResponse.SetGroupVersionKind(admissionReviewRequest.GroupVersionKind())
	admissionReviewResponse.Response.UID = admissionReviewRequest.Request.UID

	resp, err := json.Marshal(admissionReviewResponse)
	if err != nil {
		klog.Errorf("sliceclaim decode error: %v", err)
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
