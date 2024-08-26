## Automated Release

Release process:

- Verify the release version by looking at the [releases page](https://github.com/kanisterio/kanister/releases) on the Kanister repo.
- Trigger the [pre-release](#pre-release-workflow) workflow with the desired version number (e.g. bump the minor version portion 0.106.0 -> 0.107.0), which will result in a PR getting created in Kanister repo
- Review and validate created PR that it doesn't have any unintended changes
- Make sure to validate that all merged PRs in the release have [release notes](#release-notes)
	- Make sure that CHANGELOG.md and CHANGELOG_CURRENT.md contain release notes for the release version
	- **NOTE** While we establish the new process of release notes, it may be required to add notes in pre-release step by commiting them into pre-release branch
- Approve and merge the pre-release PR (it will be merged by `kueue` when approved)
- Merging of pre-release PR will trigger the `release.yaml` pipeline, which will create a github release and publish the images
- The Kanister release job will publish a new tag, update documentation, build all the docker images, and push them to the [ghcr.io](https://github.com/orgs/kanisterio/packages) registry
- Post release announcement in kanister slack and https://groups.google.com/g/kanisterio

### Pre-release workflow

`pre-release` workflow serves to create a new version of kanister, and updates the older version number in source files of kanister repo with new version and creates a PR in kanister repo.

The workflow can be triggered using workflow dispatch from the `Actions` tab in the repo: https://github.com/kanisterio/kanister/actions/workflows/pre-release.yml
It has a required input of `release_tag`, which should be set to a version next to the current version.

This will result in creating a PR with a version bump, like https://github.com/kanisterio/kanister/pull/2629
**NOTE** PR description would look like `pre-release: Update version to ...`

This PR would update the older Kanister version reference in various files.

Pre-release PR should contain updates to CHANGELOG.md and CHANGELOG_CURRENT.md, which will be auto-generated
on workflow run.

**IMPORTANT** Reviewer of the pre-release PR should check if there are changelog items (release notes) for merged PRs and may add them in the pre-release PR if necessary.

After adding release notes, CHANGELOG.md and CHANGELOG_CURRENT.md should be re-generated using:
```
make reno-report VERSION=<pre-release-verstion>
```
And then committed to the pre-release branch.

### Release workflow

`release` workflow tags the repo using `release_tag` variable either from the merged pre-release PR or from workflow dispatch.
It then uses the `goreleaser` tool to produce a github release and core images such as `ghcr.io/kanisterio/controller`, `ghcr.io/kanisterio/repo-server-controller`, `ghcr.io/kanisterio/kanister-tools` and `ghcr.io/kanisterio/kanister-kubectl-1.18`

Published release artifacts include helm chart for the operator.

Release workflow also builds and publishes docs and helm chart index into GH pages.

### Release notes

When working on pre-release PR the person doing the release should check that merged PRs have release notes when necessary.

It can be checked by running a diff with previous release and looking at the PR commits:

```
git log 0.106.0..HEAD --invert-grep --grep='deps' --grep='docs' --grep='build' | grep -oh '^.*\(#[0-9]*\)'
```

And verifying that PRs have release notes.

This is a bit tedious, but should be less of an issue as we establish a process of adding release notes.


## Handling Failures

### Retry

Depending on where the release failed, some of the following steps may have succeeded:

- Creation of a new tag in the kanister repo
- PR raised in kanister to update kanister tools

Before rerunning the release job manually, do the following:

- Delete the tag manually from GitHub
- Close the PRs created (to avoid confusion)

## Manual Release (Only needed when automation is broken)

Prerequisites:

- Make sure you have admin access or access to push tags on Kanister
- Make sure you have access to [push images to GHCR](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)

### Create pre-release

If pre-release pipeline does not work.

Assuming previous version is `0.42.0` and new release version is `0.43.0`

```
$ export PREV_TAG="0.42.0"
$ export RELEASE_TAG="0.43.0"
$ git checkout -b "kan-docs-${RELEASE_TAG}"
$ ./build/bump_version.sh "${PREV_TAG}" "${RELEASE_TAG}"
$ make reno-report VERSION="${RELEASE_TAG}"
$ git add -A
$ git commit -m"pre-release: Update version to ${RELEASE_TAG}"
$ git push origin kan-docs-${RELEASE_TAG}
```

Create PR from this branch.

### Create release binaries and images

If release pipeline does not work

1. Make sure to merge pre-release PR first.

2. Create a release tag

```
$ export RELEASE_TAG="0.43.0"
$ git checkout master
$ git pull origin master
$ git tag -a "${RELEASE_TAG}" -m "Release version";
$ git push origin "${RELEASE_TAG}"
```

3. Build helm charts

```
$ export PACKAGE_FOLDER=helm_package
$ export HELM_RELEASE_REPO_URL=https://github.com/kanisterio/kanister/releases/download/${RELEASE_TAG}
$ export HELM_RELEASE_REPO_INDEX=https://charts.kanister.io/
$ make package-helm VERSION=${RELEASE_TAG}
```

4. Release binaries and docker images

```
$ make gorelease CHANGELOG_FILE=./CHANGELOG.md GORELEASE_PARAMS='--draft'
```

5. Update and release docs and helms charts

**Currently Github pages publishing is only supported via Github actions**

### Publish Kanister Release

Finally, go to https://github.com/kanisterio/kanister/releases, and publish the draft release.

## Post-release Checks

- Verify the Kanister [repo](https://github.com/kanisterio/kanister/releases) for the new release tag.

- Verify if the docker [images](https://github.com/orgs/kanisterio/packages?repo_name=kanister) have a new tag. NOTE: Not all docker images are relevant.
	TODO: Add a list of the most relevant docker images to be verified here.

- Update the helm repo and check that helm charts version is up to date

```
helm repo update kanister
helm show chart kanister/kanister-operator
```

