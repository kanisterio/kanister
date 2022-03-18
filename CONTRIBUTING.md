# Contributing to Kanister

Welcome, and thank you for considering contributing to Kanister. We welcome all
help in raising issues, improving documentation, fixing bugs, or adding new
features.

If you are interested in contributing, start by reading this document. Please
also take a look at our [code of conduct](CODE_OF_CONDUCT.md).

If you have any questions at all, do not hesitate to reach out to us on
[Slack](kanisterio.slack.com).

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

### Commit messages

The basic idea is that we ask all contributors to practice
[good git commit hygiene](https://www.futurelearn.com/info/blog/telling-stories-with-your-git-history)
to make reviews and retrospection easy. Use your git commits to provide context
for the reviewers, and the folks who will be reading the codebase in the months
and years to come.

Finalized commit messages should look similar to the following format:

```text
Short one line title

An explanation of the problem, providing context, and why the change is being
made.
```

### Submitting Pull Requests

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

Please don't hesitate to reach out to us on [Slack](kanisterio.slack.com) if you
have any questions about contributing!
