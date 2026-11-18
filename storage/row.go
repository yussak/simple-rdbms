package storage

import (
	"encoding/binary"
	"fmt"
	"unicode/utf8"

	"simple-rdbms/value"
)

// EncodeRow は列定義順の値を、型タグを持たない行へ変換する。
func EncodeRow(types []value.Type, values []value.Value) ([]byte, error) {
	if len(types) != len(values) {
		return nil, fmt.Errorf("列数と値の数が一致しません")
	}
	var row []byte
	for i, typ := range types {
		v := values[i]
		if v.Type != typ {
			return nil, fmt.Errorf("列 %d の型が一致しません", i)
		}
		switch typ {
		case value.TypeInt:
			var buf [8]byte
			binary.LittleEndian.PutUint64(buf[:], uint64(v.Int))
			row = append(row, buf[:]...)
		case value.TypeText:
			if !utf8.ValidString(v.Text) {
				return nil, fmt.Errorf("列 %d のTEXTがUTF-8ではありません", i)
			}
			if uint64(len(v.Text)) > uint64(^uint32(0)) || len(row)+4+len(v.Text) > PageSize-headerSize-slotSize {
				return nil, fmt.Errorf("行が1ページに収まりません")
			}
			var buf [4]byte
			binary.LittleEndian.PutUint32(buf[:], uint32(len(v.Text)))
			row = append(row, buf[:]...)
			row = append(row, v.Text...)
		default:
			return nil, fmt.Errorf("未対応の列型: %d", typ)
		}
		if len(row) > PageSize-headerSize-slotSize {
			return nil, fmt.Errorf("行が1ページに収まりません")
		}
	}
	if len(row) == 0 {
		return nil, fmt.Errorf("空の行は保存できません")
	}
	return row, nil
}

// DecodeRow は保存済みの列定義を使い、行を型付きの値へ戻す。
func DecodeRow(types []value.Type, row []byte) ([]value.Value, error) {
	values := make([]value.Value, 0, len(types))
	pos := 0
	for i, typ := range types {
		switch typ {
		case value.TypeInt:
			if len(row)-pos < 8 {
				return nil, fmt.Errorf("列 %d のINTが途中で終わっています", i)
			}
			values = append(values, value.Value{Type: typ, Int: int64(binary.LittleEndian.Uint64(row[pos : pos+8]))})
			pos += 8
		case value.TypeText:
			if len(row)-pos < 4 {
				return nil, fmt.Errorf("列 %d のTEXT長が途中で終わっています", i)
			}
			length := binary.LittleEndian.Uint32(row[pos : pos+4])
			pos += 4
			if uint64(length) > uint64(len(row)-pos) {
				return nil, fmt.Errorf("列 %d のTEXTが途中で終わっています", i)
			}
			text := string(row[pos : pos+int(length)])
			if !utf8.ValidString(text) {
				return nil, fmt.Errorf("列 %d のTEXTがUTF-8ではありません", i)
			}
			values = append(values, value.Value{Type: typ, Text: text})
			pos += int(length)
		default:
			return nil, fmt.Errorf("未対応の列型: %d", typ)
		}
	}
	if pos != len(row) {
		return nil, fmt.Errorf("行に余分なデータがあります")
	}
	return values, nil
}
