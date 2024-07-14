.PHONY: install
install:
	go install

.PHONY: generate
generate:
	cd ./tools/generator && go run .

.PHONY: test
test:
	chainsaw test ./tests