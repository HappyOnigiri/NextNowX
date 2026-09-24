## nnx config auth

List or manage host-scoped authentication methods

```
nnx config auth [flags]
```

### Examples

```
nnx config auth
```

### Options

```
  -h, --help   help for auth
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: NNX_CONFIG)
      --db string               SQLite database path (env: NNX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [nnx config](nnx_config.md)	 - Show or manage GitHub hosts and authentication
* [nnx config auth add](nnx_config_auth_add.md)	 - Add a host-scoped authentication method
* [nnx config auth remove](nnx_config_auth_remove.md)	 - Remove an authentication method and its cached use
* [nnx config auth reorder](nnx_config_auth_reorder.md)	 - Set authentication priority order
* [nnx config auth update](nnx_config_auth_update.md)	 - Update a host-scoped authentication method

