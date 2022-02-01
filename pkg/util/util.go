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

package util

import (
	"fmt"
	"hash/adler32"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	yaml "gopkg.in/yaml.v2"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	//cmdconfig "k8s.io/kubernetes/pkg/kubectl/cmd/config"
	//cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

// A part of the general structure of a kubeconfig file
type clusterDetails struct {
	CA     []byte `json:"certificate-authority-data"`
	Server string `json:"server"`
}
type clusters struct {
	Cluster clusterDetails `json:"cluster"`
	Name    string         `json:"name"`
}
type contextDetails struct {
	Cluster string `json:"cluster"`
	User    string `json:"user"`
}
type contexts struct {
	Context contextDetails `json:"context"`
	Name    string         `json:"name"`
}
type configView struct {
	Clusters       []clusters `json:"clusters"`
	Contexts       []contexts `json:"contexts"`
	CurrentContext string     `json:"current-context"`
}

// Structure of Namecheap access credentials
type namecheap struct {
	App      string `yaml:"app"`
	APIUser  string `yaml:"apiUser"`
	APIToken string `yaml:"apiToken"`
	Username string `yaml:"username"`
}

// This reads the kubeconfig file by admin context and returns it in json format.
func getConfigView() (api.Config, error) {
	/*pathOptions := clientcmd.NewDefaultPathOptions()
	streamsIn := &bytes.Buffer{}
	streamsOut := &bytes.Buffer{}
	streamsErrOut := &bytes.Buffer{}
	streams := genericclioptions.IOStreams{
		In:     streamsIn,
		Out:    streamsOut,
		ErrOut: streamsErrOut,
	}*/

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	// if you want to change the loading rules (which files in which order), you can do so here

	configOverrides := &clientcmd.ConfigOverrides{}
	// if you want to change override values or bind them to flags, there are methods to help you

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	rawConfig, err := kubeConfig.RawConfig()
	if err != nil {
		// Do something
		return rawConfig, err
	}

	/*configCmd := cmdconfig.NewCmdConfigView(cmdutil.NewFactory(genericclioptions.NewConfigFlags(false)), streams, pathOptions)
	// "context" is a global flag, inherited from base kubectl command in the real world
	configCmd.Flags().String("context", "kubernetes-admin@kubernetes", "The name of the kubeconfig context to use")
	configCmd.Flags().Parse([]string{"--minify", "--output=json", "--raw=true"})
	if err := configCmd.Execute(); err != nil {
		log.Printf("unexpected error executing command: %v", err)
		return "", err
	}

	fmt.Sprintf
	output := fmt.Sprint(streams.Out)*/
	return rawConfig, nil
}

// GetClusterServerOfCurrentContext provides cluster and server info of the current context
func GetClusterServerOfCurrentContext() (string, string, []byte, error) {
	rawConfig, err := getConfigView()
	if err != nil {
		log.Printf("unexpected error executing command: %v", err)
		return "", "", nil, err
	}
	/*
		var configViewDet configView
		err = json.Unmarshal([]byte(configStr), &configViewDet)
		if err != nil {
			log.Printf("unexpected error executing command: %v", err)
			return "", "", nil, err
		}*/

	// currentContext := rawConfig.CurrentContext

	// var cluster string = rawConfig.Contexts[currentContext].Cluster
	/*for _, contextRaw := range rawConfig.Contexts {
		if contextRaw.Name == currentContext {
			cluster = contextRaw.Context.Cluster
		}
	}*/
	var server string = rawConfig.Clusters["kubernetes"].Server
	var CA []byte = rawConfig.Clusters["kubernetes"].CertificateAuthorityData
	/*for _, clusterRaw := range rawConfig.Clusters {
		if clusterRaw.Name == cluster {
			server = clusterRaw.Cluster.Server
			CA = clusterRaw.Cluster.CA
		}
	}*/
	return "kubernetes", server, CA, nil
}

// GetServerOfCurrentContext provides the server info of the current context
func GetServerOfCurrentContext() (string, error) {
	rawConfig, err := getConfigView()
	if err != nil {
		log.Printf("unexpected error executing command: %v", err)
		return "", err
	}
	/*var configViewDet configView
	err = json.Unmarshal([]byte(configStr), &configViewDet)
	if err != nil {
		log.Printf("unexpected error executing command: %v", err)
		return "", err
	}*/
	// currentContext := rawConfig.CurrentContext

	// var cluster string = rawConfig.Contexts[currentContext].Cluster
	/*for _, contextRaw := range configViewDet.Contexts {
		if contextRaw.Name == currentContext {
			cluster = contextRaw.Context.Cluster
		}
	}*/
	var server string = rawConfig.Clusters["kubernetes"].Server
	/*for _, clusterRaw := range configViewDet.Clusters {
		if clusterRaw.Name == cluster {
			server = clusterRaw.Cluster.Server
		}
	}*/
	return server, nil
}

// GetNamecheapCredentials provides authentication info to have API Access
func GetNamecheapCredentials() (string, string, string, error) {
	// The path of the yaml config file of namecheap
	file, err := os.Open("../../configs/namecheap.yaml")
	if err != nil {
		log.Printf("unexpected error executing command: %v", err)
		return "", "", "", err
	}

	decoder := yaml.NewDecoder(file)
	var namecheap namecheap
	err = decoder.Decode(&namecheap)
	if err != nil {
		log.Printf("unexpected error executing command: %v", err)
		return "", "", "", err
	}
	return namecheap.APIUser, namecheap.APIToken, namecheap.Username, nil
}

// GenerateRandomString to have a unique code
func GenerateRandomString(n int) string {
	var letter = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

	b := make([]rune, n)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}

// Contains returns whether slice contains the value
func Contains(slice []string, value string) (bool, int) {
	for i, ele := range slice {
		if value == ele {
			return true, i
		}
	}
	return false, 0
}

// SliceContains returns whether slice contains the slice
func SliceContains(slice [][]string, value []string) (bool, int) {
	for i, ele := range slice {
		if reflect.DeepEqual(value, ele) {
			return true, i
		}
	}
	return false, 0
}

// Assert fails the test if the condition is false.
func Assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		//tb.FailNow()
		tb.Fail()
	}
}

// OK fails the test if an err is not nil.
func OK(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		//tb.FailNow()
		tb.Fail()
	}
}

// Equals fails the test if exp is not equal to act.
func Equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		//tb.FailNow()
		tb.Fail()
	}
}

// NotEquals fails the test if exp is equal to act.
func NotEquals(tb testing.TB, exp, act interface{}) {
	if reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp different from: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		//tb.FailNow()
		tb.Fail()
	}
}

// EqualsMultipleExp fails the test if exp is not equal to one of act.
func EqualsMultipleExp(tb testing.TB, exp interface{}, act interface{}) {
	check := func(exp, act interface{}) bool {
		fail := true
		if !reflect.DeepEqual(exp, act) {
			_, file, line, _ := runtime.Caller(1)
			fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		} else {
			fail = false
		}
		return fail
	}
	if reflect.TypeOf(exp).Kind() == reflect.Slice {
		val := reflect.ValueOf(exp)
		expRaw, ok := val.Interface().([]string)
		if !ok {
			expRaw, ok := val.Interface().([]int)
			if !ok {
				expRaw, ok := val.Interface().([]bool)
				if !ok {
					Equals(tb, exp, act)
				} else {
					fail := true
					for _, expRow := range expRaw {
						fail = check(expRow, act)
						if !fail {
							break
						}
					}
					if fail {
						tb.Fail()
					}
				}
			} else {
				fail := true
				for _, expRow := range expRaw {
					fail = check(expRow, act)
					if !fail {
						break
					}
				}
				if fail {
					tb.Fail()
				}
			}
		} else {
			fail := true
			for _, expRow := range expRaw {
				fail = check(expRow, act)
				if !fail {
					break
				}
			}
			if fail {
				tb.Fail()
			}
		}
	} else {
		Equals(tb, exp, act)
	}
}

func Hash(strs ...string) (string, error) {
	str := strings.Join(strs, "-")
	hasher := adler32.New()
	if hash, err := hasher.Write([]byte(str)); err == nil {
		return strconv.Itoa(hash), nil
	} else {
		return "", err
	}
}
