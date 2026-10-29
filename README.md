# simple-rdbms

学習目的で RDBMS をゼロから自作するプロジェクト。
外部のデータベースライブラリには頼らず、ストレージ層からクエリ実行までを Go で実装していく。

## 現在の構成

- `lexer` / `parser`: SQL をトークンと AST に変換する。
- `value`: `INT` / `TEXT` の内部型と、型付きの値を定義する。
- `engine`: 列名と型を持つメモリ上のテーブル、および CREATE / INSERT / SELECT / DELETE の実行を担当する。

DELETE は `DELETE FROM テーブル名 [WHERE 条件]` に対応する。WHERE は SELECT と同じ比較・AND / OR・括弧に対応し、条件に一致する行をすべて削除する。WHERE 省略時は全行削除する。条件全体を検証してから行を置き換え、エラー時は既存データを変更しない。テーブルと列定義は保持する。

SELECT の WHERE は比較演算子・AND / OR・括弧に対応し、フルスキャンで条件に合う行を取得する。列の存在と比較する値の型は、行の走査前に条件全体で検証する。対応範囲は [WHERE の仕様](docs/specs/where.md) に記載する。

CREATE の型名は実行器で内部型に変換し、SELECT は値と型を保持して返す。
INSERT は単一行・全列に対応し、列の定義順に型付きの値を保存する。
テーブル不在・列数不一致・型不一致はエラーとし、失敗時は既存データを変更しない。暗黙の型変換は行わない。
SQL 入力用 CLI・ファイル永続化は未実装。実装状況は [docs/todo.md](docs/todo.md)、型の設計は [ADR 0003](docs/adr/0003-typed-values-and-columns.md) に記載する。
