TEST?=./...

default: test

bin:
	@sh -c "$(CURDIR)/scripts/build.sh"

dev:
	@TF_DEV=1 sh -c "$(CURDIR)/scripts/build.sh"

test:
	"$(CURDIR)/scripts/test.sh"

testrace:
	go test -race $(TEST) $(TESTARGS)

updatedeps:
	go get -d -v -p 2 ./...

install:
	@[ -d "${HOME}/.packer.d/plugins" ] || mkdir -p "${HOME}/.packer.d/plugins"
	@cp $(wildcard bin/*) "${HOME}/.packer.d/plugins"

.PHONY: bin default dev install test testrace updatedeps
