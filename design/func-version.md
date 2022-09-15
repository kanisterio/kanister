# Kanister Functions Version Management

<!-- toc -->
- [Problem Statement](#problem-statement)
- [Proposed Solution](#proposed-solution)
  - [Summary](#summary)
  - [Assumptions And Constraints](#assumptions-and-constraints)
  - [Function Versioning](#function-versioning)
  - [Function Registration And Discovery](#function-registration-and-discovery)
  - [User Experience](#user-experience)
  - [Version Promotion And Deprecation](#version-promotion-and-deprecation)
  - [Versions Discovery](#versions-discovery)
- [Test Cases](#test-cases)
<!-- /toc -->

## Problem Statement

As Kanister continues to grow and evolve to meet the needs of its community,
it's inevitable that eventually, some of the existing built-in [Kanister
Functions][0] will be updated with breaking changes, deprecated or even
removed entirely.

The lack of a proper strategy to manage and rollout such changes in a
responsible backwards compatible manner can lead to costly operational overhead
for Kanister users. In addition, it also becomes difficult for Kanister
maintainers to assess the blast radius of the change, determine the rollout
timing, provide timely support etc.

This proposal outlines a version management scheme that can be used to manage
the lifecycle of Kanister Functions as they evolve from inception to retirement.

## Proposed Solution

### Summary

The proposed solution uses a version management scheme to communicate the
stability, availability, promotion and deprecation of Kanister Functions.

The design extends the existing implementation of the action-level
[`PreferredVersion` mechanism][1] to the phase-level by enabling users to
specify the desired  Kanister Function version within the blueprint. It also
proposes changes to the relevant interfaces with code examples demonstrating the
implementation of multiple versions of a Kanister Function.

### Assumptions And Constraints

* The proposed solution is compatible with existing downstream custom Kanister
Function implementation, without any code change.

### Function Versioning

The proposed solution introduces a new Go interface named `VersionedFunc`. The
`VersionedFunc` type embeds the `Func` type to include information about the
multiple versions of implementation of the underlying Kanister Function.

```go
// VersionedFunc extends the Func type to include information about the multiple
// versions of the underlying Kanister Function.
type VersionedFunc interface {
  // Func is the v0.0.0 version of the function. For existing functions, this
  // maps to their existing `Func` implementation.
  Func

  // StableVersion returns the stable version of the function.
  StableVersion() string

  // Versions maps the different versions of the function to their respective
  // implementations.
  Versions() map[string]Func
}
```

By embedding the existing `Func` type in the new `VersionedFunc` type, existing
Kanister Functions will continue to work as-is. Functions maintainers have the
flexibility to decide when/if they need to extend existing Function from being a
`Func` type to a `VersionedFunc` type.

As an example, if new versions of the `KubeExec` Function are introduced, the
existing [`kubeExecFunc` struct][2] will be updated to implement the
`VersionedFunc` interface:

```go
func init() {
  _ = kanister.RegisterVersionedFunc(&kubeExecFunc{})
}

var  _ kanister.VersionedFunc = (*kubeExecFunc)(nil)

func (*kubeExecFunc) StableVersion() string {
  return "v0.1.0"
}

func (*kubeExecFunc) Versions() map[string]kanister.Func {
  return map[string]kanister.Func{
    kanister.DefaultVersion: &kubeExecFunc{},
    "v0.1.0": &kubeExecFuncV010{
      parent: &kubeExecFunc{},
    },
    "v0.2.0": &kubeExecFuncV020{
      parent: &kubeExecFunc{},
    },
  }
}
```

The above code snippet shows the introduction of versions `v0.1.0` and `v0.2.0`
of the `KubeExec` Function, by adding the corresponding new structs to the
`Versions()` method. Callers can use the `StableVersion()` method to determine
the stable version of `KubeExec`.

> 📝 The stable version is the default version used in the blueprint, if no user
> override is provided.

The implementation of the new two versions are added to two new files named,
`pkg/function/kube_exec_v010.go` and `pkg/function/kube_exec_v020.go`. E.g., the
code for `v0.1.0` will look something like this:

```go
package function

import (
	"context"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/param"
)

type kubeExecFuncV010 struct {
	parent kanister.Func
}

func (f *kubeExecFuncV010) Name() string {
	return f.parent.Name
}

func (f *kubeExecFuncV010) RequiredArgs() []string {
	return f.parent.RequiredArgs()
}

func (f *kubeExecFuncV010) Arguments() []string {
	return f.parent.Arguments()
}

func (kef *kubeExecFuncV010) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
```

> 👀 The `kubeExecFunc010` struct holds an optional reference to the existing
> v0.0.0 implementation so that it can re-use the original function name and
> list of arguments.

### Function Registration And Discovery

The new registration functions for the `VersionedFunc` type look like this:

```go
// RegisterVersionedFunc registers the provided versioned function with the
// global function map.
func RegisterVersionedFunc(f VersionedFunc) error {
  for v, fn := range f.Versions() {
    if f == nil {
      return errors.Errorf("kanister: Cannot register nil function")
    }

    registerFunc(fn, v)
  }
  return nil
}

func registerFunc(fn Func, version string) error {
  funcMu.Lock()
  defer funcMu.Unlock()

  parsed := *semver.MustParse(version)
  if _, ok := funcs[fn.Name()]; !ok {
    funcs[fn.Name()] = make(map[semver.Version]Func)
  }
  funcs[fn.Name()][parsed] = fn

  return nil
}
```

The [`kanister.GetPhases()` function][3] is updated to use a new function named
`findFunction()` to find the correct version of the Kanister Function to
execute:

```diff
diff --git a/pkg/phase.go b/pkg/phase.go
index d6519fde..9df1901f 100644
--- a/pkg/phase.go
+++ b/pkg/phase.go
@@ -16,6 +16,7 @@ package kanister

 import (
 	"context"
+	"fmt"

 	"github.com/Masterminds/semver"
 	"github.com/pkg/errors"
@@ -120,6 +121,33 @@ func GetDeferPhase(bp crv1alpha1.Blueprint, action, version string, tp param.Tem
 	}, nil
 }

+func findFunction(name, version string) (Func, error) {
+	funcMu.RLock()
+	defer funcMu.RUnlock()
+
+	if _, ok := funcs[name]; !ok {
+		return nil, errors.Errorf("Requested function {%s} has not been registered", name)
+	}
+
+	parsed, err := semver.NewVersion(version)
+	if err != nil {
+		return nil, err
+	}
+
+	for registeredVersion, fn := range funcs[name] {
+		if registeredVersion == *parsed {
+			return fn, nil
+		}
+	}
+
+	defaultVersion := semver.MustParse(DefaultVersion)
+	if fn, ok := funcs[name][*defaultVersion]; ok {
+		return fn, nil
+	}
+
+	return nil, fmt.Errorf("function version not found: %s", version)
+}
+
 func regFuncVersion(f, version string) (semver.Version, error) {
 	funcMu.RLock()
 	defer funcMu.RUnlock()
@@ -148,7 +176,7 @@ func regFuncVersion(f, version string) (semver.Version, error) {
 }

 // GetPhases renders the returns a list of Phases with pre-rendered arguments.
-func GetPhases(bp crv1alpha1.Blueprint, action, version string, tp param.TemplateParams) ([]*Phase, error) {
+func GetPhases(bp crv1alpha1.Blueprint, action, preferredVersion string, tp param.TemplateParams) ([]*Phase, error) {
 	a, ok := bp.Actions[action]
 	if !ok {
 		return nil, errors.Errorf("Action {%s} not found in action map", action)
@@ -157,7 +185,12 @@ func GetPhases(bp crv1alpha1.Blueprint, action, version string, tp param.Templat
 	phases := make([]*Phase, 0, len(a.Phases))
 	// Check that all requested phases are registered and render object refs
 	for _, p := range a.Phases {
-		regVersion, err := regFuncVersion(p.Func, version)
+		funcVersion := preferredVersion
+		if p.FuncVersion != "" { // use user overrides if provided
+			funcVersion = p.FuncVersion
+		}
+
+		fn, err := findFunction(p.Func, funcVersion)
 		if err != nil {
 			return nil, err
 		}
@@ -169,7 +202,7 @@ func GetPhases(bp crv1alpha1.Blueprint, action, version string, tp param.Templat
 		phases = append(phases, &Phase{
 			name:    p.Name,
 			objects: objs,
-			f:       funcs[p.Func][regVersion],
+			f:       fn,
 		})
 	}
 	return phases, nil
```

If the Kanister Function doesn't implement the `VersionedFunc` interface, the
`findFunction()` function will fall back to using the existing mechanism to use
the default `v0.0.0` version.

### User Experience

The `BlueprintPhase` API will be updated to include a new `FuncVersion` property
to allow user overrides the Kanister Function version for a phase in a blueprint
:

```diff
diff --git a/pkg/apis/cr/v1alpha1/types.go b/pkg/apis/cr/v1alpha1/types.go
index d0a26517..b25eb7df 100644
--- a/pkg/apis/cr/v1alpha1/types.go
+++ b/pkg/apis/cr/v1alpha1/types.go
@@ -213,10 +213,11 @@ type BlueprintAction struct {

 // BlueprintPhase is a an individual unit of execution.
 type BlueprintPhase struct {
-	Func       string                     `json:"func"`
-	Name       string                     `json:"name"`
-	ObjectRefs map[string]ObjectReference `json:"objects,omitempty"`
-	Args       map[string]interface{}     `json:"args"`
+	Func        string                     `json:"func"`
+	FuncVersion string                     `json:"funcVersion,omitempty"`
+	Name        string                     `json:"name"`
+	ObjectRefs  map[string]ObjectReference `json:"objects,omitempty"`
+	Args        map[string]interface{}     `json:"args"`
 }

 // +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
```

An example blueprint with an action that uses this new field looks like this:

```yaml
actions:
  tail:
    phases:
    - func: KubeExec
      funcVersion: v0.1.0
      name: tail
      args:
        namespace: default
        pod: timer
        container: timer
        command:
        - sh
        - "-c"
        - |
         /bin/tail_datetime.sh
```

If the `funcVersion` property isn't specified in the blueprint action, Kanister
will determine the stable version using the `VersionedFunc` interface. If the
Kanister Function doesn't implement the `VersionedFunc` interface, then Kanister
will fall back to using the existing mechanism to use the default `v0.0.0`
version of the Function.

### Version Promotion And Deprecation

The timeframe to promote a version to the stable state is based on the
implementers' discretion. It is expected that there will be a sufficient time
window between the introduction of a new version and its promotion to the stable
state, to allow for community feedback and bug fixes.

To use the new non-stable version, users will have to specify the `funcVersion`
property in their blueprint phase, as shown in the example above. Otherwise, the
blueprint will continue to use the existing stable (default) version.

Once a new version is promoted to stable, the existing stable version will be
demoted with a deprecation time frame of at least two releases. During this
period, log and event warnings will be generated in Kanister's logs and the
actionset `status` as indication that the in-use version has been deprecated,
and will be removed in a future release. Users are responsible for updating
their blueprints to work with the new stable version.

After the deprecation time frame, the deprecated version will officially fall
out of support. Any usage of this version will cause the controller to return an
error and halt the execution. The version number will still be preserved for
posterity.

### Versions Discovery

The versions of all Kanister Functions can be stored in a version YAML file
that maps the Function name to the supported versions. Special annotation will
be used to denote all stable versions. This file can be [embedded][4] in
the Kanister controller binary during build. The controller uses this file to
determine the stable version of each Function, whenever it needs that
information.

To facilitate easy version discovery, the controller exposes an endpoint that
publishes its supported Kanister Functions versions.

## Test Cases

New unit tests will be added to verify the correctness of the registration of
multiple versions of a Kanister Function.

New test blueprints will be added to test the user-specified Kanister Function
versions.

Existing integration and e2e tests should continue to pass.

[0]: https://docs.kanister.io/functions.html
[1]: https://github.com/kanisterio/kanister/blob/56f82a2d2361556ebd3fe313a7abf771e8d49d0e/pkg/apis/cr/v1alpha1/types.go#L105-L107
[2]: https://github.com/kanisterio/kanister/blob/6b6354026b8e8961fc69d8d977bbdbc6d76ae0d5/pkg/function/kube_exec.go#L47
[3]: https://github.com/kanisterio/kanister/blob/6b6354026b8e8961fc69d8d977bbdbc6d76ae0d5/pkg/phase.go#L151
[4]: https://pkg.go.dev/embed
