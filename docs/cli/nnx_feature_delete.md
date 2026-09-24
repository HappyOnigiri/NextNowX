## nnx feature delete

Delete a feature and optionally its contained data

```
nnx feature delete FEATURE_ID [flags]
```

### Examples

```
nnx feature delete F-1 --cascade
```

### Options

```
      --cascade   delete contained tasks and references
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

* [nnx feature](nnx_feature.md)	 - List features or show one by ID

