# d3 helm chart

Minimal install (after building and publishing an image, or loading one into your cluster):

```bash
helm install my-d3 ./helm/d3 \
  --set image.repository=YOUR_REGISTRY/d3 \
  --set image.tag=YOUR_TAG
```

With an existing admin credentials Secret (YAML key `admin_user` as in [admin-credentials.dev.yaml](./admin-credentials.dev.yaml)):

```bash
helm install my-d3 ./helm/d3 \
  --set image.repository=YOUR_REGISTRY/d3 \
  --set image.tag=YOUR_TAG \
  --set admin.existingSecret=your-admin-secret \
  --set admin.autoGenerate=false
```

Use `helm show values ./helm/d3` for options (external Redis, ingress, persistence sizes, `environment`, and more).