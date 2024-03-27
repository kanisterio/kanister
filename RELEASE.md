## Automated Release

Release process:

- Verify the release version by looking at the [releases page](https://github.com/kanisterio/kanister/releases) on the Kanister repo.
- Trigger the [pre-release](#pre-release-workflow) workflow with the desired version number (e.g. bump the minor version portion 0.106.0 -> 0.107.0)
- Review and validate created PR that it doesn't have any unintended changes
- Make sure to validate that all merged PRs in the release have [release notes](#release-notes)
	- Make sure that CHANGELOG.md and CHANGELOG_CURRENT.md contain release notes for the release version
	- **NOTE** While we establish the new process of release notes, it may be required to add notes in pre-release step by commiting them into pre-release branch
- Approve and merge the pre-release PR (it will be merged by `kueue` when approved)
- Merging of pre-release PR will trigger the `kanister/release` pipeline in codefresh
- The Kanister release job will publish a new tag, update documentation, build all the docker images, and push them to the [ghcr.io](https://github.com/orgs/kanisterio/packages) registry.
- After completing the Kanister part, the job will open a PR on K10 to update the Kanister version in K10 docs, tests, and the base image of the kanister-tools docker image. Example PR: https://github.com/kastenhq/k10/pull/23956
- Once the job is complete, a Slack notification will be sent to the kanister channel on the Kasten workspace.
	- **NOTE** We need to update the GVS blueprint version in the kio/kanister/blueprint.go file to ensure that the new kanister-tools image will be used for Kopia operations. Push a commit to the PR opened above to do this. Example: https://github.com/kastenhq/k10/pull/23956/commits/83a62ccb17af52fd331012239dea97e02180817b
- Once the PR is approved, the Kanister release is complete.
- Additionally, we could also approve the K10 go.mod update to bring in the latest changes in K10. Example PR: https://github.com/kastenhq/k10/pull/24589

### Pre-release workflow

`pre-release` workflow serves to create a new version of kanister, it updates version number and creates a PR in kanister repo.

The workflow can be triggered using workflow dispatch from the `Actions` tab in the repo: https://github.com/kanisterio/kanister/actions/workflows/pre-release.yml
It has a required input of `release_tag`, which should be set to a version next to the current version.

This will result in creating a PR with a version bump, like https://github.com/kanisterio/kanister/pull/2629
**NOTE** PR description would look like `pre-release: Update version to ...`

This PR would contain an update of version in various files.


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
- PR raised in K10 to update kanister tools version

Before rerunning the release job manually, do the following:

- Delete the tag manually from GitHub
- Close the PRs created (to avoid confusion)

## Manual Release (Only needed when automation is broken)

Prerequisites:

- Make sure you have admin access or access to push tags on Kanister
- Make sure you have push access to the K10 repo
- Make sure you have access to [push images to GHCR](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry) 

### Kanister Repo

1. Create a release tag

```
$ export RELEASE_TAG="0.43.0"
$ git checkout master
$ git pull origin master
$ git tag -a "${RELEASE_TAG}" -m "Release version";
$ git push origin "${RELEASE_TAG}"
```

2. Release binaries and docker images

```
$ make reno-report
$ make gorelease CHANGELOG_FILE=./CHANGELOG.md
```

3. Update and release docs and helms charts

```
$ export PREV_TAG="0.42.0"
$ git checkout -b "kan-docs-${RELEASE_TAG}"
$ ./build/bump_version.sh "${PREV_TAG}" "${RELEASE_TAG}"
$ git add -A
$ git commit -m"Kanister docs update to version ${RELEASE_TAG}"
$ git push origin kan-docs-${RELEASE_TAG}

```

Go ahead and raise PR against master through GitHub UI

### K10 Repo

4. Refresh artifacts

```
$ build.sh docker_run invalidate_cloudfront -e kanisterrelease

```

5. Bump kanister tools version in K10

```
$ export RELEASE_TAG="0.43.0"
$ export PREV_TAG="0.41.0"
$ git checkout -b "kantools-bump-${RELEASE_TAG}"
$ /path/to/kanister/build/bump_version.sh "${PREV_TAG}" "${RELEASE_TAG}" .
$ git add -A
$ git commit -m"Update kanister tools version to ${RELEASE_TAG}"
$ git push origin kantools-bump-${RELEASE_TAG}
```

Go ahead and raise PR against master through GitHub UI

### Publish Kanister Release

Finally, go to https://github.com/kanisterio/kanister/releases, and publish the draft release

## Post-release Checks

- Verify the Kanister [repo](https://github.com/kanisterio/kanister/releases) for the new release tag.

- Verify if the docker [images](https://github.com/orgs/kanisterio/packages?repo_name=kanister) have a new tag. NOTE: Not all docker images are relevant.
	TODO: Add a list of the most relevant docker images to be verified here.

