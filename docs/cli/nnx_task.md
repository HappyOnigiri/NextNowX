## nnx task

List tasks or show one by ID

### Synopsis

List tasks or show one by ID.

Alias: t.

```
nnx task [TASK_ID] [flags]
```

### Examples

```
nnx task
nnx task --feature checkout
nnx task T-1
nnx t T-1
```

### Options

```
      --feature string   filter by feature ID
  -h, --help             help for task
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
* [nnx task create](nnx_task_create.md)	 - Create a task
* [nnx task delete](nnx_task_delete.md)	 - Delete a task and optionally its dependencies and references
* [nnx task update](nnx_task_update.md)	 - Update a task by ID

