## nnx document add

Add a document to a project, a feature, or a task

### Synopsis

Add a document to a project, a feature, or a task.

The operand is a public project, feature, or task ID.

```
nnx document add PROJECT_OR_FEATURE_OR_TASK [flags]
```

### Examples

```
nnx document add T-1 --url https://example.com
nnx document add P-1 --url https://example.com
nnx document add F-1 --markdown-file notes.md
```

### Options

```
  -h, --help                   help for add
      --implementation-plan    mark as the task implementation plan
      --local-file string      registered local file path
      --markdown-file string   read stored Markdown from a file
      --stdin                  read stored Markdown from standard input
      --title string           document title
      --url string             HTTP or HTTPS URL
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: NNX_CONFIG)
      --db string               SQLite database path (env: NNX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [nnx document](nnx_document.md)	 - List or manage documents

