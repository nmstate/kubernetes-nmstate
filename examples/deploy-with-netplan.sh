#!/bin/bash
#
# Example script to deploy kubernetes-nmstate with netplan backend
#
# Usage:
#   ./examples/deploy-with-netplan.sh

set -e

echo "=========================================="
echo "Deploying kubernetes-nmstate with netplan"
echo "=========================================="

# Build handler and operator images
echo "Building handler image..."
make handler

echo "Building operator image..."
make operator

# Deploy to cluster with netplan backend
echo "Deploying to cluster with netplan backend..."
BACKEND=netplan make cluster-sync

echo ""
echo "=========================================="
echo "Deployment complete!"
echo "=========================================="
echo ""
echo "Verify backend is set:"
echo "  kubectl get nmstate nmstate -o jsonpath='{.spec.backend}'"
echo ""
echo "Check handler pods:"
echo "  kubectl get pods -n nmstate -l component=kubernetes-nmstate-handler"
echo ""
echo "View handler logs:"
echo "  kubectl logs -n nmstate -l component=kubernetes-nmstate-handler -f"
echo ""
