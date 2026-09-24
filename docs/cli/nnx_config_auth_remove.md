## nnx config auth remove

Remove an authentication method and its cached use

```
nnx config auth remove AUTH_METHOD_ID [flags]
```

### Examples

```
nnx config auth remove work-gh
```

### Options

```
  -h, --help   help for remove
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

