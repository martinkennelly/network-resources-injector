// Copyright (c) 2019 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"github.com/k8snetworkplumbingwg/network-resources-injector/pkg/ca"
	"github.com/k8snetworkplumbingwg/network-resources-injector/pkg/keycert"
	server2 "github.com/k8snetworkplumbingwg/network-resources-injector/pkg/server"
	"github.com/k8snetworkplumbingwg/network-resources-injector/pkg/service"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/k8snetworkplumbingwg/network-resources-injector/pkg/webhook"
)

const (
	defaultClientCa               = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	readTo                        = 5 * time.Second
	writeTo                       = 10 * time.Second
	readHeaderTo                  = 1 * time.Second
	serviceTo                     = 2 * time.Second
	ciInterval                    = 30 * time.Second
)

func main() {
	var namespace string
	var clientCAPaths ca.ClientCAFlags
	/* load configuration */
	port := flag.Int("port", 8443, "The port on which to serve.")
	address := flag.String("bind-address", "0.0.0.0", "The IP address on which to listen for the --port port.")
	cert := flag.String("tls-cert-file", "cert.pem", "File containing the default x509 Certificate for HTTPS.")
	key := flag.String("tls-private-key-file", "key.pem", "File containing the default x509 private key matching --tls-cert-file.")
	insecure := flag.Bool("insecure", false, "Disable adding client CA to server TLS endpoint --insecure")
	injectHugepageDownApi := flag.Bool("injectHugepageDownApi", false, "Enable hugepage requests and limits into Downward API.")
	flag.Var(&clientCAPaths, "client-ca", "File containing client CA. This flag is repeatable if more than one client CA needs to be added to server")
	resourceNameKeys := flag.String("network-resource-name-keys", "k8s.v1.cni.cncf.io/resourceName", "comma separated resource name keys --network-resource-name-keys.")
	resourcesHonorFlag := flag.Bool("honor-resources", false, "Honor the existing requested resources requests & limits --honor-resources")
	flag.Parse()

	if *port < 1024 || *port > 65535 {
		glog.Fatalf("invalid port number. Choose between 1024 and 65535")
	}

	if *address == "" || *cert == "" || *key == "" || *resourceNameKeys == "" {
		glog.Fatalf("input argument(s) not defined correctly")
	}

	if len(clientCAPaths) == 0 {
		clientCAPaths = append(clientCAPaths, defaultClientCa)
	}

	if namespace = os.Getenv("NAMESPACE"); namespace == "" {
		namespace = "kube-system"
	}

	glog.Infof("starting mutating admission controller for network resources injection")

	keyPair, err := keycert.NewIdentity(*cert, *key)
	if err != nil {
		glog.Fatalf("error load certificate: %s", err.Error())
	}

	clientCaPool, err := ca.NewClientCertPool(&clientCAPaths, *insecure)
	if err != nil {
		glog.Fatalf("error loading client CA pool: '%s'", err.Error())
	}

	webhook.SetInjectHugepageDownApi(*injectHugepageDownApi)

	webhook.SetHonorExistingResources(*resourcesHonorFlag)

	if err = webhook.SetResourceNameKeys(*resourceNameKeys); err != nil {
		glog.Fatalf("error in setting resource name keys: %s", err.Error())
	}

	if err = webhook.SetupInClusterClient(); err != nil {
		glog.Fatalf("error trying to setup kubernetes client")
	}

	kp := .NewKeyPair(keyPair, serviceTo)
	if err = kp.Run(); err != nil {
		glog.Fatalf("starting TLS key & cert file updater failed: '%s'", err.Error())
	}

	udi := webhook.GetUDI(namespace, ciInterval, serviceTo)
	if err = udi.Run(); err != nil {
		err = webhook.CombineError(err, kp.Quit())
		glog.Fatalf("starting user defined injection updater failed: '%s'", err.Error())
	}

	server := server2.NewMutateServer(*address, *port, *insecure, readTo, writeTo, readHeaderTo, serviceTo, clientCaPool, keyPair)
	if err = server.Run(); err != nil {
		err = webhook.CombineError(err, kp.Quit(), udi.Quit())
		glog.Fatalf("starting HTTP server failed: '%s'", err.Error())
	}

	/* Blocks until termination or TLS key/cert file updater or UDI updater or HTTP server signal occurs */
	if err := service.Watch(make(chan os.Signal, 1), kp, udi, server); err != nil {
		glog.Error(err.Error())
	}
}
