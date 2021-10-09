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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	labelSelector := flag.String("l", "", "k8s label selector, e.g env=prod")
	kind := flag.String("k", "pod", "k8s object kind")
	age := flag.Bool("a", false, "print also age")

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

	sort.Sort(customSort{objects, func(x, y *Object) bool {
		if x.cluster != y.cluster {
			return x.cluster < y.cluster
		}
		if x.namespace != y.namespace {
			return x.namespace < y.namespace
		}
		if x.count != y.count {
			return x.count > y.count
		}
		return false
	}})
	printObjects(objects, *age)

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
	newest        metav1.Time
	oldest        metav1.Time
}

func getCount(config *Config, kind, labelSelector *string) (*Object, error) {
	obj := Object{
		cluster:       config.cluster,
		namespace:     config.namespace,
		kind:          *kind,
		labelSelector: *labelSelector,
	}
	clientSet, err := kubernetes.NewForConfig(config.restConfig)
	if err != nil {
		return &obj, fmt.Errorf("generating clientSet: %v", err)
	}
	var n int
	var newest, oldest metav1.Time
	switch *kind {
	case "deployment", "deploy":
		n, newest, oldest, err = countDeployments(clientSet, config.namespace, *labelSelector)
	case "pod":
		n, newest, oldest, err = countPods(clientSet, config.namespace, *labelSelector)
	case "configMap", "configmap", "cm":
		n, newest, oldest, err = countConfigMaps(clientSet, config.namespace, *labelSelector)
	case "secret":
		n, newest, oldest, err = countSecrets(clientSet, config.namespace, *labelSelector)
	case "ingress", "ing":
		n, newest, oldest, err = countIngresses(clientSet, config.namespace, *labelSelector)
	default:
		return &obj, fmt.Errorf("unsupported kind: %s", *kind)
	}
	if err != nil {
		return &obj, fmt.Errorf("counting %s objects: %v", *kind, err)
	}
	obj.count = n
	obj.newest = newest
	obj.oldest = oldest
	return &obj, nil
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

func printObjects(objects []*Object, age bool) {
	tw := new(tabwriter.Writer).Init(os.Stdout, 0, 8, 2, ' ', 0)
	if age {
		const format = "%v\t%v\t%v\t%v\t%v\t%v\t%v\n"
		fmt.Fprintf(tw, format, "Cluster", "Namespace", "Label", "Kind", "Count", "Newest", "Oldest")
		fmt.Fprintf(tw, format, "-------", "---------", "-----", "----", "-----", "------", "------")
		for _, o := range objects {
			fmt.Fprintf(tw, format, o.cluster, o.namespace, o.labelSelector, o.kind, o.count, translateTimestampSince(o.newest), translateTimestampSince(o.oldest))
		}
	} else {
		const format = "%v\t%v\t%v\t%v\t%v\n"
		fmt.Fprintf(tw, format, "Cluster", "Namespace", "Label", "Kind", "Count")
		fmt.Fprintf(tw, format, "-------", "---------", "-----", "----", "-----")
		for _, o := range objects {
			fmt.Fprintf(tw, format, o.cluster, o.namespace, o.labelSelector, o.kind, o.count)
		}
	}
	tw.Flush()
}

// translateTimestampSince returns the elapsed time since timestamp in
// human-readable approximation.
func translateTimestampSince(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}

	return duration.HumanDuration(time.Since(timestamp.Time))
}

// customSort sorts Objects according to less function.
type customSort struct {
	o    []*Object
	less func(x, y *Object) bool
}

func (x customSort) Len() int           { return len(x.o) }
func (x customSort) Less(i, j int) bool { return x.less(x.o[i], x.o[j]) }
func (x customSort) Swap(i, j int)      { x.o[i], x.o[j] = x.o[j], x.o[i] }

func countDeployments(clientset *kubernetes.Clientset, namespace string, labelSelector string) (int, metav1.Time, metav1.Time, error) {
	deployments, err := clientset.AppsV1().Deployments(namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: labelSelector})
	var newest, oldest metav1.Time
	for i, obj := range deployments.Items {
		t := obj.CreationTimestamp
		if i == 0 {
			newest, oldest = t, t
			continue
		}
		if t.After(newest.Time) {
			newest = t
		}
		if t.Before(&oldest) {
			oldest = t
		}
	}
	return len(deployments.Items), newest, oldest, err
}

func countPods(clientset *kubernetes.Clientset, namespace string, labelSelector string) (int, metav1.Time, metav1.Time, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: labelSelector})
	var newest, oldest metav1.Time
	for i, obj := range pods.Items {
		t := obj.CreationTimestamp
		if i == 0 {
			newest, oldest = t, t
			continue
		}
		if t.After(newest.Time) {
			newest = t
		}
		if t.Before(&oldest) {
			oldest = t
		}
	}
	return len(pods.Items), newest, oldest, err
}

func countConfigMaps(clientset *kubernetes.Clientset, namespace string, labelSelector string) (int, metav1.Time, metav1.Time, error) {
	configMaps, err := clientset.CoreV1().ConfigMaps(namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: labelSelector})
	var newest, oldest metav1.Time
	for i, obj := range configMaps.Items {
		t := obj.CreationTimestamp
		if i == 0 {
			newest, oldest = t, t
			continue
		}
		if t.After(newest.Time) {
			newest = t
		}
		if t.Before(&oldest) {
			oldest = t
		}
	}
	return len(configMaps.Items), newest, oldest, err
}

func countSecrets(clientset *kubernetes.Clientset, namespace string, labelSelector string) (int, metav1.Time, metav1.Time, error) {
	secrets, err := clientset.CoreV1().Secrets(namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: labelSelector})
	var newest, oldest metav1.Time
	for i, obj := range secrets.Items {
		t := obj.CreationTimestamp
		if i == 0 {
			newest, oldest = t, t
			continue
		}
		if t.After(newest.Time) {
			newest = t
		}
		if t.Before(&oldest) {
			oldest = t
		}
	}
	return len(secrets.Items), newest, oldest, err
}

func countIngresses(clientset *kubernetes.Clientset, namespace string, labelSelector string) (int, metav1.Time, metav1.Time, error) {
	ingresses, err := clientset.NetworkingV1().Ingresses(namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: labelSelector})
	var newest, oldest metav1.Time
	for i, obj := range ingresses.Items {
		t := obj.CreationTimestamp
		if i == 0 {
			newest, oldest = t, t
			continue
		}
		if t.After(newest.Time) {
			newest = t
		}
		if t.Before(&oldest) {
			oldest = t
		}
	}
	return len(ingresses.Items), newest, oldest, err
}
