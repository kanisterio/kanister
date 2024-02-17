# Installation {#install}

Kanister can be easily installed and managed with
[Helm](https://helm.sh). You will need to configure your `kubectl` CLI
tool to target the Kubernetes cluster you want to install Kanister on.

Start by adding the Kanister repository to your local setup:

``` bash
helm repo add kanister <https://charts.kanister.io/>
```

Use the `helm install` command to install Kanister in the `kanister`
namespace:

``` bash
helm -n kanister upgrade \--install kanister \--create-namespace
kanister/kanister-operator
```

Confirm that the Kanister workloads are ready:

``` bash
kubectl -n kanister get po
```

You should see the operator pod in the `Running` state:

``` bash
NAME READY STATUS RESTARTS AGE
kanister-kanister-operator-85c747bfb8-dmqnj 1/1 Running 0 15s
```

::: tip NOTE

Kanister is guaranteed to work with the 3 most recent versions of
Kubernetes. For example, if the latest version of Kubernetes is 1.24,
Kanister will work with 1.24, 1.23, and 1.22. Support for older versions
is provided on a best-effort basis. If you are using an older version of
Kubernetes, please consider upgrading to a newer version.
:::

## Configuring Kanister

Use the `helm show values` command to list the configurable options:

``` bash
helm show values kanister/kanister-operator
```

For example, you can use the `image.tag` value to specify the Kanister
version to install.

The source of the `values.yaml` file can be found on
[GitHub](https://github.com/kanisterio/kanister/blob/master/helm/kanister-operator/values.yaml).

## Managing Custom Resource Definitions (CRDs)

The default RBAC settings in the Helm chart permit Kanister to manage
and auto-update its own custom resource definitions, to ease the user\'s
operation burden. If your setup requires the removal of these settings,
you will have to install Kanister with the
`--set controller.updateCRDs=false` option:

``` bash
helm -n kanister upgade \--install kanister \--create-namespace
kanister/kanister-operator \--set controller.updateCRDs=false
```

This option lets Helm manage the CRD resources.

## Using custom certificates with the Validating Webhook Controller

Kanister installation also creates a validating admission webhook server
that is invoked each time a new Blueprint is created.

By default the Helm chart is configured to automatically generate a
self-signed certificates for Admission Webhook Server. If your setup
requires custom certificates to be configured, you will have to install
kanister with `--set bpValidatingWebhook.tls.mode=custom` option along
with other certificate details.

Create a Secret that stores the TLS key and certificate for webhook
admission server:

``` bash
kubectl create secret tls my-tls-secret \--cert /path/to/tls.crt \--key
/path/to/tls.key -n kanister
```

Install Kanister, providing the PEM-encoded CA bundle and the
[tls]{.title-ref} secret name like below:

``` bash
helm upgrade \--install kanister kanister/kanister-operator \--namespace
kanister \--create-namespace \--set bpValidatingWebhook.tls.mode=custom
\--set bpValidatingWebhook.tls.caBundle=\$(cat /path/to/ca.pem \| base64
-w 0) \--set bpValidatingWebhook.tls.secretName=tls-secret
```

## Building and Deploying from Source

Follow the instructions in the `BUILD.md` file in the [Kanister GitHub
repository](https://github.com/kanisterio/kanister/blob/master/BUILD.md)
to build Kanister from source code.
