package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"sync"
	"text/tabwriter"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	labelSelector := flag.String("l", "", "k8s label selector, e.g env=prod")
	kind := flag.String("k", "pod", "k8s object kind")
	flag.Parse()

	log.SetPrefix(os.Args[0] + ": ")
	log.SetFlags(0)

	var configs []*Config
	configs, err := getConfigs(flag.Args())
	if err != nil {
		log.Fatalf("getting configs: %v", err)
	}
	if len(configs) == 0 {
		log.Fatal("run in cluster or supply at least one kubeconfig")
	}

	var mu sync.Mutex
	var objects []*Object
	var wg sync.WaitGroup
	for _, config := range configs {
		wg.Add(1)
		go func(config *Config) {
			obj, err := getCount(config, kind, labelSelector)
			if err != nil {
				log.Print(err)
			}
			mu.Lock()
			objects = append(objects, obj)
			mu.Unlock()
			defer wg.Done()
		}(config)
	}
	wg.Wait()

	sort.Sort(sort.Reverse(byCount(objects)))
	printObjects(objects)
}

// Config represents kubernetes cluster configuration obtained from within a
// cluster or from a kubeconfig file.
type Config struct {
	restConfig         *rest.Config
	cluster, namespace string
}

func getConfigs(kubeconfigs []string) ([]*Config, error) {
	var configs []*Config

	restConfig, err := rest.InClusterConfig()
	if err != nil && err != rest.ErrNotInCluster {
		return configs, err
	}
	if err != rest.ErrNotInCluster { // we are in cluster
		data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			return configs, err
		}
		configs = append(configs, &Config{restConfig: restConfig, namespace: string(data)})
	}

	for _, kubeconfig := range kubeconfigs {
		apiConfig, err := clientcmd.LoadFromFile(kubeconfig)
		if err != nil {
			return configs, err
		}

		clientConfig := clientcmd.NewDefaultClientConfig(*apiConfig, nil)
		restConfig, err := clientConfig.ClientConfig()
		if err != nil {
			return configs, err
		}

		namespace := apiConfig.Contexts[apiConfig.CurrentContext].Namespace
		cluster := apiConfig.Contexts[apiConfig.CurrentContext].Cluster

		configs = append(configs, &Config{restConfig: restConfig, namespace: namespace, cluster: cluster})
	}

	return configs, nil
}

// Object represents a kubernetes object.
type Object struct {
	cluster       string
	namespace     string
	kind          string
	labelSelector string
	count         int
}

func getCount(config *Config, kind, labelSelector *string) (*Object, error) {
	object := Object{
		cluster:       config.cluster,
		namespace:     config.namespace,
		kind:          *kind,
		labelSelector: *labelSelector,
	}
	clientSet, err := kubernetes.NewForConfig(config.restConfig)
	if err != nil {
		return &object, fmt.Errorf("generating clientSet: %v", err)
	}
	var n int
	switch *kind {
	case "deployment", "deploy":
		n, err = countDeployments(clientSet, config.namespace, *labelSelector)
	case "pod":
		n, err = countPods(clientSet, config.namespace, *labelSelector)
	case "configMap", "configmap", "cm":
		n, err = countConfigMaps(clientSet, config.namespace, *labelSelector)
	case "secret":
		n, err = countSecrets(clientSet, config.namespace, *labelSelector)
	case "ingress", "ing":
		n, err = countIngresses(clientSet, config.namespace, *labelSelector)
	default:
		return &object, fmt.Errorf("unsupported kind: %s", *kind)
	}
	if err != nil {
		return &object, fmt.Errorf("counting %s objects: %v", *kind, err)
	}
	object.count = n
	return &object, nil
}

// func countsEqual(objects []*Object) bool {
// 	var countFirst int
// 	for i, o := range objects {
// 		if i == 0 {
// 			countFirst = o.count
// 			continue
// 		}
// 		if countFirst != o.count {
// 			return false
// 		}
// 	}
// 	return true
// }

func printObjects(objects []*Object) {
	const format = "%v\t%v\t%v\t%v\t%v\n"
	tw := new(tabwriter.Writer).Init(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintf(tw, format, "Cluster", "Namespace", "Label Selector", "Kind", "Count")
	fmt.Fprintf(tw, format, "-------", "---------", "--------------", "----", "-----")
	for _, o := range objects {
		fmt.Fprintf(tw, format, o.cluster, o.namespace, o.labelSelector, o.kind, o.count)
	}
	tw.Flush()
}

type byCount []*Object

func (x byCount) Len() int           { return len(x) }
func (x byCount) Less(i, j int) bool { return x[i].count < x[j].count }
func (x byCount) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }

func countDeployments(clientset *kubernetes.Clientset, namespace string, labelSelector string) (int, error) {
	deployments, err := clientset.AppsV1().Deployments(namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: labelSelector})
	return len(deployments.Items), err
}

func countPods(clientset *kubernetes.Clientset, namespace string, labelSelector string) (int, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: labelSelector})
	return len(pods.Items), err
}

func countConfigMaps(clientset *kubernetes.Clientset, namespace string, labelSelector string) (int, error) {
	configMaps, err := clientset.CoreV1().ConfigMaps(namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: labelSelector})
	return len(configMaps.Items), err
}

func countSecrets(clientset *kubernetes.Clientset, namespace string, labelSelector string) (int, error) {
	secrets, err := clientset.CoreV1().Secrets(namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: labelSelector})
	return len(secrets.Items), err
}

func countIngresses(clientset *kubernetes.Clientset, namespace string, labelSelector string) (int, error) {
	ingresses, err := clientset.NetworkingV1().Ingresses(namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: labelSelector})
	return len(ingresses.Items), err
}
