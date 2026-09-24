## nnx config auth update

Update a host-scoped authentication method

```
nnx config auth update AUTH_METHOD_ID [flags]
```

### Examples

```
nnx config auth update work-gh --user octocat
```

### Options

```
      --account string    Keychain account
  -h, --help              help for update
      --host string       configured host
      --new-id string     new authentication method ID
      --service string    Keychain service
      --token-stdin       read an inline token from stdin
      --type string       keychain, environment, inline, or gh_cli
      --user string       gh CLI user
      --variable string   environment variable name
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

