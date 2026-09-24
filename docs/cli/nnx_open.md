## nnx open

Open the running Next Now X WebUI in a browser

### Synopsis

Open the running Next Now X WebUI in a browser.

The URL comes from the running server itself, so it is correct even when the port fell back to an ephemeral one. This never starts a server; use nnx daemon start for that.

```
nnx open [flags]
```

### Examples

```
nnx open --print
```

### Options

```
  -h, --help    help for open
      --print   print the URL instead of opening a browser
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: NNX_CONFIG)
      --db string               SQLite database path (env: NNX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [nnx](nnx.md)	 - Manage pull-request dependency roadmaps

