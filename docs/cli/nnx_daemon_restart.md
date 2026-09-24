## nnx daemon restart

Replace the background server with a fresh one

### Synopsis

Replace the background server with a fresh one.

Use this after installing a new Next Now X binary so the server stops answering with an older embedded schema.

```
nnx daemon restart [flags]
```

### Examples

```
nnx daemon restart
```

### Options

```
  -h, --help   help for restart
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

