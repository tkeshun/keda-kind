# ArgoCD Secret 配置

- 実 Secret は `*.local.yaml` としてこのディレクトリに置く
- `argocd/secrets/*.local.yaml` は `.gitignore` 済み
- GitHub App の private key は Git に commit しない

例:

- `argocd/secrets/github-repository.local.yaml`
