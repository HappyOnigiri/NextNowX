## nnx pr attach

Attach a GitHub pull request to a task

```
nnx pr attach TASK_ID URL [flags]
```

### Examples

```
nnx pr attach TASK_ID https://github.com/acme/payments/pull/42
```

### Options

```
  -h, --help   help for attach
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: NNX_CONFIG)
      --db string               SQLite database path (env: NNX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [nnx pr](nnx_pr.md)	 - List or attach GitHub pull requests

