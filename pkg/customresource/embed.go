package customresource

import "embed"

// embed.go embeds the CRD yamls (actionset, profile, blueprint) with the
// controller binary so that we can read these manifests in runtime.

// We need these manifests at two places, at `pkg/customresource/` and at
// `helm/kanister-operator/crds`. To make sure we are not duplicating the
// things we have original files at `pkg/customresource` and have soft links
// at `helm/kanister-operator/crds`.

//go:embed actionset.yaml
//go:embed blueprint.yaml
//go:embed profile.yaml
//go:embed repositoryserver.yaml
var yamls embed.FS
