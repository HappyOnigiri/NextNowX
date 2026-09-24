## nnx document

List or manage documents

### Synopsis

List or manage URL, local file, and stored Markdown documents.

Alias: doc.

```
nnx document [flags]
```

### Examples

```
nnx document
nnx document --task T-1
nnx document --project P-1
nnx doc
```

### Options

```
      --feature string   filter by feature ID
  -h, --help             help for document
      --project string   filter by project ID
      --task string      filter by task ID
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
* [nnx document add](nnx_document_add.md)	 - Add a document to a project, a feature, or a task
* [nnx document delete](nnx_document_delete.md)	 - Delete a document; missing documents return not_found
* [nnx document get](nnx_document_get.md)	 - Get one document
* [nnx document update](nnx_document_update.md)	 - Update a document

