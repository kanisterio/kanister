## Release notes

Release notes use [https://docs.openstack.org/reno/latest/](reno)

Release notes are stored in `releasenotes/notes` directory.

Reno allows to generate release notes using files in the repository as opposed to generating from commit messages.
This makes it easier to review and edit release notes.

## Development flow

When submitting a PR with some changes worthy of mentioning in the notes (new feature, bugfix, deprecation, update requirements),
committer should add a new note file using `reno new <note_name>` or `make reno-new note=<note_name>`.

New file will be created in `releasenotes/notes` directory with default template.
Change notes should be added to this file to reflect the change and additional information such as deprecations or upgrade requirements.
It's recommended to remove unused template fields.

When reviewing a PR, a reviewer should check if there are change notes added if necessary and either request or add a new note if they have push access to the branch

## Generating changelogs

Changelog can be generated using:

```
make reno-report
```
or to mark specific current version:
```
make reno-report VERSION=$CURRENT_VERSION
```

This will create a CHANGELOG.md and CHANGELOG_CURRENT.md files with changes from committed release notes.
CHANGELOG_CURRENT.md only containing changes for the current version.

It will be used when running goreleaser build using make (will use `CHANGELOG_CURRENT.md` by default):

```
make gorelease CHANGELOG_FILE=./CHANGELOG_CURRENT.md
```

