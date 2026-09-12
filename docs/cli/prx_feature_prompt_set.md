## prx feature prompt set

Set a prompt template override

```
prx feature prompt set FEATURE_ID KIND [flags]
```

### Examples

```
prx feature prompt set f-1 design --file prompt.txt
```

### Options

```
      --file string   read the template from a file
  -h, --help          help for set
      --stdin         read the template from standard input
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [prx feature prompt](prx_feature_prompt.md)	 - Manage prompt template overrides

