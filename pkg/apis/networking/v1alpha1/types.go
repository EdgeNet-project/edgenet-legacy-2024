/*
Copyright 2021 Contributors to the EdgeNet project.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VPNPeer describes a WireGuard peer
type VPNPeer struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the vpnpeer resource spec
	Spec VPNPeerSpec `json:"spec"`
}

// VPNPeerSpec is the spec for a VPNPeer resource
type VPNPeerSpec struct {
	// IPv4 address of VPN peer.
	AddressV4 string `json:"addressV4"`
	// IPv6 address of VPN peer.
	AddressV6 string `json:"addressV6"`
	// Endpoint address of the VPN tunnel.
	EndpointAddress *string `json:"endpointAddress"`
	// Endpoint port of the VPN tunnel.
	EndpointPort *int `json:"endpointPort"`
	// VPN public key of the peer.
	PublicKey string `json:"publicKey"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VPNPeerList is a list of VPNPeer resources
type VPNPeerList struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	metav1.ListMeta `json:"metadata"`
	// VPNPeerList is a list of VPNPeer resources thus, VPNPeers are contained here.
	Items []VPNPeer `json:"items"`
}
