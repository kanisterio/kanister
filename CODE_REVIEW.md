# Code review requiements

This document describes responsibilities of code reviewers when reviewing PRs.

Pull request process is described in [contributing guide](./CONTRIBUTING.md#submitting-pull-requests)

## Base checklist

- All automated test steps pass (e.g. tests, lints, build)
- PR title follows [commit conventions](CONTRIBUTING.md#commit-conventions)
    - If PR format is different, reviewer should change it to follow the conventions
- PR has a description with reasoning and change overview
    - If description is missing but clear for reviewer, reviewer may request the author to add the description or edit description by themselves
- New feature or fix has tests proving it works
    - Reviewer should request changes from contributor to add tests
- If change in the PR needs documentation
    - Reviewer should request new docs or update to existing docs
    - `/docs` and `/docs_new` need to be kept in sync until we deprecate `/docs`
- If PR introduces breaking changes, fixes a bug or adds a new feature, there should be a [release note](#release-notes)
    - Reviewer may request changes from the contributor to add a release note
    - Reviewer may add a release note by themself in order to unblock the merge process

## Requesting changes

It's recommended to request changes by submitting `comment` type reviews.
`Request changes` type review would block merging until requester approves the
changes, this can slow down the process if there are multiple reviewers.

## Approving and merging

We use `kueue` bot to merge approved PRs.

If PR is approved, all checks are passing and it has the `kueue` label, it will
be automatically squashed and merged.

For PRs from Kanister developers, the author should add the `kueue` label after
PR was approved.

For PRs from community members, the reviewer should add the `kueue` label.

## Release notes

Kanister is using the [reno](https://docs.openstack.org/reno/latest/) tool to generate changelogs from release note files.

To add release note one could run:

```
make reno-new note=<note_name>
```

Note name should be a short description of a change.

File format is described in [reno docs](https://docs.openstack.org/reno/latest/user/usage.html#editing-a-release-note)

Typical examples would be:

```
---
features:
  - Added new functionality doing X
```

Or:

```
---
fixes:
  - Fixed bug with pod output format
upgrade:
  - Make sure custom blueprints follow pod output format spec
```

See [release notes](./releasenotes/README.md) for more info.

