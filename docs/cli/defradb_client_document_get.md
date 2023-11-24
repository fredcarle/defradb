## defradb client document get

View detailed document info.

### Synopsis

View detailed document info.

Example:
  defradb client document get --collection User bae-123
		

```
defradb client document get --collection <collection> <docID> [--show-deleted] [flags]
```

### Options

```
  -c, --collection string   Collection name
  -h, --help                help for get
      --show-deleted        Show deleted documents
```

### Options inherited from parent commands

```
      --logformat string     Log format to use. Options are csv, json (default "csv")
      --logger stringArray   Override logger parameters. Usage: --logger <name>,level=<level>,output=<output>,...
      --loglevel string      Log level to use. Options are debug, info, error, fatal (default "info")
      --lognocolor           Disable colored log output
      --logoutput string     Log output path (default "stderr")
      --logtrace             Include stacktrace in error and fatal logs
      --rootdir string       Directory for data and configuration to use (default: $HOME/.defradb)
      --tx uint              Transaction ID
      --url string           URL of HTTP endpoint to listen on or connect to (default "localhost:9181")
```

### SEE ALSO

* [defradb client document](defradb_client_document.md)	 - Create, read, update, and delete documents.

