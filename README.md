# simpledb

学習目的で RDBMS をゼロから自作するプロジェクト。
外部のデータベースライブラリには頼らず、ストレージ層からクエリ実行までを Go で実装していく。

## ステータス

Phase 0 の途中。`CREATE TABLE` と `SELECT` は、SQL の解析からメモリ上の実行まで対応している。
`INSERT` と SQL 入力用の CLI は未実装で、`go run .` はまだ `simpledb: hello` を表示する。

できていること、次の目標、今後のロードマップは [TODO](docs/todo.md) にまとめている。

## 技術スタック

- 言語: Go 1.23
- 実行環境: Docker（任意）

### 設計メモ（早めに固定する前提）

- 型は MVP では `INT` と `TEXT` の 2 つに絞る。
- ページサイズは 4KB 固定。
- AST / 内部値表現は Phase 0 で決める（全 Phase に波及するため丁寧に）。
