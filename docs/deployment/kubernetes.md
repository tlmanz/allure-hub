# Kubernetes

## Prerequisites

- Kubernetes 1.24+
- A storage class that supports `ReadWriteMany` (e.g. NFS, CephFS, EFS)
- An ingress controller (e.g. ingress-nginx)

## Deploy

```bash
# Apply all manifests in dependency order
make k8s-apply

# Tear down
make k8s-delete
```

Or apply manually:

```bash
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/pvc.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/ingress.yaml
```

## Manifests overview

| File | Description |
|---|---|
| `namespace.yaml` | `allure-hub` namespace |
| `pvc.yaml` | `ReadWriteMany` PVC for `/data` |
| `configmap.yaml` | Non-secret environment variables |
| `deployment.yaml` | Single-pod deployment |
| `service.yaml` | ClusterIP service on port 8080 |
| `ingress.yaml` | Ingress with TLS termination |

## Required changes before deploying

### `k8s/ingress.yaml`

Replace the placeholder hostname with your actual domain:

```yaml
spec:
  rules:
    - host: allure.example.com   # ← your domain
```

### Secrets

Create a Kubernetes secret for sensitive values:

```bash
kubectl create secret generic allure-hub-secrets \
  --namespace allure-hub \
  --from-literal=SESSION_SECRET="$(openssl rand -hex 32)" \
  --from-literal=GOOGLE_CLIENT_ID="your-client-id" \
  --from-literal=GOOGLE_CLIENT_SECRET="your-client-secret"
```

Reference the secret in `deployment.yaml`:

```yaml
env:
  - name: SESSION_SECRET
    valueFrom:
      secretKeyRef:
        name: allure-hub-secrets
        key: SESSION_SECRET
  - name: GOOGLE_CLIENT_ID
    valueFrom:
      secretKeyRef:
        name: allure-hub-secrets
        key: GOOGLE_CLIENT_ID
  - name: GOOGLE_CLIENT_SECRET
    valueFrom:
      secretKeyRef:
        name: allure-hub-secrets
        key: GOOGLE_CLIENT_SECRET
```

## Storage

The PVC uses `ReadWriteMany`. Ensure your storage class supports it:

```yaml
# k8s/pvc.yaml
accessModes:
  - ReadWriteMany
```

If you only have `ReadWriteOnce` (single-node), change this to `ReadWriteOnce` — it works as long as you run a single replica.

## Health check

The deployment uses `/api/healthz` for liveness and readiness probes:

```yaml
livenessProbe:
  httpGet:
    path: /api/healthz
    port: 8080
readinessProbe:
  httpGet:
    path: /api/healthz
    port: 8080
```

## RBAC policy in Kubernetes

Mount `policy.yaml` from a ConfigMap:

```bash
kubectl create configmap allure-hub-policy \
  --namespace allure-hub \
  --from-file=policy.yaml=./backend/policy.yaml
```

```yaml
# In deployment.yaml
volumeMounts:
  - name: policy
    mountPath: /app/policy.yaml
    subPath: policy.yaml

volumes:
  - name: policy
    configMap:
      name: allure-hub-policy

# Set env var
env:
  - name: AUTH_POLICY_FILE
    value: /app/policy.yaml
```

The policy is hot-reloaded every 30 seconds — update the ConfigMap and it takes effect without a pod restart.
