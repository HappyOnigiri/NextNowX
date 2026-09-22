## prx setup

Choose how PRX should start

### Synopsis

Choose how PRX should start.

Setup first asks for a language and saves it as the language setting; its questions, its progress messages, and the sample data follow that choice.
On macOS, the setup walk can register the LaunchAgent, start the background server, and open the WebUI.
When it creates the database, setup adds a small sample project.

```
prx setup [flags]
```

### Examples

```
prx setup
```

### Options

```
  -h, --help             help for setup
      --no-sample-data   skip the first-run sample data (env: PRX_NO_SAMPLE_DATA, any non-empty value)
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

