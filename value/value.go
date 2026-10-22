// Package value は、データベースで扱う型と値を定義する。
package value

// Type は値や列のデータ型を表す。
type Type uint8

const (
	TypeInvalid Type = iota // 型を指定していないゼロ値
	TypeInt
	TypeText
)

// Value は型付きの値を表す。Type に応じて Int または Text を使用する。
type Value struct {
	Type Type
	Int  int64
	Text string
}
