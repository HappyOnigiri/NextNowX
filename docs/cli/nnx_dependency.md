## nnx dependency

List or manage directed blocker edges

### Synopsis

List or manage directed blocker edges.

Alias: dep.

```
nnx dependency [flags]
```

### Examples

```
nnx dependency
nnx dep
```

### Options

```
  -h, --help   help for dependency
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
* [nnx dependency add](nnx_dependency_add.md)	 - Add a blocker-to-blocked dependency
* [nnx dependency remove](nnx_dependency_remove.md)	 - Remove a dependency; missing edges return not_found

