## nnx project label unset

Remove a task label text or color override

```
nnx project label unset PROJECT_ID KEY [flags]
```

### Examples

```
nnx project label unset PROJECT_ID status.in_progress --text
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

* [nnx project label](nnx_project_label.md)	 - Manage project task label overrides

