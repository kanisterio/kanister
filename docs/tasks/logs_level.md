# Modifying Kanister Log Level

Kanister uses structured logging to ensure that its logs can be easily
categorized, indexed and searched by downstream log aggregation
software.

The default logging level of Kanister is set to `info`. This logging
level can be changed by modifying the value of the `LOG_LEVEL`
environment variable of the Kanister container.

When using Helm, this value can be configured using the
`controller.logLevel` variable. For example, to set the logging level to
`debug`:

``` bash
helm -n kanister upgrade --install kanister \
  --set controller.logLevel=debug \
  --create-namespace kanister/kanister-operator
```

The supported logging levels are:

- `panic`
- `fatal`
- `error`
- `info`
- `debug`
- `trace`
