## nnx config label unset

Remove a global task label text or color override

```
nnx config label unset KEY [flags]
```

### Examples

```
nnx config label unset status.in_progress --text
```

### Options

```
      --color   remove color override
  -h, --help    help for unset
      --text    remove text override
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

