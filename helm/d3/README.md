# d3 helm chart

Minimal install (after building and publishing an image, or loading one into your cluster):

```bash
helm install my-d3 ./helm/d3 \
  --set image.repository=YOUR_REGISTRY/d3 \
  --set image.tag=YOUR_TAG
```

With an existing admin credentials Secret (YAML key `admin_user` as in the repo file [admin-credentials.dev.yaml](../../admin-credentials.dev.yaml)):

```bash
helm install my-d3 ./helm/d3 \
  --set image.repository=YOUR_REGISTRY/d3 \
  --set image.tag=YOUR_TAG \
  --set admin.existingSecret=your-admin-secret \
  --set admin.autoGenerate=false
```

Use `helm show values ./helm/d3` for options (external Redis, ingress, persistence sizes, `environment`, and more).

## Local checks

CI runs chart-testing (`ct lint`), `helm unittest`, and kubeconform on every `helm/d3/ci/*-values.yaml` overlay. Locally:

- `task helm:all` — unittest + kubeconform (kubeconform binary is cached under `.cache/kubeconform`).
- `task helm:lint` — same as `ct lint --config ct.yaml --all` (install `ct` plus Python packages `yamale` and `yamllint`, e.g. in a venv).
