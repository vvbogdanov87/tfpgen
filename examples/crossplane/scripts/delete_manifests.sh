#!/bin/bash

# Delete all the manifests in reverse order
for f in $(ls -r1 ./manifests/*.yaml); do
  kubectl delete -f "$f"
done

# Delete the secret with AWS credentials for the provider
kubectl delete secret aws-secret -n crossplane-system