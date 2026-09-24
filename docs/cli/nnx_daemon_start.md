## nnx daemon start

Ask launchd to start the background server

### Synopsis

Ask launchd to start the background server.

Starting an already running server succeeds without replacing it.

```
nnx daemon start [flags]
```

### Examples

```
nnx daemon start
```

### Options

```
  -h, --help   help for start
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: NNX_CONFIG)
      --db string               SQLite database path (env: NNX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [nnx daemon](nnx_daemon.md)	 - Show or manage the background Next Now X server

