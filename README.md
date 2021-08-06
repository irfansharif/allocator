Allocator
---

### Contributing

Because of the C++ dependencies, the library is compiled/tested using
[Bazel](https://bazel.build). The top-level Makefiles packages most things
you'd need.

```sh
# supported bazel version >= 4.0.0

$ make help
Supported commands: build, test, generate, rewrite

$ make generate
--- tidying go.{mod,sum}
--- generating bazel files
ok

$ make build
ok

$ make test
...
INFO: Build completed successfully, 4 total actions
```

```sh
# to run a specific tests
bazel test ... --test_output=all \
  --cache_test_results=no \
  --test_arg='-test.v' \
  --test_filter='Test.*'
```
