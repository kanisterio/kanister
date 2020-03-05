# CGO Build Container

This build container supports glibc based CGO executables using an ubuntu based container.

## Build the container
```
docker build -f ./Dockerfile -t kanisterio/cgo-build:v1 .
```
Then push the image to a suitable repo.

Update the `CGO_BUILD_IMAGE` value in the top level Makefile if you update the version.
