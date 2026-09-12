## prx prompt

Print the agent prompt for a task

### Synopsis

Print the agent prompt for a task.

A task without an implementation plan gets the design prompt, and a task with one gets the implementation prompt.
The result resolves global, project, and feature overrides, so the WebUI copies the same text.

```
prx prompt TASK_ID [flags]
```

### Examples

```
prx prompt T-1
prx prompt T-1 --json
```

### Options

```
  -h, --help   help for prompt
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [prx](prx.md)	 - Manage pull-request dependency roadmaps

