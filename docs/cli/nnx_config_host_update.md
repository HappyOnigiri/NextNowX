## nnx config host update

Update a configured GitHub host

```
nnx config host update HOST [flags]
```

### Examples

```
nnx config host update ghe.example.com --api-url https://ghe.example.com/api/v3/
```

### Options

```
      --api-url string       new HTTPS API base URL
      --graphql-url string   new HTTPS GraphQL URL
  -h, --help                 help for update
      --new-host string      new hostname
      --upload-url string    new HTTPS upload base URL
      --web-url string       new HTTPS web URL
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

