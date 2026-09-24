## nnx project update

Update a project by ID

```
nnx project update PROJECT_ID [flags]
```

### Examples

```
nnx project update P-1 --title "Payments platform"
```

### Options

```
      --archived             archive (true) or unarchive (false) the project
      --description string   new description
  -h, --help                 help for update
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

* [nnx project](nnx_project.md)	 - List projects or show one by ID

