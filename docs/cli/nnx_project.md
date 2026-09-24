## nnx project

List projects or show one by ID

### Synopsis

List projects or show one by ID.

Alias: proj.

```
nnx project [PROJECT_ID] [flags]
```

### Examples

```
nnx project
nnx project P-1
nnx proj P-1
```

### Options

```
  -h, --help   help for project
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
* [nnx project archive](nnx_project_archive.md)	 - Archive a project and make its features read-only
* [nnx project create](nnx_project_create.md)	 - Create a project
* [nnx project delete](nnx_project_delete.md)	 - Delete a project; --cascade removes its documents and the features it holds
* [nnx project label](nnx_project_label.md)	 - Manage project task label overrides
* [nnx project prompt](nnx_project_prompt.md)	 - Manage prompt template overrides
* [nnx project unarchive](nnx_project_unarchive.md)	 - Unarchive a project and let its features accept writes again
* [nnx project update](nnx_project_update.md)	 - Update a project by ID

