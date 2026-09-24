## nnx config sync update

Update the automatic GitHub synchronization interval

### Synopsis

Update the automatic GitHub synchronization interval.

INTERVAL_SECONDS is a whole number of seconds and must be at least 600.

```
nnx config sync update INTERVAL_SECONDS [flags]
```

### Examples

```
nnx config sync update 3600 --json
```

### Options

```
  -h, --help   help for update
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: NNX_CONFIG)
      --db string               SQLite database path (env: NNX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [nnx config sync](nnx_config_sync.md)	 - Show or manage automatic GitHub synchronization settings

