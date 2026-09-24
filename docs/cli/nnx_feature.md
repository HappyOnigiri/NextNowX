## nnx feature

List features or show one by ID

### Synopsis

List features or show one by ID.

Alias: f.

```
nnx feature [FEATURE_ID] [flags]
```

### Examples

```
nnx feature
nnx feature F-1
nnx f F-1
```

### Options

```
  -h, --help   help for feature
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
* [nnx feature archive](nnx_feature_archive.md)	 - Archive a feature by ID
* [nnx feature create](nnx_feature_create.md)	 - Create a feature in a project
* [nnx feature delete](nnx_feature_delete.md)	 - Delete a feature and optionally its contained data
* [nnx feature label](nnx_feature_label.md)	 - Manage feature task label overrides
* [nnx feature prompt](nnx_feature_prompt.md)	 - Manage prompt template overrides
* [nnx feature unarchive](nnx_feature_unarchive.md)	 - Unarchive a feature by ID
* [nnx feature update](nnx_feature_update.md)	 - Update a feature by ID

