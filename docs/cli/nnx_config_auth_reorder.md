## nnx config auth reorder

Set authentication priority order

```
nnx config auth reorder AUTH_METHOD_ID... [flags]
```

### Examples

```
nnx config auth reorder ghe-environment ghe-cli
```

### Options

```
  -h, --help   help for reorder
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: NNX_CONFIG)
      --db string               SQLite database path (env: NNX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [nnx config auth](nnx_config_auth.md)	 - List or manage host-scoped authentication methods

