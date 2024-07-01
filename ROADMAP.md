# Kanister Roadmap

Please join the Kanister community to give feedback on the roadmap
 and open issues with your suggestions.

## Project and [Governance](Governance.md) Work

1. Lifecycle for contributors: Roles, Privs, and lifecycle
   1. project admin?
   2. maintainer (core)
   3. reviewer+approval: PR + branch protections?
   4. contributor
2. Blueprints Maintenance and Support Policy
   1. project test matrix: Kopia vs Restic vs Stow, downstream adopters vs Kanister standalone
   2. community maintained examples: move to a new public repo

## Development Work

### Prioritized (Doing):
1. Fork block storage functions, deprecate unused Kanister code
1. Kopia.io Repository Controller with a CR to control the lifecycle of a Kopia Repository Server
1. Replace [github.com/pkg/errors](http://github.com/pkg/errors) package with a supported fork = https://github.com/kanisterio/kanister/issues/1838
1. Release notes
1. [Adopt OpenSSF Best Practices Badge](https://github.com/kanisterio/kanister/issues/2783)

### Backlog: (Ideas and maintenance, unprioritized)

#### Discussion and Issues to be created/qualified:
1. Track and log events triggered by Blueprint Actions: @PrasadG193 to create
1. Vault integration for Repository Server secrets: @mlavi to create
1. Deprecate Restic, blocked on Kopia work: @e-sumin to create
1. Replace http://gopkg.in/check.v1 with a better test framework
1. Plugability of data mover; S3 and more: GH discussion attempt; add to agenda
1. App mobility: discussion

#### Existing Requests
1. [ARM support](https://github.com/kanisterio/kanister/issues/2254)
1. Generate Kanister controller using KubeBuilder - currently based on Rook operator
  resurrect https://github.com/kanisterio/kanister/issues/193
1. Merge the Repository controller into the Kanister controller.
1. Support for creation of blueprints/actionsets in application namespaces https://github.com/kanisterio/kanister/discussions/2922
