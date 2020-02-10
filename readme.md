# Tedi - Test Environment with Dependency Injection

Tedi tries to make testing in go less tedious by extending the built-in test framework in Golang with dependency injection.

## How to use `tedi`?

You can simply swap `tedi test` with `go test`.

`tedi test` will first generate the `tedi_test.go` file and then call the go test command.

### With `go test`

If you still want to use `go test` you can add:

```
    //go:generate tedi generate
```

to a file in the go package where you want to use `tedi`. Before running your run `go test` run `go generate`.

## What does `tedi` provide?



## The command line tool

### Match functions

## Annotations

## Auto matching
