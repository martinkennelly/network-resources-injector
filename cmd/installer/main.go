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

	"github.com/golang/glog"
	"github.com/k8snetworkplumbingwg/network-resources-injector/pkg/installer"
)

func main() {
	namespace := flag.String("namespace", "kube-system", "Namespace in which all Kubernetes resources will be created.")
	prefix := flag.String("name", "network-resources-injector", "Prefix added to the names of all created resources.")
	failurePolicy := flag.String("failure-policy", "Ignore", "K8 admission controller failure policy to handle unrecognized errors and timeout errors")
	flag.Parse()

	glog.Info("starting webhook installation")
	installer.Install(*namespace, *prefix, *failurePolicy)
}
