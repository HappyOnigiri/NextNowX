## nnx daemon

Show or manage the background Next Now X server

### Synopsis

Show or manage the background Next Now X server.

On macOS a LaunchAgent starts nnx serve at login. The LaunchAgent never passes --addr or --demo, so the background server always listens on loopback with real data.
Other operating systems report daemon_unsupported; run nnx serve directly there.

```
nnx daemon [flags]
```

### Examples

```
nnx daemon
```

### Options

```
  -h, --help   help for daemon
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: NNX_CONFIG)
      --db string               SQLite database path (env: NNX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [nnx](nnx.md)	 - Manage pull-request dependency roadmaps
* [nnx daemon install](nnx_daemon_install.md)	 - Register the LaunchAgent that starts Next Now X at login
* [nnx daemon restart](nnx_daemon_restart.md)	 - Replace the background server with a fresh one
* [nnx daemon start](nnx_daemon_start.md)	 - Ask launchd to start the background server
* [nnx daemon stop](nnx_daemon_stop.md)	 - Stop the background server without removing the LaunchAgent
* [nnx daemon uninstall](nnx_daemon_uninstall.md)	 - Remove the LaunchAgent and stop starting Next Now X at login

