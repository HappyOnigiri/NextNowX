## nnx sync

Refresh GitHub state for pull-request tasks

```
nnx sync [flags]
```

### Examples

```
nnx sync --feature FEATURE_ID
nnx sync --task TASK_ID
```

### Options

```
      --feature string   feature ID
  -h, --help             help for sync
      --task string      task ID
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
* [nnx sync status](nnx_sync_status.md)	 - Show automatic GitHub synchronization status

