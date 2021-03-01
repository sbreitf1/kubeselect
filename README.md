# kubeselect

Visually switch between configured Kubernetes Contexts in your KUBECONFIG on command line.

## Install

```sh
GO111MODULE=on go get github.com/sbreitf1/kubeselect
```

## Usage

```sh
kubeselect
```

Use arrow keys to navigate to another context, press enter to switch to the highlighted context. Currently selected context is marked as yellow.

# Comparison

- [kubeswitch](https://github.com/danielb42/kubeswitch): also tree-view based selection, but with K8s integration that requires some privileges on configured clusters.
- [kubectx](https://github.com/ahmetb/kubectx): includes more functionality but does not offer visual selection.
- [kpick](https://github.com/dcaiafa/kpick): same functionality but drops all unknown configuration values in your KUBECONFIG and thus might break it