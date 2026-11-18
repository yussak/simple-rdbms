# 永続化を作りながら学ぶメモ

この文書には、最初のゴールである CREATE → INSERT → SELECT → 閉じる → 開き直す → SELECT に向けて、確認できた仕組みと次に調べる問いを記録する。設計は [永続化の設計](../design/persistence.md) に分ける。

## 現在の CREATE・INSERT・SELECT

例: `CREATE TABLE users (id INT, name TEXT)` の後に `(1, 'Alice')` を INSERT して SELECT する。

```text
CREATE の AST（テーブル名、列名、型名）
  → executeCreate: 型名を value.Type に変換
  → Database.tables["users"] = &Table{Columns: [...], Rows: []}

INSERT の AST（テーブル名、型付きの値）
  → executeInsert: tables から Table を取得し、列数と型を検証
  → 値のスライスをコピーして Table.Rows に追加

SELECT の AST（テーブル名、選択列、WHERE）
  → executeSelect: tables から Table を取得
  → Table.Rows を走査・フィルタし、列を選んで Result を返す
```

- `Database.tables` はテーブル名から `*Table` へのメモリ上のマップ。`Table.Columns` は列名と内部型を順番に保持し、`Table.Rows` は型付きの行を保持する（`engine/table.go`）。
- CREATE は AST の型名 `INT` / `TEXT` を内部型へ変換し、新しい `Table` をマップに登録する。同名テーブルを拒否する処理はまだなく、既存の登録を上書きする（`engine/executor.go` の `executeCreate`）。
- INSERT は列数と型を全て検証した後、AST の値スライスをコピーして `Rows` に追加する（`executeInsert`）。
- SELECT は `Rows` を読み、WHERE があれば走査中に絞り、`*` または指定列の結果を組み立てる（`executeSelect`、`engine/condition.go` の `filterRows`）。
- `engine/insert_integration_test.go` は同じ `Database` インスタンスで CREATE → INSERT → SELECT を確認する。別のインスタンスで開き直す結合テストも追加したが、開閉APIが未実装のため `t.Skip` としている（`engine/persistence_integration_test.go`）。

現在はファイルを開く処理がないため、`NewDatabase()` を再度呼ぶと空のマップができ、前のテーブル定義と行は見つからない。最初の永続化では、両方を保存・復元する経路が必要になる。

## PostgreSQL での保存先とたどり方

- テーブル名などの情報はシステムカタログに保存される。`pg_class` はテーブルの名前と識別子、通常のテーブルのファイル名に対応する `relfilenode` を持つ。`pg_attribute` は、そのテーブルに属する各列の名前・型・列番号を持つ。カタログ自身もテーブルとして保存される（[システムカタログ](https://www.postgresql.org/docs/current/catalogs.html)、[pg_class](https://www.postgresql.org/docs/current/catalog-pg-class.html)、[pg_attribute](https://www.postgresql.org/docs/current/catalog-pg-attribute.html)）。
- 標準の保存先では、データベースごとのディレクトリが `PGDATA/base/データベースのOID` にある。通常のテーブルはそれぞれ別のファイルを持ち、ファイル名には `relfilenode` を使う。カタログもそのデータベースのディレクトリに保存される（[ファイル構成](https://www.postgresql.org/docs/current/storage-file-layout.html)）。
- テーブルのファイルは固定長ページの並び。ページ内には行本体と、その位置・長さを示す項目識別子がある。行を指す位置はページ番号と項目識別子の番号で表せる（[ページ構成](https://www.postgresql.org/docs/current/storage-page-layout.html)）。

`users(id INT, name TEXT)` の行を再オープン後に読む流れを、ここまでの資料から概念的にたどると次のようになる。

```text
テーブル名 users
  → カタログでテーブルと列定義を確認
  → テーブルのファイルを特定
  → ファイルのページを読む
  → ページ内の行を取り出し、列定義に従って値として解釈する
```

これは PostgreSQL の保存構造から整理した流れであり、起動時のカタログ読み込み順や内部の呼び出し手順を示すものではない。この調査では、カタログ自身を起動時に見つける方法、ページの正確なバイト配置、WAL・キャッシュの詳細までは扱っていない。

## 最初のゴールを1行でたどる

このプロジェクトで `CREATE TABLE users (id INT, name TEXT);`、`INSERT INTO users VALUES (1, 'Alice');` を実行する場合に、永続化後に必要となる流れを示す。この図は動作の順序を示し、ファイル構成は[永続化の設計](../design/persistence.md)に記録する。

```text
CREATE TABLE users (id INT, name TEXT)
  → 既存のパーサーがテーブル名・列名・型名を AST にする
  → 実行器が型名を内部型に変換する
  → 保存層が「users / 1列目 id INT / 2列目 name TEXT」を永続化する
                                      ↓
INSERT INTO users VALUES (1, 'Alice')
  → 実行器が保存済みの列定義を使って列数と型を検証する
  → 保存層が「INT 1 / TEXT Alice」の行をデータページに永続化する
                                      ↓
SELECT * FROM users
  → 列定義とデータページを参照して行を値に戻し、結果を返す
                                      ↓
DBを閉じる → 新しいDBとして開き直す
  → 永続化した「users / 列の順序・名前・型」を読み、行の保存先を見つける
                                      ↓
SELECT * FROM users
  → データページから行を読み、列定義に従って INT 1 / TEXT Alice に戻す
  → Columns: [id, name], Rows: [[1, Alice]] を返す
```

開き直した後の SELECT が成功するには、列定義と行の両方を読み出せる必要がある。さらに、列定義からそのテーブルの行の保存先をたどれる必要がある。ページのバイト配置は、この図だけでは確定しない。

## 実装中に調べる問い

- 列定義と行を、それぞれのファイルのどのページに置くか。
- ページ内で行の位置と長さをどう表すか。
- 型付きの INT・TEXT をどのバイト列に変換するか。
