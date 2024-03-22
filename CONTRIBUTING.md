# Contributing to Kanister

Welcome, and thank you for considering contributing to Kanister. We welcome all
help in raising issues, improving documentation, fixing bugs, or adding new
features.

If you are interested in contributing, start by reading this document. Please
also take a look at our [code of conduct](CODE_OF_CONDUCT.md).

If you have any questions at all, do not hesitate to reach out to us on
[Slack](https://join.slack.com/t/kanisterio/shared_invite/enQtNzg2MDc4NzA0ODY4LTU1NDU2NDZhYjk3YmE5MWNlZWMwYzk1NjNjOGQ3NjAyMjcxMTIyNTE1YzZlMzgwYmIwNWFkNjU0NGFlMzNjNTk).

We look forward to working together! 🎈

## Developer Certificate of Origin

To contribute to this project, you must agree to the Developer Certificate of
Origin (DCO) for each commit you make. The DCO is a simple statement that you,
as a contributor, have the legal right to make the contribution.

See the [DCO](DCO) file for the full text of what you must agree to.

The most common way to signify your agreement to the DCO is to add a signoff to
the git commit message, using the `-s` flag of the `git commit` command. It will
add a signoff the looks like following line to your commit message:

```txt
Signed-off-by: John Smith <john.smith@example.com>
```

You must use your real name and a reachable email address in the signoff.

Alternately, instead of commits signoff, you can also leave a comment on the PR
with the following statement:

> I agree to the DCO for all the commits in this PR.

Note that this option still requires that your commits are made under your real
name and a reachable email address.

## Contributing Code

### Finding Issues

Generally, pull requests that address and fix existing GitHub issues are assigned
higher priority over those that don't. Use the existing issue labels to help you
identify relevant and interesting issues.

If you think something ought to be fixed but none of the existing issues
sufficiently address the problem, feel free to
[raise a new one](https://github.com/kanisterio/kanister/issues/new/choose).

For new contributors, we encourage you to tackle issues that are labeled as
`good first issues`.

Regardless of your familiarity with this project, documentation help is always
appreciated.

Once you found an issue that interests you, post a comment to the issue, asking
the maintainers to assign it to you.

### Coding Standard

In this project, we adhere to the style and best practices established by the Go
project and its community.

Specifically, this means:

* adhering to guidelines found in the [Effective Go](https://go.dev/doc/effective_go) document
* following the common [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

The [golangci-lint](https://golangci-lint.run/) tool is used to enforce many
styling and safety rules.

### Creating A Local Build

See the [BUILD.md](BUILD.md) document for instructions on how to build, test and
run Kanister locally.

### Updating the API types

If your changes involve the Kanister API types, generate the API documentation using the `make crd_docs` command and push the updated `API.md` file along with any other changes.

### Commit messages

The basic idea is that we ask all contributors to practice
[good git commit hygiene](https://www.freecodecamp.org/news/how-to-write-better-git-commit-messages/)
to make reviews and retrospection easy. Use your git commits to provide context
for the reviewers, and the folks who will be reading the codebase in the months
and years to come.

We're trying to keep all commits in `master` to follow [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/) format.
See [conventions](#commit-conventions) for more info on types and scopes.
We are using squash and merge approach to PRs which means that commit descriptions are generated from PR titles.

It's recommended to use conventional commits when strarting a PR, but follow-up commits in the PR don't have to follow the convention.

### Release notes

If submitted change fixes a bug, introduces a new feature or breaking change, contributor should add a release note.
Kanister is using the [reno](https://docs.openstack.org/reno/latest/) tool to track release notes.

Release note can be added with `make reno-new note=<note_name>` command, which will create a note file.
Contributor should edit and commit the note file.

See [release notes](./releasenotes/README.md) for more info.

### Submitting Pull Requests

**PR titles should be in following format:**

```text
<type>[optional scope]: <description>
```

See [conventions](#commit-conventions) for more info on types and scopes.

When submitting a pull request, it's important that you communicate your intent,
by clearly:

1. describing the problem you are trying to solve with links to the relevant
GitHub issues
1. describing your solution with links to any design documentation and
discussion
1. defining how you test and validate your solution
1. updating the relevant documentation and examples where appropriate

The pull request template is designed to help you convey this information.

In general, smaller pull requests are easier to review and merge than bigger
ones. It's always a good idea to collaborate with the maintainers to determine
how best to break up a big pull request.

Once the maintainers approve your PR, they will label it as `kueue`. The
`mergify` bot will then squash the commits in your PR, and add it to the merge
queue. The bot will auto-merge your work when it's ready.

Congratulations! Your pull request has been successfully merged! 👏

Thank you for reading through our contributing guide to ensure your
contributions are high quality and easy for our community to review and accept. 🤝

Please don't hesitate to reach out to us on [Slack](https://join.slack.com/t/kanisterio/shared_invite/enQtNzg2MDc4NzA0ODY4LTU1NDU2NDZhYjk3YmE5MWNlZWMwYzk1NjNjOGQ3NjAyMjcxMTIyNTE1YzZlMzgwYmIwNWFkNjU0NGFlMzNjNTk). if you
have any questions about contributing!

### Commit conventions

#### Types:

- `feat` - new feature/functionality
	it's recommended to link a GH issue or discussion which describes the feature request or describe it in the commit message
- `fix` - bugfix
	it's recommended to link a GH issue or discussion to describe the bug or describe it in the commit message
- `refactor` - code restructure/refactor which does not affect the (public) behaviour
- `docs` - changes in documentation
- `test` - adding, improving, removing tests
- `build` - changes to build scripts, dockerfiles, ci pipelines
- `deps` - updates to dependencies configuration
- `chore` - none of the above
	use is generally discuraged
- `revert` - revert previous commit

#### Scopes:

There is no strict list of scopes to be used, suggested scopes are:

- `build(ci)` - changes in github workflows
- `build(release)` - changes in release process
- `deps(go)` - dependabot updating go library
- `docs(examples)` - changes in examples
- `docs(readme)` - changes in MD files at the repo root
- `feat(kanctl)` - new functionality for `kanctl` (e.g. new command)
- `refactor(style)` - formatting, adding newlines, etc. in code

#### Breaking changes indicator:

There can be optional `!` after the type and scope to indicate breaking changes

`fix(scope)!: fix with breaking changes`

#### Description:

Short description of WHAT was changed in the commit. SHOULD start with lowercase. MUST NOT have a `.` at the end.
