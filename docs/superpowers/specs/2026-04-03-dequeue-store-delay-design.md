# Dequeue Store Delay

## 目的

`dequeue-app` がメッセージを受信したあと、PostgreSQL に保存する直前に任意秒数だけ待機できるようにする。

今回の変更では、待機時間を環境変数で制御し、未設定時は既存挙動を維持する。

## スコープ

- `sample-app/internal/dequeue`
- `sample-app/internal/config`
- `sample-app/cmd/dequeue`
- `manifest/dequeue-app`
- `compose.yaml`
- 関連ユニットテスト

PostgreSQL のスキーマ変更や enqueue 側の挙動変更は対象外とする。

## 設計

### 設定

`DEQUEUE_STORE_DELAY_SECONDS` を `dequeue-app` 用の環境変数として追加する。

- 未設定時は `0` 秒
- 0 以上の整数秒を受け付ける
- 数値でない場合は設定エラーにする

`config.Dequeue` では `time.Duration` として保持し、呼び出し側で再解釈しないようにする。

### dequeue の処理順

`Worker.RunOnce` の処理順は次のようにする。

1. メッセージを受信する
2. メッセージ本文を decode する
3. 保存用の時刻を決める
4. 設定された待機時間だけ待つ
5. DB に保存する
6. キューから削除する

待機位置は「DB 保存の直前」とし、メッセージ受信前や削除前全体を遅らせる設計にはしない。

### 待機の実装

本番コードでは `time.Sleep` 相当の待機を行うが、テストでは実時間待ちを避ける。

そのため `Worker` に待機処理の差し替え口を追加し、通常時は `context` を見ながら指定 `Duration` を待つ実装を使う。

これにより、ユニットテストでは「何秒待とうとしたか」を即時に検証できる。

### デプロイ設定

Helm chart の `manifest/dequeue-app` と `compose.yaml` に `DEQUEUE_STORE_DELAY_SECONDS` を追加できるようにする。

- chart の default は `"0"`
- 開発者が values override で `"5"` を渡せるようにする
- compose でも同じ環境変数名を使う

## テスト

先に次のユニットテストを追加する。

- `LoadDequeue` が `DEQUEUE_STORE_DELAY_SECONDS` を読み取り、`time.Duration` に変換する
- `LoadDequeue` が不正な数値を拒否する
- `Worker.RunOnce` が DB 保存前に設定された待機時間で待機処理を呼ぶ
- 待機時間が `0` のときは待機処理を呼ばない

既存の保存内容と削除処理の検証は維持する。

## エラーハンドリング

- `DEQUEUE_STORE_DELAY_SECONDS` が不正なら起動時に失敗する
- 待機中に `context` がキャンセルされたら保存せずに失敗を返す
- DB 保存失敗時の扱いは既存のまま変えない

## 非目標

- ミリ秒単位の遅延設定
- enqueue 側への同種設定追加
- KEDA の polling interval や scaling 条件の変更
