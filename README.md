# Kubevali

![](https://img.shields.io/github/workflow/status/darwinia-network/kubevali/CI)
![](https://img.shields.io/github/v/release/darwinia-network/kubevali)

Kubevali (pronounced as *kube-vali*) helps running multiple Darwinia or any substrate-based chain nodes on a Kubernetes cluster.

- [Kubevali](#kubevali)
  - [The Idea](#the-idea)
  - [Features](#features)
  - [Usage](#usage)
    - [Basics](#basics)
    - [Integration with Kubernetes](#integration-with-kubernetes)
    - [Watchlog](#watchlog)
  - [License](#license)

## The Idea

Usually we use a StatefulSet to deploy chain nodes in Kubernetes clusters and every node uses the same CLI arguments. If the StatefulSet has multiple replicas, we're not able to expose the P2P ports (default 30333) to public. Using a HostPort or enabling hostNetwork can cause conflicts when more than one pods are scheduled to a single Kubernetes node. Unlike centralized service, a chain node has to "know" and broadcast the its IP and P2P port, so other nodes can connect to our node. Thus, using a NodePort or LoadBalancer service for individual node is not an option either, as they require specifying `--port` or `--public-addr` for every node.

To allow chain nodes running in Kubernetes clusters not only having outcoming peers but also incoming peers, we developed kubevali. Kubevali runs as a parent process of the node. Before the "real" node process starts, kubevali reads the `kubevali.yaml` config file and generates the CLI arguments, for example `--port` = `statefulset pod index + 30333`. That way, once we create the NodePort services for individual chain node or directly enable hostNetwork, the node will know the correct address, so incoming connections can be established.

![architecture.png](https://i.loli.net/2020/11/26/tYnqjNfsMvQe1hu.png)

Beyond that, kubevali has several other features which may help you managing multiple validator nodes.

## Features

- Configure and run the node with a YAML config file.
- Dynamic node CLI args based on Go template and [sprig](http://masterminds.github.io/sprig/), using environment variables, Kubernetes node external IP, etc.
- Obtain node health status by analyzing logs and report to [healthchecks.io](https://healthchecks.io).

## Usage

Until `v1` released, kubevali is still under development, any config or CLI options may be changed in future versions.

- Releases: <https://github.com/darwinia-network/kubevali/releases>
- Docker images: [Quay.io](https://quay.io/repository/darwinia-network/kubevali?tab=tags)
- Config reference: [./docs/kubevali.yaml](docs/kubevali.yaml)

### Basics

Kubevali uses a YAML file defining the command-line arguments of the node. This is an alternative solution for [paritytech/substrate#6856](https://github.com/paritytech/substrate/issues/6856).

Also, every flags and options will be rendered before it being passed to the node. This allows users launching a group of instances at once, and some of the args (e.g. `--validator`) persist for all nodes, some of the them may be dynamic or sequential (e.g. `--port`, `--ws-port`, `--rpc-port`).

Here is an example minimal config:

```yaml
nodeTemplate:
  # The index of the node
  # Usually the Pod's ordinal index of StatefulSet in Kubernetes
  index: '{{ env "HOSTNAME" | splitList "-" | mustLast }}'

  # CLI command and following arguments for chain node
  command:
    - darwinia
    - --validator
    # more flags here...

  args:
    # If `.nodeTemplate.index` == 2, this generates:
    #  --name '[KUBE-VALI] Development 02'
    name: '[KUBE-VALI] Development {{ printf "%02d" .Index }}'

    # If `.nodeTemplate.index` == 2, this generates:
    #  --port     30335
    #  --rpc-port 9935
    #  --ws-port  9946
    port: '{{ add 30333 .Index }}'
    rpc-port: '{{ add 9933 .Index }}'
    ws-port: '{{ add 9944 .Index }}'
```

Save the config to `./kubevali.yaml` and run:

```bash
$ HOSTNAME=sts-pod-2 kubevali --dry-run

...
INFO Starting node: "darwinia" "--validator" "--name" "[KUBE-VALI] Development 02" "--port" "30335" "--rpc-port" "9935" "--ws-port" "9946"
```

Remove `--dry-run` to launch the node once you confirm that the commands are correct. You can also specify the config file path using `-c, --config PATH`.

### Integration with Kubernetes

There're 2 methods deploying kubevali on Kubernetes.

1. Build you own chain node image and add kubevali into the image.
2. Run kubevali as a `initContainer` first, copy the binary into a `emptyDir` volume, and override the container entrypoint to kubevali by setting Pod `.spec.containers[].command`. Check out [./deploy/manifests](deploy/manifests/) and [./example](example/) for the full example.

### Watchlog

Watchlog is a feature that actively watches and monitors the chain node logs. Kubevali expects a certain `keyword` should appears in logs in a period `lastThreshold`.

Kubevali notify [healthchecks.io](https://healthchecks.io) the node health status, once the `keyword` first time found in node log lines. And start to continuously report the status once per minute. If the time duration since the `keyword` was found is above `lastThreshold`, the node is considered as unhealthy, a `/<check_id>/fail` HTTP request will be sent. Otherwise, it is considered healthy and a usual `/<check_id>` HTTP request will be sent.

See [healthchecks.io docs](https://healthchecks.io/docs/signalling_failures/) for more info.

Here is an example config for validators that are expected to "prepare a block" within every 60 minutes:

```yaml
watchlog:
  # Whether enable watchlog or not, default: false
  enabled: true

  keyword: "Prepared block"
  lastThreshold: 60m

  # The check ID of healthchecks.io. It is allowed to put multiple IDs here
  # and kubevali uses the healthcheckIDs[.nodeTemplate.index].
  healthcheckIDs:
    - check_id_0
    - check_id_1

nodeTemplate:
  index: 0 # This indicates that the first check ID `check_id_0` will be used
```

## License

MIT
