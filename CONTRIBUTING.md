# Contributing to Kanister

## How Can I Contribute?
  ### * Reporting Bugs
  Before creating bug reports, please check the list of issues as you might find out that you don't need to create one. Also when creating a bug report, please include as many details as possible as this information may help us resolve the issue faster.

  ### * Suggesting Enhancements
  You can suggest minor improvements to existing functionality and or a completely new feature. 

  ### * Your first code contribution
  You can start by looking through `good-first-issue` issues:

## Local Development
Once you are done with your changes, you need to ensure that your changes do not fail in the CI build

```bash

#Run golint command to make sure your code is properly formatted
make golint

#build the project
make build

#Run unit test 
make test

#Run E2E test
#From the project root directory
make integration-test

```
To test and deploy your changes to a local Kubernetes cluster refer this https://docs.kanister.io/install.html#building-and-deploying-from-source

## Contributing to documentation
For complete documentation visit https://docs.kanister.io/

Kanister docs are generated using [Sphinx](https://www.sphinx-doc.org/en/master/) and are written in [reStructuredText](https://docutils.sourceforge.io/rst.html). The source `.rst` files are in the Kanister repository under the `/docs` folder.

### Updating documentation
- Modify or add `.rst` file(s) under the `/docs` folder.

- Build Docs locally.
```bash
make docs
```

- The above command will build the documentation, check for errors and place the final output in the `/docs/_build/html` directory.

- The built docs can be viewed and validated visually by opening `/docs/_build/html/index.html`.

- Push a PR with the changes for review.

## Submitting issues
If you find a bug or have a feature request, please submit an issue at https://github.com/kanisterio/kanister/issues

## Submitting code via Pull Requests
* We follow the [Github Pull Request Model](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/about-pull-requests) for all contributions.
* For large bodies of work, we recommend creating an issue and labeling it design outlining the feature that you wish to build, and describing how it will be implemented. This gives a chance for review to happen early and ensures no wasted effort occurs.
* For new features, documentation must be included.
* Once review has occurred, please rebase your PR down to a single commit. This will ensure a nice clean Git history.

## Contacting Developers
Using [Slack](https://join.slack.com/t/kanisterio/shared_invite/enQtNzg2MDc4NzA0ODY4LTU1NDU2NDZhYjk3YmE5MWNlZWMwYzk1NjNjOGQ3NjAyMjcxMTIyNTE1YzZlMzgwYmIwNWFkNjU0NGFlMzNjNTk) is the quickest way to get in touch with developers.
