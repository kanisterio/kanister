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
3. Move the Kanister.io website to GitHub Repo
4. Leverage GitHub Issues and Projects for planning

## Development Work

### Prioritized (Doing):

1. Fork block storage functions, deprecate unused Kanister code
2. Kopia.io Repository Controller with a CR to control the lifecycle of a Kopia Repository Server
3. [Progress tracking for individual Phases in an Action](https://github.com/kanisterio/kanister/blob/master/design/progress-tracking.md)
4. ActionSet metrics
5. Container image vulnerability scanning

### New Features:

1. Track and log events triggered by Blueprint Actions
2. ARM support
3. Use GitHub pages with Jekyll or Hugo for documentation
4. Vault integration for Repository Server secrets

### Backlog: (ideas and maintenance)

1. Deprecate Restic
2. Generate Kanister controller using KubeBuilder - current one is based on rook operator
3. Merge the Repository controller into the Kanister controller after 2.
4. Replace [github.com/pkg/errors](http://github.com/pkg/errors) package with a supported fork
5. Replace http://gopkg.in/check.v1 with a better test framework
6. Release notes
7. Adopt license scanning tool and OpenSSF Best Practices Badge
