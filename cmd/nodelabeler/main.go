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

// This nodelabeler feature includes GeoLite2 data created by MaxMind, available from
// https://www.maxmind.com.

package main

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/EdgeNet-project/edgenet/pkg/controller/core/v1/nodelabeler"
	"github.com/EdgeNet-project/edgenet/pkg/signals"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/klog"
)

func main() {
	klog.InitFlags(nil)
	flag.String("kubeconfig-path", bootstrap.GetDefaultKubeconfigPath(), "Path to the kubeconfig file's directory")
	flag.Parse()

	stopCh := signals.SetupSignalHandler()
	var authentication string
	if authentication = strings.TrimSpace(os.Getenv("AUTHENTICATION_STRATEGY")); authentication != "kubeconfig" {
		authentication = "serviceaccount"
	}
	config, err := bootstrap.GetRestConfig(authentication)
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	kubeclientset, err := bootstrap.CreateKubeClientset(config)
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	edgenetclientset, err := bootstrap.CreateEdgeNetClientset(config)
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}

	maxmindUrl := strings.TrimSpace(os.Getenv("MAXMIND_URL"))
	if maxmindUrl == "" {
		maxmindUrl = "https://geoip.maxmind.com/geoip/v2.1/city/"
	}
	maxmindAccountId := strings.TrimSpace(os.Getenv("MAXMIND_ACCOUNT_ID"))
	maxmindLicenseKey := strings.TrimSpace(os.Getenv("MAXMIND_LICENSE_KEY"))

	// Start the controller to provide the functionalities of nodelabeler resource
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeclientset, time.Second*30)

	controller := nodelabeler.NewController(
		kubeclientset,
		edgenetclientset,
		kubeInformerFactory.Core().V1().Nodes(),
		maxmindUrl,
		maxmindAccountId,
		maxmindLicenseKey,
	)

	kubeInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}
