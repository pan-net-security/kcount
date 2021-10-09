`kcount` counts Kubernetes objects in clusters or namespaces. It gets the
cluster or namespace from kubeconfig files or from within a cluster (when
running in a pod).

```
$ go install
$ KUBECONFIGS=$(find ~/.kube/project/* -type f)
$ kcount -l env=prod -k deploy $KUBECONFIGS
Cluster               Namespace Label Selector  Kind     Count
-------               --------- --------------  ----     -----
cluster1.example.com  ns1       env=prod        deploy   34
cluster2.example.com  ns1       env=prod        deploy   21
cluster2.example.com  ns1       env=prod        deploy   34
```