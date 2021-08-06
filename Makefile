help:
	@echo "Supported commands: build, test, generate"

build:
	@bazel build ...
	@echo "ok"

test:
	@bazel test --test_output=all --test_arg='-test.v' ...
	@echo "ok"

bzl: FORCE 
	@echo "--- generating bazel files"
	@echo "build --cxxopt=-std=c++17 --experimental_convenience_symlinks=ignore --define gotags=bazel" > $@.tmp
	@echo "test --cxxopt=-std=c++17 --define gotags=bazel" >> $@.tmp
	@echo "try-import %workspace%/.bazelrc.user" >> $@.tmp
	@mv $@.tmp .bazelrc
	@bazel run //:gazelle -- update-repos \
		-from_file=go.mod -prune=true \
		-build_file_proto_mode=disable_global \
		-to_macro=DEPS.bzl%go_deps &> /dev/null
	@bazel run //:gazelle &> /dev/null

gogenerate:
	@echo "--- tidying go.{mod,sum}"
	@go mod tidy

generate: gogenerate bzl
	@echo "ok"
	
FORCE: ;
