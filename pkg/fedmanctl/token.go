package fedmanctl

import (
	b64 "encoding/base64"
	"encoding/json"
)

type WorkerClusterToken struct {
	CACertificate string `json:"ca.crt"`
	Namespace     string `json:"namespace"`
	Token         string `json:"token"`
	UID           string `json:"uid"`
}

func Tokenize(w *WorkerClusterToken) (string, error) {
	src, err := json.Marshal(w)

	if err != nil {
		return "", err
	}

	return b64.StdEncoding.EncodeToString(src), nil
}

func Detokenize(token string) (*WorkerClusterToken, error) {
	src, err := b64.StdEncoding.DecodeString(token)

	if err != nil {
		return nil, err
	}

	w := &WorkerClusterToken{}

	err = json.Unmarshal(src, w)

	if err != nil {
		return nil, err
	}

	return w, nil
}
