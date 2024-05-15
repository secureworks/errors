# Errors

This package provides a suite of tools meant to work with Go 1.13 error 
wrapping and Go 1.20 multierrors. These helpers allow users to rely on
standard Go error patterns while they include some "missing pieces" or
additional features that are useful in practice.

> _Another errors package? Why does Go need all of these error libraries?_

Because the language and the standard library have a minimal approach to error 
handling that leaves out some primitives power users expect to have on 
hand.

Important among these primitives are stack traces, explicit error collection
types (multierrors and error groups) and error context management.

While Go 1.13 introduced error wrapping utilities and Go 1.20 added a minimal
multierror collection, it is really useful to have a few more tools on hand.

### Installation

> _This package **may not be used** in environments:_
>
> 1. &nbsp; _running Go 1.19 or lower;_
> 2. &nbsp; _running on 32-bit architecture;_
> 3. &nbsp; _running on Windows (**currently** not supported)._

Add the following to your file:

```go
import "github.com/secureworks/errors"
```

Then, when you run any Go command the toolchain will resolve and fetch the 
required modules automatically.

> If you are using Go 1.13 to Go 1.19, you should use the previous version of 
> this library, which has the same functionality but does not support the 
> specific form that Go 1.20 multierrors take:
>
> ```
> $ go get github.com/secureworks/errors@v0.1.2
> ```

Because this package re-exports the package-level functions of the standard 
library `"errors"` package, you do not need to include that package as well to 
get `New`, `As`, `Is`, `Unwrap`, and `Join`.

Note that `Join` is a special case: for consistency, and since our multierror is
a better implementation, `Join` returns our implementation (which uses our
formatting), not the standard library's implementation.

### Use

[Documentation is available on pkg.go.dev][docs]. You may also look at the 
examples in the package.

The most important features are listed below.

Package `github.com/secureworks/errors`:

- use in place of the standard library with no change in behavior;
- use the `errors.MultiError` type as either an explicit multierror 
  implementation or as an implicit multierror passed around with the default 
  `error` interface; use `errors.Join`, `errors.Append` and others to simplify
  multierror management in your code;
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
