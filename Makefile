.PHONY: install
install: # Compile and install tfpgen
	go install

.PHONY: generate-provider
generate-provider: # Generate provider
	./scripts/generate-provider.sh

.PHONY: test
test: # Run chainsaw tests
	chainsaw test ./tests

.PHONY: test-local
test-local: install generate-provider test
