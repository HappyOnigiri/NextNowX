## nnx config language update

Update the shared display and prompt language

### Synopsis

Update the shared display and prompt language.

LANGUAGE is "auto", "en", or "ja". With "auto", Next Now X reads LC_ALL, LC_MESSAGES, and LANG, and falls back to "en".
The language selects the built-in prompt templates and the WebUI display language. Templates you have customized keep the text you wrote.

```
nnx config language update LANGUAGE [flags]
```

### Examples

```
nnx config language update ja
```

### Options

```
  -h, --help   help for update
```

### Options inherited from parent commands

```
      --config string           YAML configuration path (env: NNX_CONFIG)
      --db string               SQLite database path (env: NNX_DB)
      --github-fixture string   GitHub fixture JSON path, or demo
      --json                    output JSON
```

### SEE ALSO

* [nnx config language](nnx_config_language.md)	 - Show or manage the shared display and prompt language

