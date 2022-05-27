# kcount

Sometimes I need to get the number of objects inside Kubernetes cluster(s). Or
I need to compare two or more clusters to see the number and age of the objects
that get replicated.

`kcount` counts Kubernetes objects across namespaces and clusters. It can be
used as CLI tool or as daemon (service) exposing Prometheus metrics.

## CLI tool

Count all (supported) kinds of objects in all namespaces. Use cluster
configuration from `KUBECONFIG` environment variable or `$HOME/.kube/config`.

```
$ kcount -A
Cluster               Namespace  Label  Kind        Count
-------               ---------  -----  ----        -----
cluster1.example.com  <All>             configmap   2735
cluster1.example.com  <All>             pod         551
cluster1.example.com  <All>             secret      360
cluster1.example.com  <All>             service     116
cluster1.example.com  <All>             deployment  78
cluster1.example.com  <All>             ingress     39
                                                    -----
                                                    3879
```

Count pods and ingresses (and show age info) with a given label across multiple
clusters.

```
$ kcount -k pod,ingress -a -l env=prod $HOME/.kube/project/*/*
Cluster                Namespace  Label     Kind     Count  Newest  Oldest
-------                ---------  -----     ----     -----  ------  ------
cluster1.example.com   ns1        env=prod  pod      68     1d4h    37d
cluster2.example.com   ns1        env=prod  pod      68     1d4h    10d
cluster3.example.com   ns1        env=prod  pod      42     23d     23d
cluster1.example.com   ns1        env=prod  ingress  34     2d4h    137d
cluster2.example.com   ns1        env=prod  ingress  34     2d4h    110d
cluster3.example.com   ns1        env=prod  ingress  21     33d     123d
                                                     -----
                                                     267
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
