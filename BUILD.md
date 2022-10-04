# Development Guide

This document provides instructions on how to build and run Kanister locally.

## Architecture

Kanister is a data management framework written in Go. It allows users to
express data protection workflows using blueprints and actionsets. These
resources are defined as Kubernetes
[Custom Resource Definitions](https://docs.kanister.io/architecture.html#custom-resources)
, following the operator pattern.

[![kanister workflow](./graphic/kanister_workflow.png)](https://docs.kanister.io/architecture.html)

## Repository Layout

* `build` - A collection of shell scripts used by the Makefile targets to build,
test and package Kanister
* `cmd` - Go `main` packages containing the source of the `controller`,
`kanctl` and `kando` executables
* `docker` - A collection of Dockerfiles for build and demos
* `docs` - Source of the documentation at docs.kanister.io
* `examples` - A collection of example blueprints to show how Kanister works
with different data services
* `graphic` - Image files used in documentation
* `helm` - Helm chart for the Kanister operator
* `pkg` - Go library packages used by Kanister

## Development

The [Makefile](Makefile) provides a set of targets to help simplify the build
tasks. To ensure cross-platform consistency, many of these targets use Docker
to spawn build containers based on the `ghcr.io/kanisterio/build` public image.

For `make test` to succeed, a valid kubeconfig file must be found at 
`$HOME/.kube/config`. See the Docker command that runs make test [here](https://github.com/kanisterio/kanister/blob/fa04d77eb6f5c92521d1413ddded385168f39f42/Makefile#L219).

Use the `check` target to ensure your development environment has the necessary
development tools:

```sh
make check
```

The following targets can be used to lint, test and build the Kanister
controller:
```sh
make golint

make test

make build-controller
```

To build the controller OCI image:
```sh
make release-controller \
  IMAGE=<your_registry>/<your_controller_image> \
  VERSION=<your_image_tag>
```
If `VERSION` is not specified, the Makefile will auto-generate one for you.

You can test your Kanister controller locally by using Helm to deploy the local
Helm chart:
```sh
helm install kanister ./helm/kanister-operator \
  --create-namespace \
  --namespace kanister \
  --set image.repository=<your_registry>/<your_controller_image> \
  --set image.tag=<your_image_tag>
```

Subsequent changes to your Kanister controller can be applied using the `helm
upgrade` command:

```sh
helm upgrade kanister ./helm/kanister-operator \
  --namespace kanister \
  --set image.repository=<your_registry>/<your_controller_image> \
  --set image.tag=<your_new_image_tag>
```

### Non-Docker Setup

Most of the Makefile targets can work in a non-Docker development setup, by
setting the `DOCKER_BUILD` variable to `false`.

## Documentation

The source of the documentation is found in the `docs` folder. They are written
in the [reStructuredText](https://docutils.sourceforge.io/rst.html) format.

To rebuild the documentation:

```sh
make docs
```

The `docs` target uses the `ghcr.io/kanisterio/docker-sphinx` public image to
generate the HTML documents and store them in your local `/docs/_build/html`
folder.

## New Blueprints

If you have new blueprints that you think will benefit the community, feel free
to add them to the `examples` folder via pull requests. Use the existing folder
layout as a guide. Be sure to include a comprehensive README.md to demonstrate
end-to-end usage.

## New Kanister Functions

Kanister can be extended with custom Kanister Functions. All the functions are
written in Go. They are located in the `pkg/function` folder.

Take a look at this [PR](https://github.com/kanisterio/kanister/pull/1282) to
see how to write a new Kanister Function.

Don't forget to update the documentation at `docs/functions.rst` with
configuration information and examples to show off your new function.
