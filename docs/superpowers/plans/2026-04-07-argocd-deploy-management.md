# ArgoCD デプロイ管理導入実装計画

1. ArgoCD 導入に必要な spec を追加する
2. レイアウトテストに ArgoCD 関連ファイルと Make ターゲットの期待値を追加する
3. `manifest/argocd`, `manifest/infra-bundle`, `manifest/app-bundle` を追加する
4. `argocd/applications` に `infra-core`, `keda-operator`, `sample-app` の静的 Application を追加する
5. `Makefile` と `README.md` に ArgoCD 導線を追加する
6. `TODO.md` に ArgoCD 導入手順の完了条件を追加する
7. `go test ./sample-app/layout`, `make test`, `helm dependency build`, `helm template` で検証する
