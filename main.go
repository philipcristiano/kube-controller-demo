/*
Copyright 2017 The Kubernetes Authors.

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

package main

import (
    "fmt"
    "flag"

    appsv1 "k8s.io/api/apps/v1"
    // corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog"

    "k8s.io/client-go/informers"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/cache"
    "k8s.io/client-go/tools/clientcmd"
	// clientset "k8s.io/sample-controller/pkg/generated/clientset/versioned"
	// "k8s.io/sample-controller/pkg/signals"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

)

var (
	masterURL  string
	kubeconfig string
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	// stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

    factory := informers.NewSharedInformerFactory(kubeClient, 0)
    deploymentsInformer := factory.Apps().V1().Deployments().Informer()
    stopper := make(chan struct{})
    defer close(stopper)
    defer runtime.HandleCrash()
    deploymentsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc: onAdd,
    })
    go deploymentsInformer.Run(stopper)
    if !cache.WaitForCacheSync(stopper, deploymentsInformer.HasSynced) {
        runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
        return
    }
    <-stopper
}

// onAdd is the function executed when the kubernetes informer notified the
// presence of a new kubernetes node in the cluster
func onAdd(obj interface{}) {
    deployment := obj.(*appsv1.Deployment)
    fmt.Println("Deployment: " + deployment.ObjectMeta.Name)
    // Cast the obj as node
    // node := obj.(*corev1.Node)
    // _, ok := node.GetLabels()["label"]
    // if ok {
    //     fmt.Printf("It has the label!")
    // }
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
