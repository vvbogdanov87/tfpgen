.PHONY: install
install: # Compile and install tfpgen
	go install

.PHONY: generate-provider
generate-provider: # Generate provider
	./tests/scripts/generate-provider.sh

.PHONY: test
test: # Run chainsaw tests
	chainsaw test ./tests

.PHONY: install-crossplane
install-crossplane: # Install Crossplane
	./tests/scripts/install_crossplane.sh

.PHONY: apply-common-manifests
apply-common-manifests: # Apply common manifests
	kubectl apply -f ./tests/manifests/01_k8s_provider.yaml
	sleep 10
	kubectl apply -f ./tests/manifests/02_k8s_provider_config.yaml

.PHONY: test-local
test-local: install generate-provider install-crossplane apply-common-manifests test