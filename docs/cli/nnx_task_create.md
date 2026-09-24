## nnx task create

Create a task

```
nnx task create FEATURE_ID TITLE [flags]
```

### Examples

```
nnx task create F-1 "Add payment intent API" --assignee Bob
nnx task create F-1 -- "-fix login redirect"
```

### Options

```
      --assignee string   assignee
  -h, --help              help for create
      --scope string      scope description
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

