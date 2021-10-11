package main

import (
	"io/ioutil"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Cluster represents configuration of a kubernetes cluster.
type Cluster struct {
	restConfig         *rest.Config
	cluster, namespace string
}

// Clusters gets cluster configurations from kubeconfigs and from within a
// cluster if running inside one.
func Clusters(kubeconfigs []string) ([]Cluster, error) {
	var clusters []Cluster

	for _, kc := range kubeconfigs {
		c, err := fromKubeconfig(kc)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, c)
	}

	c, err := fromInCluster()
	if err != nil && err != rest.ErrNotInCluster {
		return clusters, err
	}
	if err == nil {
		clusters = append(clusters, c)
	}

	return clusters, nil
}

// fromKubeconfig gets cluster configuration from a kubeconfig file.
func fromKubeconfig(kubeconfig string) (Cluster, error) {
	var c Cluster

	apiConfig, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return c, err
	}

	clientConfig := clientcmd.NewDefaultClientConfig(*apiConfig, nil)
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return c, err
	}

	namespace := apiConfig.Contexts[apiConfig.CurrentContext].Namespace
	cluster := apiConfig.Contexts[apiConfig.CurrentContext].Cluster

	c.restConfig = restConfig
	c.namespace = namespace
	c.cluster = cluster

	return c, nil
}

// fromInCluster gets cluster configuration from whithin a cluster.
func fromInCluster() (Cluster, error) {
	var c Cluster

	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return c, err
	}

	ns, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return c, err
	}

	c.restConfig = restConfig
	c.namespace = string(ns)

	return c, nil
}
