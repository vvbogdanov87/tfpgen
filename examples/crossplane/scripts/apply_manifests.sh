#!/bin/bash

# Create a secret with AWS credentials for the provider
kubectl create secret \
    generic aws-secret \
    -n crossplane-system \
    --from-file=creds=$HOME/.aws/credentials

# Apply all the manifests in order
for f in ./manifests/*.yaml; do
  kubectl apply -f "$f"
done