## nnx project prompt set

Set a prompt template override

```
nnx project prompt set PROJECT_ID KIND [flags]
```

### Examples

```
nnx project prompt set p-1 design --file prompt.txt
```

### Options

```
      --file string   read the template from a file
  -h, --help          help for set
      --stdin         read the template from standard input
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: NNX_CONFIG)
      --db string               SQLite database path (env: NNX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [nnx project prompt](nnx_project_prompt.md)	 - Manage prompt template overrides

