## prx update

Check for a newer PRX release and install it

### Synopsis

Check for a newer PRX release and install it.

The check reads the public GitHub releases of PRX without credentials, and the install runs the install.sh attached to the release being installed.
Development builds and demo runs report that updates are disabled.

```
prx update [flags]
```

### Examples

```
prx update
```

### Options

```
      --apply   install the newest release without asking
  -h, --help    help for update
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: PRX_CONFIG)
      --db string               SQLite database path (env: PRX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [prx](prx.md)	 - Manage pull-request dependency roadmaps

