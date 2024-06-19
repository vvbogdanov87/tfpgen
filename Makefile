.PHONY: install
install:
	go install

.PHONY: generate
generate:
	cd ./tools/generator && go run .