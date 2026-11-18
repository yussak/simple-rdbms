package storage_test

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"simple-rdbms/storage"
	"simple-rdbms/value"
)

func TestRowRoundTripAndMalformedInput(t *testing.T) {
	types := []value.Type{value.TypeInt, value.TypeText}
	want := []value.Value{{Type: value.TypeInt, Int: -42}, {Type: value.TypeText, Text: "東京"}}
	data, err := storage.EncodeRow(types, want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := storage.DecodeRow(types, data)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("行を復元できない: got=%v err=%v", got, err)
	}
	if _, err := storage.DecodeRow(types, data[:len(data)-1]); err == nil {
		t.Fatal("途中で切れた行を受け入れた")
	}
	if _, err := storage.EncodeRow(types, []value.Value{want[0], {Type: value.TypeText, Text: strings.Repeat("x", storage.PageSize)}}); err == nil {
		t.Fatal("1ページを超える行を受け入れた")
	}
}

func TestRelationSpansPagesAndReopens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db")
	s, err := storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CreateRelation(3); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		if err := s.Append(3, []byte(strings.Repeat(string(rune('a'+i)), 1000))); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	rows, err := s.ReadAll(3)
	if err != nil || len(rows) != 20 {
		t.Fatalf("20行を読めない: count=%d err=%v", len(rows), err)
	}
	for i, row := range rows {
		if string(row) != strings.Repeat(string(rune('a'+i)), 1000) {
			t.Fatalf("行 %d の内容が違う", i)
		}
	}
}

func TestPageRejectsCorruptSlot(t *testing.T) {
	p := storage.NewPage()
	if err := p.Append([]byte("row")); err != nil {
		t.Fatal(err)
	}
	data := append([]byte(nil), p.Bytes()...)
	binary.LittleEndian.PutUint16(data[8:10], 0)
	if _, err := storage.ParsePage(data); err == nil {
		t.Fatal("不正な項目識別子を受け入れた")
	}
}

func TestOpenRejectsCorruptControl(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db")
	s, err := storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "CONTROL"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.Open(path); err == nil {
		t.Fatal("壊れたCONTROLを受け入れた")
	}
}

func TestDeleteWhereLeavesRowsUntouchedWhenLaterPageFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db")
	s, err := storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.CreateRelation(3); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		if err := s.Append(3, []byte(strings.Repeat("x", 1000))); err != nil {
			t.Fatal(err)
		}
	}
	seen := 0
	_, err = s.DeleteWhere(3, func([]byte) (bool, error) {
		seen++
		if seen == 10 {
			return false, errors.New("判定エラー")
		}
		return true, nil
	})
	if err == nil {
		t.Fatal("判定エラーを返さなかった")
	}
	rows, err := s.ReadAll(3)
	if err != nil || len(rows) != 20 {
		t.Fatalf("失敗時に行が変わった: count=%d err=%v", len(rows), err)
	}
}
