## nnx update

Check for a newer Next Now X release and install it

### Synopsis

Check for a newer Next Now X release and install it.

The check reads the public GitHub releases of Next Now X without credentials, and the install runs the install.sh attached to the release being installed.
Development builds and demo runs report that updates are disabled.

```
nnx update [flags]
```

### Examples

```
nnx update
```

### Options

```
      --apply   install the newest release without asking
  -h, --help    help for update
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

