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

const timeout = 5 // cluster API call timeout in seconds

// Count represents count and age of Kubernetes objects. The objects are inside
// a given cluster and namespace, of given kind and matching given label
// selector.
type Count struct {
	Cluster       string
	Namespace     string
	Kind          string
	LabelSelector string
	Count         int
	Newest        objectTime
	Oldest        objectTime
}

// CountObjectsAcrossClusters counts objects across all clusters concurrently.
func CountObjectsAcrossClusters(clusters []Cluster, flags Flags) Counts {
	var counts []Count
	ch := make(chan Count)

	for _, cluster := range clusters {
		for _, kind := range flags.kind {
			go func(cluster Cluster, kind string) {
				obj, err := countObjects(cluster, kind, flags.labelSelector)
				if err != nil {
					log.Printf("counting objects in cluster '%s': %v", cluster.cluster, err)
				}
				ch <- obj
			}(cluster, kind)
		}
	}

	for range clusters {
		for range flags.kind {
			obj := <-ch
			if obj != (Count{}) { // check obj is not "empty"
				counts = append(counts, obj)
			}
		}
	}

	return counts
}

// countObjects counts objects of kind within a cluster.
func countObjects(cluster Cluster, kind, labelSelector string) (Count, error) {
	clientSet, err := kubernetes.NewForConfig(cluster.restConfig)
	if err != nil {
		return Count{}, fmt.Errorf("generating clientSet: %v", err)
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
	case "service":
		n, newest, oldest, err = countServices(clientSet, cluster.namespace, labelSelector, timeout)
	default:
		return Count{}, fmt.Errorf("unsupported kind: %s", kind)
	}
	if err != nil {
		return Count{}, fmt.Errorf("counting %s objects: %v", kind, err)
	}

	return Count{
		Cluster:       cluster.cluster,
		Namespace:     cluster.namespace,
		Kind:          kind,
		LabelSelector: labelSelector,
		Count:         n,
		Newest:        objectTime(newest),
		Oldest:        objectTime(oldest),
	}, nil
}

type Counts []Count

// Sort sorts objects by count and then by cluster name and namespace
// name.
func (c Counts) Sort() {
	sort.Slice(c, func(i, j int) bool {
		if c[i].Count != c[j].Count {
			return c[i].Count > c[j].Count
		}
		if c[i].Kind != c[j].Kind {
			return c[i].Kind < c[j].Kind
		}
		if c[i].Cluster != c[j].Cluster {
			return c[i].Cluster < c[j].Cluster
		}
		if c[i].Namespace != c[j].Namespace {
			return c[i].Namespace < c[j].Namespace
		}
		return false
	})
}

// Print prints a table with Kubernetes objects. The table can optionally
// contain the age of the objects.
func (c Counts) Print(age bool) {
	if len(c) == 0 {
		return
	}
	tw := new(tabwriter.Writer).Init(os.Stdout, 0, 8, 2, ' ', 0)

	var total int

	if age {
		const format = "%v\t%v\t%v\t%v\t%v\t%s\t%s\n"
		fmt.Fprintf(tw, format, "Cluster", "Namespace", "Label selector", "Kind", "Count", "Newest", "Oldest")
		fmt.Fprintf(tw, format, "-------", "---------", "--------------", "----", "-----", "------", "------")
		for _, o := range c {
			total += o.Count
			fmt.Fprintf(tw, format, o.Cluster, o.Namespace, o.LabelSelector, o.Kind, o.Count, o.Newest, o.Oldest)
		}
		fmt.Fprintf(tw, format, "", "", "", "", "-----", "", "")
		fmt.Fprintf(tw, format, "", "", "", "", total, "", "")
	} else {
		const format = "%v\t%v\t%v\t%v\t%v\n"
		fmt.Fprintf(tw, format, "Cluster", "Namespace", "Label selector", "Kind", "Count")
		fmt.Fprintf(tw, format, "-------", "---------", "--------------", "----", "-----")
		for _, o := range c {
			total += o.Count
			fmt.Fprintf(tw, format, o.Cluster, o.Namespace, o.LabelSelector, o.Kind, o.Count)
		}
		fmt.Fprintf(tw, format, "", "", "", "", "-----")
		fmt.Fprintf(tw, format, "", "", "", "", total)
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

func countServices(clientset *kubernetes.Clientset, namespace string, labelSelector string, timeoutSeconds int64) (int, metav1.Time, metav1.Time, error) {
	services, err := clientset.CoreV1().Services(namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeoutSeconds})
	if err != nil {
		return 0, metav1.Time{}, metav1.Time{}, err
	}

	var items []metav1.ObjectMeta
	for _, item := range services.Items {
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
