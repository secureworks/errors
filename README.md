# Errors

This package provides a suite of tools meant to work with Go 1.13 error 
wrapping to give users all the basics to handle errors in a useful way.

> _Another errors package? Why does Go need all of these error libraries?_

Because the language and the standard library have a minimal approach to error 
handling that leaves out some primitives power users expect to have on 
hand. 

Important among these primitives are stack traces, error collections 
(multi-errors and error groups) and error types. While Go 1.13 introduced 
error wrapping utilities that have fixed some immediate issues, it is really 
useful to have a few more tools on hand.

### Installation

> _This package **may not be used** in environments:_
>
> 1. &nbsp; _running Go 1.12 or lower;_
> 2. &nbsp; _running on 32-bit architecture;_
> 3. &nbsp; _running on Windows (**currently** not supported)._

Given that we are running on Go 1.13, your project should be using Go modules. 
Add the following to your file:

```go
import "github.com/secureworks/errors"
```

Then, when you run any Go command the toolchain will resolve and fetch the 
required modules automatically.

Because this package re-exports the package-level functions of the standard 
library `"errors"` package, you do not need to include that package as well to 
get `New`, `As`, `Is`, or `Unwrap`.

### Use

[Documentation is available on pkg.go.dev][docs]. You may also look at the 
examples in the package.

The most important features are listed below.

Package `github.com/secureworks/errors`:

- use in place of the standard library with no change in behavior;
- use the `errors.MultiError` type as either an explicit multierror 
  implementation or as an implicit multierror passed around with the default 
  `error` interface; use `errors.Append` and others to simplify multierror 
  management in your code;
- embed (singular) stack frames with `errors.NewWithFrame("...")`, 
  `errors.WithFrame(err)`, and `fmt.Errorf("...: %w", err)`;
- embed stack traces with `errors.NewWithStackTrace("...")` and
  `errors.WithStackTrace(err)`;
- remove error context with `errors.Mask(err)`, `errors.Opaque(err)`, and
  `errors.WithMessage(err, "...")`;
- marshal and unmarshal stack traces as text or JSON.

Package `github.com/secureworks/errors/syncerr`:

- use `syncerr.CoordinatedGroup` to run a group of go routines (in parallel or 
  in series) and synchronize the controlling process on their completion; in 
  essence do exactly what is done in `golang.org/x/sync/errgroup`;
- use `syncerr.ParallelGroup` to run a group of go routines in parallel and 
  coalesce their results into a single multierror.

### Roadmap

Possible improvements before reaching `v1.0` include:

- **Add support for Windows filepaths in call frames.**
- Add direct integrations with other errors packages (especially those listed 
  in the codebase).
- Include either a linter or a suggested [`golang-ci`][golang-ci] lint YAML 
  to support idiomatic use.

### License

This library is distributed under the [Apache-2.0 license][Apache-2.0] found in the 
[LICENSE](./LICENSE) file.

### Dependencies

This package has no dependencies, but is modeled on other, similar open source 
libraries. See the codebase for any specific attributions.

<!-- LINKS -->

[docs]: https://pkg.go.dev/github.com/secureworks/errors
[golang-ci]: https://github.com/golangci/golangci-lint
[Apache-2.0]: https://choosealicense.com/licenses/apache-2.0/
