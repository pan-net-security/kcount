# kcount

`kcount` counts Kubernetes objects across clusters or namespaces. It gets the
cluster configuration, including cluster name and namespace, from `KUBECONFIG`,
supplied kubeconfig file(s) or from within a cluster (when running in a pod).

It can be used as CLI tool or as daemon (service) exposing Prometheus metrics.

## CLI tool

Count objects across multiple clusters using kubeconfig files supplied as
command line arguments:

```
$ kcount -a -l env=prod -k ingress,pod ~/.kube/project/*/*
Cluster                Namespace  Label selector  Kind     Count  Newest  Oldest
-------                ---------  --------------  ----     -----  ------  ------
cluster1.example.com   ns1        env=prod        pod      68     1d4h    37d
cluster2.example.com   ns1        env=prod        pod      68     1d4h    10d
cluster3.example.com   ns1        env=prod        pod      42     23d     23d
cluster1.example.com   ns1        env=prod        ingress  34     2d4h    137d
cluster2.example.com   ns1        env=prod        ingress  34     2d4h    110d
cluster3.example.com   ns1        env=prod        ingress  21     33d     123d
                                                           -----
                                                           267
```

Count objects in all namespaces using KUBECONFIG environment variable:

```
$ kcount -A -k deployment,pod,configmap,secret,ingress
Cluster               Namespace  Label selector  Kind        Count
-------               ---------  --------------  ----        -----
cluster1.example.com                             configmap   2736
cluster1.example.com                             pod         499
cluster1.example.com                             secret      358
cluster1.example.com                             deployment  78
cluster1.example.com                             ingress     40
                                                             -----
                                                             3711
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
kubectl apply -f k8s-example.yaml

# Check Prometheus metrics
kubectl run alpine --image=alpine --rm -it --restart=Never --command -- \
wget -O- kcount/metrics --timeout 5 | grep objects_
```
