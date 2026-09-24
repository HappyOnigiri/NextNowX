## nnx config label set

Set a global task label text or color

```
nnx config label set KEY [flags]
```

### Examples

```
nnx config label set status.in_progress --text Working
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

* [nnx config label](nnx_config_label.md)	 - Show or manage global task label overrides

