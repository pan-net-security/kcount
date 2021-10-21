# kcount

`kcount` counts Kubernetes (K8s) objects across clusters. It gets the cluster
configuration, including cluster name and namespace, from kubeconfig files or
from within a cluster (when running in a pod).

It can be used as CLI tool or as daemon (service) exposing Prometheus metrics.

## CLI tool

```
$ kcount -l env=prod -k ingress,pod -a ~/.kube/project/*/*
Cluster                Namespace  Label selector  Kind     Count  Newest  Oldest
-------                ---------  --------------  ----     -----  ------  ------
cluster1.example.com   ns1        env=prod        ingress  34     2d4h    137d
cluster2.example.com   ns1        env=prod        ingress  21     33d     123d
cluster2.example.com   ns1        env=prod        ingress  34     2d4h    110d
cluster1.example.com   ns1        env=prod        pod      68     1d4h    37d
cluster2.example.com   ns1        env=prod        pod      42     23d     23d
cluster2.example.com   ns1        env=prod        pod      68     1d4h    10d
```

Installation

```
git clone git@github.com:pan-net-security/kcount.git
cd kcount
go install
```

or download a [release](https://github.com/pan-net-security/kcount/releases)
binary for your system and architecture.

## Kubernetes service

```
# Deploy kcount service and deployment
kubectl apply -f k8s.yaml

# Check Prometheus metrics
kubectl run alpine --image=alpine --rm -it --restart=Never --command -- \
  wget -O- kcount/metrics --timeout 5 | grep objects_
```
