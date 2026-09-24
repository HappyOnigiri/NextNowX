## nnx dependency add

Add a blocker-to-blocked dependency

```
nnx dependency add BLOCKER_TASK_ID BLOCKED_TASK_ID [flags]
```

### Examples

```
nnx dependency add BLOCKER_TASK_ID BLOCKED_TASK_ID
```

### Options

```
  -h, --help   help for add
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: NNX_CONFIG)
      --db string               SQLite database path (env: NNX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [nnx dependency](nnx_dependency.md)	 - List or manage directed blocker edges

