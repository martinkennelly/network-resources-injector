// Copyright (c) 2018 Intel Corporation
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

package installer

import (
	"context"
	"io/ioutil"
	"strings"
	"time"

	"github.com/cloudflare/cfssl/csr"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	arv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	clientset kubernetes.Interface
	namespace string
	prefix    string
)

const keyBitLength = 3072

func generateCSR() ([]byte, []byte, error) {
	glog.Infof("generating Certificate Signing Request")
	serviceName := strings.Join([]string{prefix, "service"}, "-")
	certRequest := csr.New()
	certRequest.KeyRequest = &csr.KeyRequest{A: "rsa", S: keyBitLength}
	certRequest.CN = strings.Join([]string{serviceName, namespace, "svc"}, ".")
	certRequest.Hosts = []string{
		serviceName,
		strings.Join([]string{serviceName, namespace}, "."),
		strings.Join([]string{serviceName, namespace, "svc"}, "."),
	}
	return csr.ParseRequest(certRequest)
}

func getSignedCertificate(request []byte) ([]byte, error) {
	csrName := strings.Join([]string{prefix, "csr"}, "-")
	csr, err := clientset.CertificatesV1beta1().CertificateSigningRequests().Get(context.TODO(), csrName, metav1.GetOptions{})
	if csr != nil || err == nil {
		glog.Infof("CSR %s already exists, removing it first", csrName)
		_ = clientset.CertificatesV1beta1().CertificateSigningRequests().Delete(context.TODO(), csrName, metav1.DeleteOptions{})
	}

	glog.Infof("creating new CSR %s", csrName)
	/* build Kubernetes CSR object */
	csr = &v1beta1.CertificateSigningRequest{}
	csr.ObjectMeta.Name = csrName
	csr.ObjectMeta.Namespace = namespace
	csr.Spec.Request = request
	csr.Spec.Groups = []string{"system:authenticated"}
	csr.Spec.Usages = []v1beta1.KeyUsage{v1beta1.UsageDigitalSignature, v1beta1.UsageServerAuth, v1beta1.UsageKeyEncipherment}

	/* push CSR to Kubernetes API server */
	csr, err = clientset.CertificatesV1beta1().CertificateSigningRequests().Create(context.TODO(), csr, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "error creating CSR in Kubernetes API: %s")
	}
	glog.Infof("CSR pushed to the Kubernetes API")

	if csr.Status.Certificate != nil {
		glog.Infof("using already issued certificate for CSR %s", csrName)
		return csr.Status.Certificate, nil
	}
	/* approve certificate in K8s API */
	csr.ObjectMeta.Name = csrName
	csr.ObjectMeta.Namespace = namespace
	csr.Status.Conditions = append(csr.Status.Conditions, v1beta1.CertificateSigningRequestCondition{
		Type:           v1beta1.CertificateApproved,
		Reason:         "Approved by net-attach-def admission controller installer",
		Message:        "This CSR was approved by net-attach-def admission controller installer.",
		LastUpdateTime: metav1.Now(),
	})
	_, err = clientset.CertificatesV1beta1().CertificateSigningRequests().UpdateApproval(context.TODO(), csr, metav1.UpdateOptions{})
	glog.Infof("certificate approval sent")
	if err != nil {
		return nil, errors.Wrap(err, "error approving CSR in Kubernetes API")
	}

	/* wait for the cert to be issued */
	glog.Infof("waiting for the signed certificate to be issued...")
	start := time.Now()
	ticker := time.NewTicker(time.Second)

	for range ticker.C {
		csr, err = clientset.CertificatesV1beta1().CertificateSigningRequests().Get(context.TODO(), csrName, metav1.GetOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "error getting signed ceritificate from the API server")
		}
		if csr.Status.Certificate != nil {
			return csr.Status.Certificate, nil
		}
		if time.Since(start) > 60*time.Second {
			break
		}
	}

	return nil, errors.New("error getting certificate from the API server: request timed out - verify that Kubernetes certificate signer is setup, more at https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/#a-note-to-cluster-administrators")
}

func writeToFile(certificate, key []byte, certFilename, keyFilename string) error {
	if err := ioutil.WriteFile("/etc/tls/"+certFilename, certificate, 0400); err != nil {
		return err
	}
	if err := ioutil.WriteFile("/etc/tls/"+keyFilename, key, 0400); err != nil {
		return err
	}
	return nil
}

func createMutatingWebhookConfiguration(certificate []byte) error {
	configName := strings.Join([]string{prefix, "mutating-config"}, "-")
	serviceName := strings.Join([]string{prefix, "service"}, "-")
	removeMutatingWebhookIfExists(configName)
	failurePolicy := arv1beta1.Ignore
	path := "/mutate"
	configuration := &arv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: configName,
			Labels: map[string]string{
				"app": prefix,
			},
		},
		Webhooks: []arv1beta1.MutatingWebhook{
			arv1beta1.MutatingWebhook{
				Name: configName + ".k8s.cni.cncf.io",
				ClientConfig: arv1beta1.WebhookClientConfig{
					CABundle: certificate,
					Service: &arv1beta1.ServiceReference{
						Namespace: namespace,
						Name:      serviceName,
						Path:      &path,
					},
				},
				FailurePolicy: &failurePolicy,
				Rules: []arv1beta1.RuleWithOperations{
					arv1beta1.RuleWithOperations{
						Operations: []arv1beta1.OperationType{arv1beta1.Create},
						Rule: arv1beta1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
						},
					},
				},
			},
		},
	}
	_, err := clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(context.TODO(), configuration, metav1.CreateOptions{})
	return err
}

func createService() error {
	serviceName := strings.Join([]string{prefix, "service"}, "-")
	removeServiceIfExists(serviceName)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: serviceName,
			Labels: map[string]string{
				"app": prefix,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Port:       443,
					TargetPort: intstr.FromInt(8443),
				},
			},
			Selector: map[string]string{
				"app": prefix,
			},
		},
	}
	_, err := clientset.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
	return err
}

func removeServiceIfExists(serviceName string) {
	service, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if service != nil && err == nil {
		glog.Infof("service %s already exists, removing it first", serviceName)
		err := clientset.CoreV1().Services(namespace).Delete(context.TODO(), serviceName, metav1.DeleteOptions{})
		if err != nil {
			glog.Errorf("error trying to remove service: %s", err)
		}
		glog.Infof("service %s removed", serviceName)
	}
}

func removeMutatingWebhookIfExists(configName string) {
	config, err := clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(context.TODO(), configName, metav1.GetOptions{})
	if config != nil && err == nil {
		glog.Infof("mutating webhook %s already exists, removing it first", configName)
		err := clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Delete(context.TODO(), configName, metav1.DeleteOptions{})
		if err != nil {
			glog.Errorf("error trying to remove mutating webhook configuration: %s", err)
		}
		glog.Infof("mutating webhook configuration %s removed", configName)
	}
}

// Install creates resources required by mutating admission webhook
func Install(k8sNamespace, namePrefix string) {
	/* setup Kubernetes API client */
	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("error loading Kubernetes in-cluster configuration: %s", err)
	}
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("error setting up Kubernetes client: %s", err)
	}

	namespace = k8sNamespace
	prefix = namePrefix

	/* generate CSR and private key */
	csr, key, err := generateCSR()
	if err != nil {
		glog.Fatalf("error generating CSR and private key: %s", err)
	}
	glog.Infof("raw CSR and private key successfully created")

	/* obtain signed certificate */
	certificate, err := getSignedCertificate(csr)
	if err != nil {
		glog.Fatalf("error getting signed certificate: %s", err)
	}
	glog.Infof("signed certificate successfully obtained")

	err = writeToFile(certificate, key, "tls.crt", "tls.key")
	if err != nil {
		glog.Fatalf("error writing certificate and key to files: %s", err)
	}
	glog.Infof("certificate and key written to files")

	/* create webhook configurations */
	err = createMutatingWebhookConfiguration(certificate)
	if err != nil {
		glog.Fatalf("error creating mutating webhook configuration: %s", err)
	}
	glog.Infof("mutating webhook configuration successfully created")

	/* create service */
	err = createService()
	if err != nil {
		glog.Fatalf("error creating service: %s", err)
	}
	glog.Infof("service successfully created")

	glog.Infof("all resources created successfully")
}
