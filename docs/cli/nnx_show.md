## nnx show

Show a project, a feature, or a task by public identifier

### Synopsis

Show a project, a feature, or a task by public identifier.

The operand is a public project, feature, or task ID.

```
nnx show PROJECT_OR_FEATURE_OR_TASK [flags]
```

### Examples

```
nnx show F-1
nnx show T-1
nnx show P-1
```

### Options

```
  -h, --help   help for show
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

