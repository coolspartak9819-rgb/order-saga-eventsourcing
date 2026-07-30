# Kubernetes

`edgegate.yaml` contains a self-contained demo deployment: two EdgeGate
replicas, Redis, two upstream services, resource limits, probes, an HPA and a
PodDisruptionBudget.

The GitHub Actions workflow publishes the application image to:

```text
ghcr.io/coolspartak9819-rgb/edgegate-node
```

Before applying the manifests, replace values in `edgegate-secrets` and use an
external Redis service for a production environment.

```bash
kubectl apply -f deploy/k8s/edgegate.yaml
kubectl -n edgegate rollout status deployment/edgegate
kubectl -n edgegate get pods,hpa,pdb
```

`tls.yaml` is optional. It expects nginx-ingress and cert-manager to be
installed. Replace the email and hostname before applying it:

```bash
kubectl apply -f deploy/k8s/tls.yaml
```
