# Example of Kubevali + Kubernetes

This example deploys:

- kubevali (this repo)
- darwinia node (<https://github.com/darwinia-network/darwinia>)
- node-liveness-probe (<https://github.com/darwinia-network/node-liveness-probe>)

The manifests are in [../deploy/manifests](../deploy/manifests/).

To build and apply the manifests with Kustomize:

```bash
cd example
kustomize build . | kubectl diff -f -
kustomize build . | kubectl apply -f -
```
