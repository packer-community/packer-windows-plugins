TEST?=./...

default: test

bin:
	@sh -c "$(CURDIR)/scripts/build.sh"

dev:
	@TF_DEV=1 sh -c "$(CURDIR)/scripts/build.sh"

test:
	go test $(TEST) $(TESTARGS) -timeout=10s

updatedeps:
	go get -d -v -p 2 ./...

.PHONY: bin default dev test updatedeps
