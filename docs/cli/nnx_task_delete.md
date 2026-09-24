## nnx task delete

Delete a task and optionally its dependencies and references

```
nnx task delete TASK_ID [flags]
```

### Examples

```
nnx task delete TASK_ID --cascade
```

### Options

```
      --cascade   delete dependencies and references
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

* [nnx task](nnx_task.md)	 - List tasks or show one by ID

