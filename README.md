# kcount

`kcount` counts Kubernetes (K8s) objects across clusters. It gets the cluster
configuration, including cluster name and namespace, from kubeconfig files or
from within a cluster (when running in a pod).

```
$ kcount -l env=prod -k deploy -a ~/.kube/project/*/*
Cluster                Namespace  Label     Kind    Count  Newest  Oldest
-------                ---------  -----     ----    -----  ------  ------
cluster1.example.com   ns1        env=prod  deploy  34     2d4h    137d
cluster2.example.com   ns1        env=prod  deploy  21     33d     123d
cluster2.example.com   ns1        env=prod  deploy  34     2d4h    110d
```

# Installation

```
git clone git@github.com:pan-net-security/kcount.git
cd kcount
go install
```

or download a [release](https://github.com/pan-net-security/kcount/releases)
binary for your system and architecture.