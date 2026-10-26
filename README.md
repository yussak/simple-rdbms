# simple-rdbms

学習目的で RDBMS をゼロから自作するプロジェクト。
外部のデータベースライブラリには頼らず、ストレージ層からクエリ実行までを Go で実装していく。

## 現在の構成

- `lexer` / `parser`: SQL をトークンと AST に変換する。
- `value`: `INT` / `TEXT` の内部型と、型付きの値を定義する。
- `engine`: 列名と型を持つメモリ上のテーブル、および CREATE / SELECT の実行を担当する。

CREATE の型名は実行器で内部型に変換し、SELECT は値と型を保持して返す。
INSERT は単一行・全列の構文を解析し、テーブル名と型付きの値を持つ AST を返す。
INSERT の実行・SQL 入力用 CLI・ファイル永続化は未実装。実装状況は [docs/todo.md](docs/todo.md)、型の設計は [ADR 0003](docs/adr/0003-typed-values-and-columns.md) に記載する。
