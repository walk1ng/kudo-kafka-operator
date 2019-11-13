# Run KUDO-Kafka tests

## Requirements

- A running Kubernetes cluster with `$KUBECONFIG` pointing to correct configuration
- KUDO controller running in the Kubernetes cluster
- `kubectl` installed, with `$KUBECTL_PATH` pointing to it 
- The https://github.com/kudobuilder/operators repository checked out (as a best practice, that should be inside of your $GOPATH)

## Running tests

Export `DS_KUDO_VERSION`, `KUBECONFIG` and `KUBECTL_PATH`, and run the `run-tests.sh` script.

Example:

```bash
export DS_KUDO_VERSION=v0.5.0
export KUBECTL_PATH=/usr/local/bin/kubectl
export KUBECONFIG=${HOME}/.kubeconfig/config
./run-tests.sh /path/to/kudobuilder/operators
```

## Development

### Submodules

This repository includes some tooling shared by various KUDO operators, as a
git submodule from the private data-services-kudo repository.

https://git-scm.com/book/en/v2/Git-Tools-Submodules contains some useful
information about how to work with submodules.

### Go modules
In this project we always aim to target one version of Kubernetes API and use `replace` for kubernetes client/api modules. 

In case for kubernetes 1.16.2, it would be 
```
curl -s https://proxy.golang.org/k8s.io/api/@v/kubernetes-1.16.2.info | jq -r .Version
curl -s https://proxy.golang.org/k8s.io/apimachinery/@v/kubernetes-1.16.2.info | jq -r .Version
curl -s https://proxy.golang.org/k8s.io/client-go/@v/kubernetes-1.16.2.info | jq -r .Version
```

with next output:
```
v0.0.0-20191016110408-35e52d86657a
v0.0.0-20191004115801-a2eda9f80ab8
v0.0.0-20191016111102-bec269661e48
```

and we can use them in the `go.mod`

```
replace k8s.io/api => k8s.io/api v0.0.0-20191016110408-35e52d86657a
replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004115801-a2eda9f80ab8
replace k8s.io/client-go => k8s.io/client-go v0.0.0-20191016111102-bec269661e48
```
