package fedmanctl

import (
	b64 "encoding/base64"
	"encoding/json"
)

// Contains information about the federation. Do not share this token,
// it contians sensitive information.
type WorkerClusterInfo struct {
	// These 3 fields are gathered from the secret created on worker cluster
	CACertificate string `json:"ca.crt"`
	Namespace     string `json:"namespace"`
	Token         string `json:"token"`

	// UID of the kube-system namespace
	UID string `json:"uid"`

	// Cluster IP/Port information
	ClusterIP   string `json:"clusterIP"`
	ClusterPort string `json:"clusterPort"`

	// Can be "Public" or "Private"
	Visibility string `json:"visibility"`

	// Labels of the cluster
	Labels map[string]string `json:"labels"`
}

// Converts the WorkerClusterInfo object to a base64 encoded string
func TokenizeWorkerClusterInfo(w *WorkerClusterInfo) (string, error) {
	// Remove empty labels to reduce token size
	strippedLabels := make(map[string]string, len(w.Labels))

	for label, value := range w.Labels {
		if value != "" {
			strippedLabels[label] = value
		}
	}

	w.Labels = strippedLabels

	src, err := json.Marshal(w)

	if err != nil {
		return "", err
	}

	return b64.StdEncoding.EncodeToString(src), nil
}

// Retrieves the WorkerClusterInfo object from the base64 encoded token
func DetokenizeWorkerClusterInfo(token string) (*WorkerClusterInfo, error) {
	src, err := b64.StdEncoding.DecodeString(token)

	if err != nil {
		return nil, err
	}

	w := &WorkerClusterInfo{}

	err = json.Unmarshal(src, w)

	if err != nil {
		return nil, err
	}

	return w, nil
}
