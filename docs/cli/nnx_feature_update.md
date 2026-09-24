## nnx feature update

Update a feature by ID

```
nnx feature update FEATURE_ID [flags]
```

### Examples

```
nnx feature update F-1 --archived=false
nnx feature update F-1 --project P-2
```

### Options

```
      --archived             archive (true) or unarchive (false) the feature
      --description string   new description
  -h, --help                 help for update
      --project string       project ID to move the feature to
      --status string        auto, active, paused, completed, or cancelled
      --title string         new title
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: NNX_CONFIG)
      --db string               SQLite database path (env: NNX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [nnx feature](nnx_feature.md)	 - List features or show one by ID

