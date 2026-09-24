## nnx config auth add

Add a host-scoped authentication method

### Synopsis

Add a host-scoped authentication method.

AUTH_METHOD_ID names the method and must be unique.
HOST is a configured host.
TYPE is keychain, environment, inline, or gh_cli.

Each type needs its own credential flags: keychain needs --account and --service, environment needs --variable, and inline needs --token-stdin.

```
nnx config auth add AUTH_METHOD_ID HOST TYPE [flags]
```

### Examples

```
nnx config auth add work-gh github.com gh_cli
```

### Options

```
      --account string    Keychain account
  -h, --help              help for add
      --service string    Keychain service
      --token-stdin       read an inline token from stdin
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

