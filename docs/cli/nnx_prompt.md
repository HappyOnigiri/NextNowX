## nnx prompt

Print the agent prompt for a task

### Synopsis

Print the agent prompt for a task.

Without --kind, a task with no implementation plan gets the design prompt and a task with one gets the implementation prompt.
The result resolves global, project, and feature overrides, so the WebUI copies the same text.

```
nnx prompt TASK_ID [flags]
```

### Examples

```
nnx prompt T-1
nnx prompt T-1 --kind design
nnx prompt T-1 --json
```

### Options

```
  -h, --help          help for prompt
      --kind string   template to render: design or implementation (default: derived from the implementation plan)
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

