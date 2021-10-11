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

type Flags struct {
	labelSelector string
	kind          string
	age           bool
	timeout       int64
}

func ParseFlags() Flags {
	var f Flags
	flag.StringVar(&f.labelSelector, "l", "", "k8s label selector, e.g env=prod")
	flag.StringVar(&f.kind, "k", "pod", "k8s object kind")
	flag.BoolVar(&f.age, "a", false, "print also age")
	flag.Int64Var(&f.timeout, "t", 5, "cluster API call timeout in seconds")
	flag.Parse()
	return f
}

func main() {
	flags := ParseFlags()

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
	var objects []Object
	var wg sync.WaitGroup
	for _, config := range configs {
		wg.Add(1)
		go func(config *Config) {
			defer wg.Done()
			obj, err := getCount(config, flags.kind, flags.labelSelector, flags.timeout)
			if err != nil {
				log.Print(err)
				return
			}
			mu.Lock()
			objects = append(objects, obj)
			mu.Unlock()
		}(config)
	}
	wg.Wait()

	sort.Slice(objects, func(i, j int) bool {
		if objects[i].cluster != objects[j].cluster {
			return objects[i].cluster < objects[j].cluster
		}
		if objects[i].namespace != objects[j].namespace {
			return objects[i].namespace < objects[j].namespace
		}
		if objects[i].count != objects[j].count {
			return objects[i].count > objects[j].count
		}
		return false
	})
	printObjects(objects, flags.age)
}

// Config represents kubernetes cluster configuration obtained from within a
// cluster or from a kubeconfig file.
type Config struct {
	restConfig         *rest.Config
	cluster, namespace string
}

func getConfigs(kubeconfigs []string) ([]*Config, error) {
	configs, err := getConfigsFromKubeconfigs(kubeconfigs)
	if err != nil {
		return nil, err
	}

	configClust, err := getConfigFromCluster()
	// If we are not in a cluster just return configs from kubeconfigs.
	if err != nil && err == rest.ErrNotInCluster {
		return configs, nil
	}
	if err != nil {
		return nil, err
	}
	configs = append(configs, configClust)

	return configs, err
}

func getConfigsFromKubeconfigs(kubeconfigs []string) ([]*Config, error) {
	var configs []*Config

	for _, kubeconfig := range kubeconfigs {
		apiConfig, err := clientcmd.LoadFromFile(kubeconfig)
		if err != nil {
			return nil, err
		}

		clientConfig := clientcmd.NewDefaultClientConfig(*apiConfig, nil)
		restConfig, err := clientConfig.ClientConfig()
		if err != nil {
			return nil, err
		}

		namespace := apiConfig.Contexts[apiConfig.CurrentContext].Namespace
		cluster := apiConfig.Contexts[apiConfig.CurrentContext].Cluster

		configs = append(configs, &Config{restConfig: restConfig, namespace: namespace, cluster: cluster})
	}

	return configs, nil
}

func getConfigFromCluster() (*Config, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	ns, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return nil, err
	}
	return &Config{restConfig: restConfig, namespace: string(ns)}, nil
}

type ObjectTime metav1.Time

// String returns the elapsed time since timestamp in
// human-readable approximation.
func (o ObjectTime) String() string {
	if o.IsZero() {
		return "<unknown>"
	}
	return duration.HumanDuration(time.Since(o.Time))
}

// Object represents a kubernetes object.
type Object struct {
	cluster       string
	namespace     string
	kind          string
	labelSelector string
	count         int
	newest        ObjectTime
	oldest        ObjectTime
}

func getCount(config *Config, kind, labelSelector string, timeout int64) (Object, error) {
	clientSet, err := kubernetes.NewForConfig(config.restConfig)
	if err != nil {
		return Object{}, fmt.Errorf("generating clientSet: %v", err)
	}
	var n int
	var newest, oldest metav1.Time
	switch kind {
	case "deployment", "deploy":
		n, newest, oldest, err = countDeployments(clientSet, config.namespace, labelSelector, timeout)
	case "pod":
		n, newest, oldest, err = countPods(clientSet, config.namespace, labelSelector, timeout)
	case "configMap", "configmap", "cm":
		n, newest, oldest, err = countConfigMaps(clientSet, config.namespace, labelSelector, timeout)
	case "secret":
		n, newest, oldest, err = countSecrets(clientSet, config.namespace, labelSelector, timeout)
	case "ingress", "ing":
		n, newest, oldest, err = countIngresses(clientSet, config.namespace, labelSelector, timeout)
	default:
		return Object{}, fmt.Errorf("unsupported kind: %s", kind)
	}
	if err != nil {
		return Object{}, fmt.Errorf("counting %s objects: %v", kind, err)
	}

	return Object{
		cluster:       config.cluster,
		namespace:     config.namespace,
		kind:          kind,
		labelSelector: labelSelector,
		count:         n,
		newest:        ObjectTime(newest),
		oldest:        ObjectTime(oldest),
	}, nil
}

// func countsEqual(objects []Object) bool {
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

func printObjects(objects []Object, age bool) {
	tw := new(tabwriter.Writer).Init(os.Stdout, 0, 8, 2, ' ', 0)
	if age {
		const format = "%v\t%v\t%v\t%v\t%v\t%s\t%s\n"
		fmt.Fprintf(tw, format, "Cluster", "Namespace", "Label", "Kind", "Count", "Newest", "Oldest")
		fmt.Fprintf(tw, format, "-------", "---------", "-----", "----", "-----", "------", "------")
		for _, o := range objects {
			fmt.Fprintf(tw, format, o.cluster, o.namespace, o.labelSelector, o.kind, o.count, o.newest, o.oldest)
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

func countDeployments(clientset *kubernetes.Clientset, namespace string, labelSelector string, timeoutSeconds int64) (int, metav1.Time, metav1.Time, error) {
	deployments, err := clientset.AppsV1().Deployments(namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeoutSeconds})
	if err != nil {
		return 0, metav1.Time{}, metav1.Time{}, err
	}

	var items []metav1.ObjectMeta
	for _, item := range deployments.Items {
		items = append(items, item.ObjectMeta)
	}
	count, newest, oldest := countItems(items)
	return count, newest, oldest, nil
}

func countPods(clientset *kubernetes.Clientset, namespace string, labelSelector string, timeoutSeconds int64) (int, metav1.Time, metav1.Time, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeoutSeconds})
	if err != nil {
		return 0, metav1.Time{}, metav1.Time{}, err
	}

	var items []metav1.ObjectMeta
	for _, item := range pods.Items {
		items = append(items, item.ObjectMeta)
	}
	count, newest, oldest := countItems(items)
	return count, newest, oldest, nil
}

func countConfigMaps(clientset *kubernetes.Clientset, namespace string, labelSelector string, timeoutSeconds int64) (int, metav1.Time, metav1.Time, error) {
	configMaps, err := clientset.CoreV1().ConfigMaps(namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeoutSeconds})
	if err != nil {
		return 0, metav1.Time{}, metav1.Time{}, err
	}

	var items []metav1.ObjectMeta
	for _, item := range configMaps.Items {
		items = append(items, item.ObjectMeta)
	}
	count, newest, oldest := countItems(items)
	return count, newest, oldest, nil
}

func countSecrets(clientset *kubernetes.Clientset, namespace string, labelSelector string, timeoutSeconds int64) (int, metav1.Time, metav1.Time, error) {
	secrets, err := clientset.CoreV1().Secrets(namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeoutSeconds})
	if err != nil {
		return 0, metav1.Time{}, metav1.Time{}, err
	}

	var items []metav1.ObjectMeta
	for _, item := range secrets.Items {
		items = append(items, item.ObjectMeta)
	}
	count, newest, oldest := countItems(items)
	return count, newest, oldest, nil
}

func countIngresses(clientset *kubernetes.Clientset, namespace string, labelSelector string, timeoutSeconds int64) (int, metav1.Time, metav1.Time, error) {
	ingresses, err := clientset.NetworkingV1().Ingresses(namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeoutSeconds})
	if err != nil {
		return 0, metav1.Time{}, metav1.Time{}, err
	}

	var items []metav1.ObjectMeta
	for _, item := range ingresses.Items {
		items = append(items, item.ObjectMeta)
	}
	count, newest, oldest := countItems(items)
	return count, newest, oldest, nil
}

func countItems(items []metav1.ObjectMeta) (int, metav1.Time, metav1.Time) {
	var newest, oldest metav1.Time
	for i, item := range items {
		t := item.CreationTimestamp
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
	return len(items), newest, oldest
}
