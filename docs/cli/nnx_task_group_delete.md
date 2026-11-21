## nnx task group delete

Delete a group and ungroup its tasks; --cascade deletes its references

### Synopsis

Delete a group and ungroup its tasks. If the group has attached references, --cascade is required to delete them.

```
nnx task group delete GROUP_ID [flags]
```

### Examples

```
nnx task group delete G-1 --cascade
```

### Options

```
      --cascade   delete the group's attached references
  -h, --help      help for delete
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: NNX_CONFIG)
      --db string               SQLite database path (env: NNX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [nnx task group](nnx_task_group.md)	 - Manage task groups in a feature

