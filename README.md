# kubeselect

Visually switch between configured Kubernetes Contexts in your KUBECONFIG on command line. Do not use any namespace switching tools as this would alter your configured contexts and thus make kubeselect useless.

## Install

```sh
go install github.com/sbreitf1/kubeselect@latest
```

## Usage

```sh
# show selection for all configured contexts
kubeselect

# create contexts for all namespaces for clusters that are referenced by existing contexts
kubeselect -u
```

Use arrow keys up and down to navigate to another context, press enter to switch to the highlighted context. Currently selected context is marked as yellow. Press escape to exit without changing the selected context.

Arrow key left will select the parent cluster or collapse the selected one. Arrow key right will select the last context in the selected cluster or expand the selected cluster node. Type letters to select a context with given name in the expanded cluster node.

# Comparison

- [kubeswitch](https://github.com/danielb42/kubeswitch): also tree-view based selection, but with mandatory K8s integration that requires some privileges on configured clusters.
- [kubectx](https://github.com/ahmetb/kubectx): includes more functionality but does not offer visual selection.
- [kpick](https://github.com/dcaiafa/kpick): same functionality but drops all unknown configuration values in your KUBECONFIG and thus might break it
