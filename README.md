# Turnin Compute

https://github.com/bradenn/turnin-compute

This module is responsible for handling all the computationally intense tasks generated by turnin-nexus.

### Data Flow

```
main.go -> (config.Init, server.Init) -> router.NewRouter -> controllers.Compile | controllers.Submission | controllers.Test
```

```
submission (mod) -> submission.go -> enc.Enclave -> file.go -> compile.go -> test.go -> grader.go -> submission.go  
```

### Notable Components

- Gin
- UUID

### Unix Tools Used During Operation

- Diff
- find
- ~~Valgrind~~ No longer sufficiently maintained on all platforms, see temporary
  solution: [bradenn/heapusage](https://github.com/bradenn/heapusage)

## Dependencies

Nothing really, see go.mod if you must.

## License

Copyright &copy; Braden Nicholson 2019 - 2021

All Rights Reserved. Do not distribute.
