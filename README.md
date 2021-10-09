`kcount` counts Kubernetes (K8s) objects across clusters or namespaces. It
gets the cluster or namespace from kubeconfig files or from within a cluster
(when running in a pod).

```
$ go install
$ KUBECONFIGS=$(find ~/.kube/project/* -type f)
$ kcount -l env=prod -k deploy -a $KUBECONFIGS
Cluster                Namespace  Label     Kind    Count  Newest  Oldest
-------                ---------  -----     ----    -----  ------  ------
cluster1.example.com   ns1        env=prod  deploy  34     2d4h    137d
cluster2.example.com   ns1        env=prod  deploy  21     33d     123d
cluster2.example.com   ns1        env=prod  deploy  34     2d4h    110d
```