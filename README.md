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
