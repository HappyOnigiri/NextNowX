## nnx daemon install

Register the LaunchAgent that starts Next Now X at login

### Synopsis

Register the LaunchAgent that starts Next Now X at login.

launchd starts the server right away, so the command waits until that server is listening and reports its address.

```
nnx daemon install [flags]
```

### Examples

```
nnx daemon install
```

### Options

```
  -h, --help   help for install
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

