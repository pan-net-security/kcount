package main

import (
	"io/ioutil"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Config represents kubernetes cluster configuration obtained from within a
// cluster or from a kubeconfig file.
type Config struct {
	restConfig         *rest.Config
	cluster, namespace string
}

func getConfigs(kubeconfigs []string) ([]Config, error) {
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

func getConfigsFromKubeconfigs(kubeconfigs []string) ([]Config, error) {
	var configs []Config

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

		configs = append(configs, Config{restConfig: restConfig, namespace: namespace, cluster: cluster})
	}

	return configs, nil
}

func getConfigFromCluster() (Config, error) {
	c := Config{}
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
