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
	"flag"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2b2 "k8s.io/api/autoscaling/v2beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

var (
	masterURL  string
	kubeconfig string
)

var newHPAs = make(chan autoscalingv2b2.HorizontalPodAutoscaler)

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
	go applyHPAs(kubeClient, newHPAs)
	<-stopper
	close(newHPAs)
}

func applyHPAs(client *kubernetes.Clientset, HPAsToApply chan autoscalingv2b2.HorizontalPodAutoscaler) {
	fmt.Printf("Waiting for HPAs to apply\n")
	for hpa := range HPAsToApply {
		fmt.Printf("Should try and apply an HPA\n%s\n", hpa)
		namespace := hpa.ObjectMeta.Namespace
		autoscalingClient := client.AutoscalingV2beta2().HorizontalPodAutoscalers(namespace)
		result, err := autoscalingClient.Create(&hpa)
		fmt.Printf("Namespace: %s\n\n%s, %s\n", namespace, result, err)
	}

}

// onAdd is the function executed when the kubernetes informer notified the
// presence of a new kubernetes node in the cluster
func onAdd(obj interface{}) {
	deployment := obj.(*appsv1.Deployment)
	fmt.Println("Deployment: " + deployment.ObjectMeta.Name)
	for k, v := range deployment.Spec.Template.ObjectMeta.Labels {
		fmt.Printf("Template Label: ")
		fmt.Printf("key[%s] value[%s]\n", k, v)
	}
	// for k, v := range deployment.ObjectMeta.Annotations {
	// 	fmt.Printf("Annotation: ")
	// 	fmt.Printf("key[%s] value[%s]\n", k, v)
	// }

	demo_status, ok := deployment.ObjectMeta.Annotations["kube-controller-demo"]

	if ok && demo_status == "enable" {
		fmt.Printf("Should try and create an HPA\n")
		var utilization = int32(80)
		var metricSource = autoscalingv2b2.ResourceMetricSource{
			Name: "CPU",
			Target: autoscalingv2b2.MetricTarget{
				Type:               "Utilization",
				AverageUtilization: &utilization,
			},
		}

		metadata := metav1.ObjectMeta{
			Name: deployment.ObjectMeta.Name,
			Namespace: deployment.ObjectMeta.Namespace,
			Labels: deployment.ObjectMeta.Labels,
		}

		hpa := autoscalingv2b2.HorizontalPodAutoscaler{
			ObjectMeta: metadata,
			Spec: autoscalingv2b2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2b2.CrossVersionObjectReference{
					Kind:       "Deployment",
					Name:       deployment.ObjectMeta.Name,
					APIVersion: "apps/v1",
				},
				MaxReplicas: 5,
				Metrics: []autoscalingv2b2.MetricSpec{autoscalingv2b2.MetricSpec{
					Type:     "Resource",
					Resource: &metricSource,
				}},
			},
		}
		newHPAs <- hpa

	}

	// Cast the obj as node
	// node := obj.(*corev1.Node)
	// _, ok := node.GetLabels()["label"]
	// if ok {
	//     fmt.Printf("It has the label!")
	// }
	fmt.Printf("Done with Deployment \n\n")
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
