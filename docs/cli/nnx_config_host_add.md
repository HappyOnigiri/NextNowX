## nnx config host add

Add a GitHub.com or Enterprise host

### Synopsis

Add a GitHub.com or Enterprise host.

HOST is a hostname with an optional port.

```
nnx config host add HOST [flags]
```

### Examples

```
nnx config host add ghe.example.com
```

### Options

```
      --api-url string       HTTPS API base URL (defaults from host)
      --graphql-url string   HTTPS GraphQL URL (defaults from host)
  -h, --help                 help for add
      --upload-url string    HTTPS upload base URL (defaults from host)
      --web-url string       HTTPS web URL (defaults from host)
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: NNX_CONFIG)
      --db string               SQLite database path (env: NNX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [nnx config host](nnx_config_host.md)	 - List or manage configured GitHub hosts

