# k8s-connect

The tool allows a quick switch between EKS and GKE clusters.

```shell
./k8c
```

It uses a fuzzy search algorithm to match a cluster in a prompt list.


### EKS

The EKS clusters search is enabled by default and can not be disabled.

It reads all AWS profiles from the default config location (`~/.aws/config`) and uses them to find clusters.

Currently, regions are hardcoded in the `aws.go` file.


### GKE

To enable GKE clusters to search the below parameter must be passed:

```shell
./k8c -projects=project1,project2,project3
```

The tool will use the default GCP credential.



### Lint and Build locally

```shell
make lint
make build
```
