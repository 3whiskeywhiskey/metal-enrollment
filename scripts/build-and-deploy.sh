#!/usr/bin/env bash
set -euo pipefail

# Build and deploy script for Metal Enrollment
# This script builds all components and deploys them to Kubernetes

REGISTRY="${REGISTRY:-metal-enrollment}"
NAMESPACE="${NAMESPACE:-metal-enrollment}"

echo "=== Metal Enrollment Build and Deploy ==="
echo "Registry: $REGISTRY"
echo "Namespace: $NAMESPACE"
echo ""

# Build Go binaries
echo "Building Go binaries..."
make build

# Build Docker images
echo "Building Docker images..."
make docker-build

# Tag images if using custom registry
if [ "$REGISTRY" != "metal-enrollment" ]; then
    echo "Tagging images for $REGISTRY..."
    docker tag metal-enrollment/server:latest $REGISTRY/server:latest
    docker tag metal-enrollment/builder:latest $REGISTRY/builder:latest
    docker tag metal-enrollment/ipxe-server:latest $REGISTRY/ipxe-server:latest

    echo "Pushing images to registry..."
    docker push $REGISTRY/server:latest
    docker push $REGISTRY/builder:latest
    docker push $REGISTRY/ipxe-server:latest
fi

# Deploy to Kubernetes
echo "Deploying to Kubernetes..."
make deploy

# Wait for pods to be ready
echo "Waiting for pods to be ready..."
kubectl wait --for=condition=ready pod -l app=enrollment-server -n $NAMESPACE --timeout=120s
kubectl wait --for=condition=ready pod -l app=enrollment-builder -n $NAMESPACE --timeout=120s
kubectl wait --for=condition=ready pod -l app=enrollment-ipxe-server -n $NAMESPACE --timeout=120s

echo ""
echo "=== Deployment Complete ==="
echo ""
echo "Services:"
kubectl get svc -n $NAMESPACE
echo ""
echo "Pods:"
kubectl get pods -n $NAMESPACE
echo ""
echo "To access the dashboard:"
echo "  kubectl port-forward -n $NAMESPACE svc/enrollment-server 8080:8080"
echo "  Then open http://localhost:8080"
echo ""
echo "Next steps:"
echo "  1. Build and deploy the registration image (see docs/SETUP.md)"
echo "  2. Configure your PXE infrastructure"
echo "  3. Boot a machine to test enrollment"
