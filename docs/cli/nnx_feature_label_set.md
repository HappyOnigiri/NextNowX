## nnx feature label set

Set a task label text or color override

```
nnx feature label set FEATURE_ID KEY [flags]
```

### Examples

```
nnx feature label set FEATURE_ID status.in_progress --text Working
```

### Options

```
      --color string   label color (#RRGGBB)
  -h, --help           help for set
      --text string    label text
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: NNX_CONFIG)
      --db string               SQLite database path (env: NNX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [nnx feature label](nnx_feature_label.md)	 - Manage feature task label overrides

