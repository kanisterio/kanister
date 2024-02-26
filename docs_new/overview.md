# Overview

|[![image](https://goreportcard.com/badge/github.com/kanisterio/kanister)](<https://goreportcard.com/report/github.com/kanisterio/kanister>)|[![image](https://github.com/kanisterio/kanister/actions/workflows/main.yaml/badge.svg?branch=master)](<https://github.com/kanisterio/kanister/actions>)|
|-------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------|

## Design Goals

The design of Kanister was driven by the following main goals:

1. **Application-Centric:** Given the increasingly complex and
    distributed nature of cloud-native data services, there is a growing
    need for data management tasks to be at the *application* level.
    Experts who possess domain knowledge of a specific application\'s
    needs should be able to capture these needs when performing data
    operations on that application.
2. **API Driven:** Data management tasks for each specific application
    may vary widely, and these tasks should be encapsulated by a
    well-defined API so as to provide a uniform data management
    experience. Each application expert can provide an
    application-specific pluggable implementation that satisfies this
    API, thus enabling a homogeneous data management experience of
    diverse and evolving data services.
3. **Extensible:** Any data management solution capable of managing a
    diverse set of applications must be flexible enough to capture the
    needs of custom data services running in a variety of environments.
    Such flexibility can only be provided if the solution itself can
    easily be extended.

## Getting Started

Follow the instructions in the [Installation](install.md) section to get
section to get Kanister up and running on your Kubernetes cluster. Then
see Kanister in action by going through the walkthrough under
[Tutorial](tutorial.md).

The [Architecture](architecture.md) section provides
architectural insights into how things work. We recommend that you take
a look at it.
