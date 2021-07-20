# Deploy Manifests

The [./manifests](manifests/) directory contains the YAML files for deploying kubevali, [Darwinia](https://github.com/darwinia-network/darwinia) nodes, and the following projects on Kubernetes:

- <https://github.com/darwinia-network/node-liveness-probe>
- <https://github.com/darwinia-network/chain-state-exporter>
- <https://github.com/darwinia-network/snapshot-init-container>

The image tags are defined in [./manifests/kustomization.yaml](manifests/kustomization.yaml), which could be outdated. You can bump them into the newer versions in your kustomization.yaml.

## Example

See [./examples](examples/) for a full example of using the manifests.

To build and apply the example with Kustomize:

```bash
cd example
kustomize build . | kubectl diff -f -
kustomize build . | kubectl apply -f -
```
