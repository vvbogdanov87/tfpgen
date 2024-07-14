.PHONY: install
install:
	go install

.PHONY: test
test:
	chainsaw test ./tests