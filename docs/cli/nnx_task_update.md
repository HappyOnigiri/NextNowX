## nnx task update

Update a task by ID

```
nnx task update TASK_ID [flags]
```

### Examples

```
nnx task update TASK_ID --status completed
```

### Options

```
      --assignee string   new assignee
  -h, --help              help for update
      --scope string      new scope
      --status string     not_started, designing, in_progress, completed, or closed
      --title string      new title
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

