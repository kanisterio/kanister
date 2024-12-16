# Tooling

There are two command-line tools that are built within the Kanister
repository.

## Kanctl

Although all Kanister custom resources can be managed using kubectl,
there are situations where this may be cumbersome. A canonical example
of this is backup/restore - Manually creating a restore ActionSet
requires copying Artifacts from the status of the complete backup
ActionSet, which is an error prone process. `kanctl` simplifies this
process by allowing the user to create custom Kanister resources -
ActionSets and Profiles, override existing ActionSets and validate
profiles.

`kanctl` has two top level commands:

- `create`
- `validate`

The usage of these commands, with some examples, has been show below:

### kanctl create

``` bash
$ kanctl create --help
Create a custom kanister resource

Usage:
  kanctl create [command]

Available Commands:
  actionset   Create a new ActionSet or override a <parent> ActionSet
  profile     Create a new profile
  repository-server   Create a new kopia repository server

Flags:
      --dry-run           if set, resource YAML will be printed but not created
  -h, --help              help for create
      --skip-validation   if set, resource is not validated before creation

Global Flags:
  -n, --namespace string   Override namespace obtained from kubectl context

Use "kanctl create [command] --help" for more information about a command.
```

As seen above, both ActionSets and profiles can be created using
`kanctl create`

``` bash
$ kanctl create actionset --help
Create a new ActionSet or override a <parent> ActionSet

Usage:
  kanctl create actionset [flags]

Flags:
  -a, --action string               action for the action set (required if creating a new action set)
  -b, --blueprint string            blueprint for the action set (required if creating a new action set)
  -c, --config-maps strings         config maps for the action set, comma separated ref=namespace/name pairs (eg: --config-maps ref1=namespace1/name1,ref2=namespace2/name2)
  -d, --deployment strings          deployment for the action set, comma separated namespace/name pairs (eg: --deployment namespace1/name1,namespace2/name2)
  -f, --from string                 specify name of the action set
  -h, --help                        help for actionset
  -k, --kind string                 resource kind to apply selector on. Used along with the selector specified using --selector/-l (default "all")
  -T, --namespacetargets strings    namespaces for the action set, comma separated list of namespaces (eg: --namespacetargets namespace1,namespace2)
  -O, --objects strings             objects for the action set, comma separated list of object references (eg: --objects group/version/resource/namespace1/name1,group/version/resource/namespace2/name2)
  -o, --options strings             specify options for the action set, comma separated key=value pairs (eg: --options key1=value1,key2=value2)
  -p, --profile string              profile for the action set
  -v, --pvc strings                 pvc for the action set, comma separated namespace/name pairs (eg: --pvc namespace1/name1,namespace2/name2)
  -s, --secrets strings             secrets for the action set, comma separated ref=namespace/name pairs (eg: --secrets ref1=namespace1/name1,ref2=namespace2/name2)
  -l, --selector string             k8s selector for objects
      --selector-namespace string   namespace to apply selector on. Used along with the selector specified using --selector/-l
  -t, --statefulset strings         statefulset for the action set, comma separated namespace/name pairs (eg: --statefulset namespace1/name1,namespace2/name2)

Global Flags:
      --dry-run            if set, resource YAML will be printed but not created
  -n, --namespace string   Override namespace obtained from kubectl context
      --skip-validation    if set, resource is not validated before creation
```

`kanctl create actionset` helps create ActionSets in a couple of
different ways. A common backup/restore scenario is demonstrated below.

Create a new Backup ActionSet

``` bash
# Action name and blueprint are required
$ kanctl create actionset --action backup --namespace kanister --blueprint time-log-bp \
                          --deployment kanister/time-logger                            \
                          --profile s3-profile
actionset backup-9gtmp created

# View the progress of the ActionSet
$ kubectl --namespace kanister describe actionset backup-9gtmp
```

Restore from the backup we just created

``` bash
# If necessary you can override the secrets, profile, config-maps, options etc obtained from the parent ActionSet
$ kanctl create actionset --action restore --from backup-9gtmp --namespace kanister
actionset restore-backup-9gtmp-4p6mc created

# View the progress of the ActionSet
$ kubectl --namespace kanister describe actionset restore-backup-9gtmp-4p6mc
```

Delete the Backup we created

``` bash
$ kanctl create actionset --action delete --from backup-9gtmp --namespace kanister
actionset delete-backup-9gtmp-fc857 created

# View the progress of the ActionSet
$ kubectl --namespace kanister describe actionset delete-backup-9gtmp-fc857
```

To make the selection of objects (resources on which actions are
performed) easier, you can filter on K8s labels using `--selector`.

``` bash
# backup deployment time-logger in namespace kanister using selectors
# if --kind deployment is not specified, all deployments, statefulsets and pvc matching the
# selector will be chosen for the action. You can also narrow down the search by setting the
# --selector-namespace flag
$ kanctl create actionset --action backup --namespace kanister --blueprint time-log-bp \
                          --selector app=time-logger                                   \
                          --kind deployment                                            \
                          --selector-namespace kanister --profile s3-profile
actionset backup-8f827 created
```

The `--dry-run` flag will print the YAML of the ActionSet without
actually creating it.

``` bash
# ActionSet creation with --dry-run
$ kanctl create actionset --action backup --namespace kanister --blueprint time-log-bp \
                          --selector app=time-logger                                   \
                          --kind deployment                                            \
                          --selector-namespace kanister                                \
                          --profile s3-profile                                         \
                          --dry-run
apiVersion: cr.kanister.io/v1alpha1
kind: ActionSet
metadata:
  creationTimestamp: null
  generateName: backup-
spec:
  actions:
  - blueprint: time-log-bp
    configMaps: {}
    name: backup
    object:
      apiVersion: ""
      kind: deployment
      name: time-logger
      namespace: kanister
    options: {}
    profile:
      apiVersion: ""
      kind: ""
      name: s3-profile
      namespace: kanister
    secrets: {}
```

Profile creation using `kanctl create`

``` bash
$ kanctl create profile --help
Create a new profile

Usage:
  kanctl create profile [command]

Available Commands:
  s3compliant Create new S3 compliant profile

Flags:
  -h, --help                    help for profile
      --skip-SSL-verification   if set, SSL verification is disabled for the profile

Global Flags:
      --dry-run            if set, resource YAML will be printed but not created
  -n, --namespace string   Override namespace obtained from kubectl context
      --skip-validation    if set, resource is not validated before creation

Use "kanctl create profile [command] --help" for more information about a command.
```

A new S3Compliant profile can be created using the s3compliant
subcommand

``` bash
$ kanctl create profile s3compliant --help
Create new S3 compliant profile

Usage:
  kanctl create profile s3compliant [flags]

Flags:
  -a, --access-key string   access key of the s3 compliant bucket
  -b, --bucket string       s3 bucket name
  -e, --endpoint string     endpoint URL of the s3 bucket
  -h, --help                help for s3compliant
  -p, --prefix string       prefix URL of the s3 bucket
  -r, --region string       region of the s3 bucket
  -s, --secret-key string   secret key of the s3 compliant bucket

Global Flags:
      --dry-run                 if set, resource YAML will be printed but not created
  -n, --namespace string        Override namespace obtained from kubectl context
      --skip-SSL-verification   if set, SSL verification is disabled for the profile
      --skip-validation         if set, resource is not validated before creation
```

``` bash
$ kanctl create profile s3compliant --bucket <bucket> --access-key ${AWS_ACCESS_KEY_ID} \
                                    --secret-key ${AWS_SECRET_ACCESS_KEY}               \
                                    --region us-west-1                                  \
                                    --namespace kanister
secret 's3-secret-chst2' created
profile 's3-profile-5mmkj' created
```

Kopia Repository Server resource creation using `kanctl create`

``` bash
$ kanctl create repository-server --help
Create a new RepositoryServer

Usage:
  kanctl create repository-server [flags]

Flags:
  -a, --admin-user-access-secret string           name of the secret having admin credentials to connect to connect to kopia repository server
  -r, --kopia-repository-password-secret string   name of the secret containing password for the kopia repository
  -k, --kopia-repository-user string              name of the user for accessing the kopia repository
  -c, --location-creds-secret string              name of the secret containing kopia repository storage credentials
  -l, --location-secret string                    name of the secret containing kopia repository storage location details
  -p, --prefix string                             prefix to be set in kopia repository
  -t, --tls-secret string                         name of the tls secret needed for secure kopia client and kopia repository server communication
  -u, --user string                               name of the user to be created for the kopia repository server
  -s, --user-access-secret string                 name of the secret having access credentials of the users that can connect to kopia repository server
  -w, --wait                                      wait for the kopia repository server to be in ready state after creation
  -h, --help                                      help for repository-server

Global Flags:
      --dry-run            if set, resource YAML will be printed but not created
  -n, --namespace string   Override namespace obtained from kubectl context
      --skip-validation    if set, resource is not validated before creation
      --verbose            Display verbose output
```

### kanctl validate

Profile and Blueprint resources can be validated using
`kanctl validate <resource>` command.

``` bash
$ kanctl validate --help
Validate custom Kanister resources

Usage:
  kanctl validate <resource> [flags]

Flags:
  -f, --filename string             yaml or json file of the custom resource to validate
  -v, --functionVersion string      kanister function version, e.g., v0.0.0 (defaults to v0.0.0)
  -h, --help                        help for validate
      --name string                 specify the K8s name of the custom resource to validate
      --resource-namespace string   namespace of the custom resource. Used when validating resource specified using
                                    --name. (default "default")
      --schema-validation-only      if set, only schema of resource will be validated

Global Flags:
  -n, --namespace string   Override namespace obtained from kubectl context
```

You can either validate an existing profile in K8s or a new profile yet
to be created.

``` bash
# validation of a yet to be created profile
$ cat << EOF | kanctl validate profile -f -
apiVersion: cr.kanister.io/v1alpha1
kind: Profile
metadata:
  name: s3-profile
  namespace: kanister
location:
  type: s3Compliant
  s3Compliant:
    bucket: XXXX
    endpoint: XXXX
    prefix: XXXX
    region: XXXX
credential:
  type: keyPair
  keyPair:
    idField: aws_access_key_id
    secretField: aws_secret_access_key
    secret:
      apiVersion: v1
      kind: Secret
      name: aws-creds
      namespace: kanister
skipSSLVerify: false
EOF
Passed the 'Validate Profile schema' check.. ✅
Passed the 'Validate bucket region specified in profile' check.. ✅
Passed the 'Validate read access to bucket specified in profile' check.. ✅
Passed the 'Validate write access to bucket specified in profile' check.. ✅
All checks passed.. ✅
```

Blueprint resources can be validated by specifying locally present
blueprint manifest using `-f` flag and optionally `-v` flag for kanister
function version.

``` bash

\# Download mysql blueprint locally \$ curl -O
<https://raw.githubusercontent.com/kanisterio/kanister/%7Cversion%7C/examples/mysql/mysql-blueprint.yaml>

\# Run blueprint validator \$ kanctl validate blueprint -f
mysql-blueprint.yaml Passed the \'validation of phase dumpToObjectStore
in action backup\' check.. ✅ Passed the \'validation of phase
deleteFromBlobStore in action delete\' check.. ✅ Passed the
\'validation of phase restoreFromBlobStore in action restore\' check..
✅
```

`kanctl validate blueprint` currently verifies the Kanister function
names and presence of the mandatory arguments to those functions.

## Kando

A common use case for Kanister is to transfer data between Kubernetes
and an object store like AWS S3. We\'ve found it can be cumbersome to
pass Profile configuration to tools like the AWS command line from
inside Blueprints.

`kando` is a tool to simplify object store interactions from within
blueprints. It also provides a way to create desired output from a
blueprint phase.

It has the following commands:

- `location push`
- `location pull`
- `location delete`
- `output`

The usage for these commands can be displayed using the `--help` flag:

``` bash
$ kando location pull --help
Pull from s3-compliant object storage to a file or stdout

Usage:
  kando location pull <target> [flags]

Flags:
  -h, --help   help for pull

Global Flags:
  -s, --path string      Specify a path suffix (optional)
  -p, --profile string   Pass a Profile as a JSON string (required)
```

``` bash
$ kando location push --help
Push a source file or stdin stream to s3-compliant object storage

Usage:
  kando location push <source> [flags]

Flags:
  -h, --help   help for push

Global Flags:
  -s, --path string      Specify a path suffix (optional)
  -p, --profile string   Pass a Profile as a JSON string (required)
```

``` bash
$ kando location delete --help
Delete artifacts from s3-compliant object storage

Usage:
  kando location delete [flags]

Flags:
  -h, --help   help for delete

Global Flags:
  -s, --path string      Specify a path suffix (optional)
  -p, --profile string   Pass a Profile as a JSON string (required)
```

``` bash
$ kando output --help
Create phase output with given key:value

Usage:
  kando output <key> <value> [flags]

Flags:
  -h, --help   help for output
```

The following snippet is an example of using kando from inside a
Blueprint.

``` bash
kando location push \--profile \'{{ toJson .Profile }}\' \--path
\'/backup/path\' -

kando location delete \--profile \'{{ toJson .Profile }}\' \--path
\'/backup/path\'

kando output version
```

## Install the tools

Installation of the tools requires [Go](https://golang.org/doc/install)
to be installed

``` bash
# The script installs both kanctl and kando
$ curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash
```

## Docker Image

These tools, especially `kando` are meant to be invoked inside
containers via Blueprints. Although suggest using the released image
when possible, we\'ve also made it simple to add these tools to your
container.

The released image, `ghcr.io/kanisterio/kanister-tools`, is hosted by
[github container
registry](https://github.com/orgs/kanisterio/packages/container/package/kanister-tools).

The Dockerfile for this image is in the [kanister github
repo](https://github.com/kanisterio/kanister/blob/master/docker/tools/Dockerfile).

To add these tools to your own image, you can add the following command
to your Dockerfile:

``` console
RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash
``` -->
