## nnx daemon stop

Stop the background server without removing the LaunchAgent

### Synopsis

Stop the background server without removing the LaunchAgent.

Next Now X sends SIGTERM to the recorded process; launchctl bootout is not used because it would unregister the LaunchAgent until the next login.

```
nnx daemon stop [flags]
```

### Examples

```
nnx daemon stop
```

### Options

```
  -h, --help   help for stop
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

