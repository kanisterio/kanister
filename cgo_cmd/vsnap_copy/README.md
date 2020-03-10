# vSphere snapshot commands

This is an application intended to copy data between and object store and a vSphere snapshot.

## Usage

### push
- **description:** Writes data from a vSphere snapshot to an object store.
- **inputs:** 
- - snapshotID 
- - (-p) [profile](https://docs.kanister.io/architecture.html#profiles)
- - (-v) vsphere credentials. Must have the json form - 
```bash
{ "vchost":"xxxx", "vcuser":"xxxx", "vcpass":"xxxx", "s3urlbase": "xxxx"}'
``` 
- - (-s) an optional path

- **example usage:**
```bash
LD_LIBRARY_PATH=/opt/vddk/lib64 bin/amd64/vsnap_copy push ivd:asdfaf:adfaf -p '{"apiVersion":"cr.kanister.io/v1alpha1","credential":{"secret":{"apiVersion":"v1","group":"","kind":"Secret","name":"XXXX","namespace":"kasten-io","resource":""},"type":"secret"},"kind":"Profile","location":{"bucket":"XXXX","endpoint":"","prefix":"","region":"us-west-1","type":"s3Compliant"},"skipSSLVerify":false}' -v '{ "vchost":"host", "vcuser":"user", "vcpass":"pass", "s3urlbase": "something"}'
```

### pull
Unsupported