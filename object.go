package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/client-go/kubernetes"
)

// K8sObject represents count and age of a Kubernetes object. The object is of a
// given kind, in a given cluster and namespace and matching a given label
// selector.
type K8sObject struct {
	cluster       string
	namespace     string
	kind          string
	labelSelector string
	count         int
	newest        objectTime
	oldest        objectTime
}

const timeout = 5 // cluster API call timeout in seconds

// CountObjectsAcrossClusters counts objects across all clusters concurrently.
func CountObjectsAcrossClusters(clusters []Cluster, flags Flags) []K8sObject {
	var objects []K8sObject
	ch := make(chan K8sObject)

	for _, cluster := range clusters {
		for _, kind := range flags.kind {
			go func(cluster Cluster, kind string) {
				obj, err := CountObjects(cluster, kind, flags.labelSelector)
				if err != nil {
					log.Printf("counting objects in cluster %s: %v", cluster.cluster, err)
				}
				ch <- obj
			}(cluster, kind)
		}
	}

	for range clusters {
		for range flags.kind {
			obj := <-ch
			if obj != (K8sObject{}) { // check obj is not "empty"
				objects = append(objects, obj)
			}
		}
	}

	return objects
}

// CountObjects counts objects of kind within a cluster.
func CountObjects(cluster Cluster, kind, labelSelector string) (K8sObject, error) {
	clientSet, err := kubernetes.NewForConfig(cluster.restConfig)
	if err != nil {
		return K8sObject{}, fmt.Errorf("generating clientSet: %v", err)
	}

	var n int
	var newest, oldest metav1.Time
	switch kind {
	case "deployment":
		n, newest, oldest, err = countDeployments(clientSet, cluster.namespace, labelSelector, timeout)
	case "pod":
		n, newest, oldest, err = countPods(clientSet, cluster.namespace, labelSelector, timeout)
	case "configmap":
		n, newest, oldest, err = countConfigMaps(clientSet, cluster.namespace, labelSelector, timeout)
	case "secret":
		n, newest, oldest, err = countSecrets(clientSet, cluster.namespace, labelSelector, timeout)
	case "ingress":
		n, newest, oldest, err = countIngresses(clientSet, cluster.namespace, labelSelector, timeout)
	default:
		return K8sObject{}, fmt.Errorf("unsupported kind: %s", kind)
	}
	if err != nil {
		return K8sObject{}, fmt.Errorf("counting %s objects: %v", kind, err)
	}

	return K8sObject{
		cluster:       cluster.cluster,
		namespace:     cluster.namespace,
		kind:          kind,
		labelSelector: labelSelector,
		count:         n,
		newest:        objectTime(newest),
		oldest:        objectTime(oldest),
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

// PrintObjects prints a table with Kubernetes objects.
func PrintObjects(objects []K8sObject, age bool) {
	if len(objects) == 0 {
		return
	}
	tw := new(tabwriter.Writer).Init(os.Stdout, 0, 8, 2, ' ', 0)

	if age {
		const format = "%v\t%v\t%v\t%v\t%v\t%s\t%s\n"
		fmt.Fprintf(tw, format, "Cluster", "Namespace", "Label selector", "Kind", "Count", "Newest", "Oldest")
		fmt.Fprintf(tw, format, "-------", "---------", "--------------", "----", "-----", "------", "------")
		for _, o := range objects {
			fmt.Fprintf(tw, format, o.cluster, o.namespace, o.labelSelector, o.kind, o.count, o.newest, o.oldest)
		}
	} else {
		const format = "%v\t%v\t%v\t%v\t%v\n"
		fmt.Fprintf(tw, format, "Cluster", "Namespace", "Label selector", "Kind", "Count")
		fmt.Fprintf(tw, format, "-------", "---------", "--------------", "----", "-----")
		for _, o := range objects {
			fmt.Fprintf(tw, format, o.cluster, o.namespace, o.labelSelector, o.kind, o.count)
		}
	}

	tw.Flush()
}

// SortObjects sorts objects by count and then by cluster name and namespace
// name.
func SortObjects(objects []K8sObject) {
	sort.Slice(objects, func(i, j int) bool {
		if objects[i].count != objects[j].count {
			return objects[i].count > objects[j].count
		}
		if objects[i].cluster != objects[j].cluster {
			return objects[i].cluster < objects[j].cluster
		}
		if objects[i].namespace != objects[j].namespace {
			return objects[i].namespace < objects[j].namespace
		}
		return false
	})
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

type objectTime metav1.Time

// String returns the elapsed time since timestamp in
// human-readable approximation.
func (o objectTime) String() string {
	if o.IsZero() {
		return "<unknown>"
	}
	return duration.HumanDuration(time.Since(o.Time))
}
