package engine

import (
	"errors"
	"fmt"
	"sort"

	"simple-rdbms/storage"
	"simple-rdbms/value"
)

var tableCatalogTypes = []value.Type{value.TypeInt, value.TypeText, value.TypeInt}
var columnCatalogTypes = []value.Type{value.TypeInt, value.TypeInt, value.TypeText, value.TypeInt}

// OpenDatabase は既存のDBディレクトリを開き、存在しなければ作る。
func OpenDatabase(path string) (*Database, error) {
	store, err := storage.Open(path)
	if err != nil {
		return nil, err
	}
	db := &Database{tables: make(map[string]*Table), store: store, nextRelationID: 3}
	if err := db.loadCatalogs(); err != nil {
		return nil, errors.Join(err, store.Close())
	}
	return db, nil
}

// Close はDBを閉じる。書き込みは各操作の成功前に同期済みである。
func (db *Database) Close() error {
	if db.closed {
		return fmt.Errorf("DBは既に閉じています")
	}
	db.closed = true
	if db.store != nil {
		return db.store.Close()
	}
	return nil
}

func (db *Database) loadCatalogs() error {
	tableRows, err := db.store.ReadAll(1)
	if err != nil {
		return fmt.Errorf("テーブルカタログを読めません: %w", err)
	}
	ids := make(map[int64]bool)
	for _, raw := range tableRows {
		values, err := storage.DecodeRow(tableCatalogTypes, raw)
		if err != nil {
			return fmt.Errorf("テーブルカタログの行が不正です: %w", err)
		}
		id, name, fileID := values[0].Int, values[1].Text, values[2].Int
		if id < 3 || fileID != id || name == "" || ids[id] || db.tables[name] != nil {
			return fmt.Errorf("テーブルカタログに不正または重複した定義があります")
		}
		ids[id] = true
		if id >= db.nextRelationID {
			db.nextRelationID = id + 1
		}
		if fileID >= db.nextRelationID {
			db.nextRelationID = fileID + 1
		}
		db.tables[name] = &Table{relationID: fileID}
	}
	columnRows, err := db.store.ReadAll(2)
	if err != nil {
		return fmt.Errorf("列カタログを読めません: %w", err)
	}
	columns := make(map[int64]map[int64]Column)
	for _, raw := range columnRows {
		values, err := storage.DecodeRow(columnCatalogTypes, raw)
		if err != nil {
			return fmt.Errorf("列カタログの行が不正です: %w", err)
		}
		id, number, name := values[0].Int, values[1].Int, values[2].Text
		if !ids[id] || number < 1 || name == "" || (values[3].Int != int64(value.TypeInt) && values[3].Int != int64(value.TypeText)) {
			return fmt.Errorf("列カタログに不正な定義があります")
		}
		typ := value.Type(values[3].Int)
		if columns[id] == nil {
			columns[id] = make(map[int64]Column)
		}
		if _, exists := columns[id][number]; exists {
			return fmt.Errorf("列カタログに重複した列番号があります")
		}
		columns[id][number] = Column{Name: name, Type: typ}
	}
	for name, table := range db.tables {
		// 初版ではrelation IDとファイル番号を同じ値に割り当てる。
		byNumber := columns[table.relationID]
		if len(byNumber) == 0 {
			return fmt.Errorf("テーブル %q の列定義がありません", name)
		}
		numbers := make([]int, 0, len(byNumber))
		for number := range byNumber {
			numbers = append(numbers, int(number))
		}
		sort.Ints(numbers)
		for i, number := range numbers {
			if number != i+1 {
				return fmt.Errorf("テーブル %q の列番号が連続していません", name)
			}
			table.Columns = append(table.Columns, byNumber[int64(number)])
		}
		if err := db.store.ValidateRelation(table.relationID); err != nil {
			return fmt.Errorf("テーブル %q のrelationを確認できません: %w", name, err)
		}
	}
	return nil
}

func (db *Database) createPersistentTable(name string, columns []Column) error {
	if _, exists := db.tables[name]; exists {
		return fmt.Errorf("テーブル %q は既に存在します", name)
	}
	if len(columns) == 0 {
		return fmt.Errorf("列のないテーブルは作成できません")
	}
	id := db.nextRelationID
	tableRow, err := storage.EncodeRow(tableCatalogTypes, []value.Value{
		{Type: value.TypeInt, Int: id}, {Type: value.TypeText, Text: name}, {Type: value.TypeInt, Int: id},
	})
	if err != nil {
		return err
	}
	columnRows := make([][]byte, 0, len(columns))
	for i, col := range columns {
		row, err := storage.EncodeRow(columnCatalogTypes, []value.Value{
			{Type: value.TypeInt, Int: id}, {Type: value.TypeInt, Int: int64(i + 1)},
			{Type: value.TypeText, Text: col.Name}, {Type: value.TypeInt, Int: int64(col.Type)},
		})
		if err != nil {
			return err
		}
		columnRows = append(columnRows, row)
	}
	// 入力由来のエラーは、ファイルを作る前にすべて検出する。
	if err := db.store.CreateRelation(id); err != nil {
		return err
	}
	if err := db.store.Append(1, tableRow); err != nil {
		return err
	}
	for _, row := range columnRows {
		if err := db.store.Append(2, row); err != nil {
			return err
		}
	}
	db.tables[name] = &Table{Columns: columns, relationID: id}
	db.nextRelationID++
	return nil
}

func (db *Database) insertPersistentRow(table *Table, values []value.Value) error {
	types := make([]value.Type, len(table.Columns))
	for i, col := range table.Columns {
		types[i] = col.Type
	}
	row, err := storage.EncodeRow(types, values)
	if err != nil {
		return err
	}
	return db.store.Append(table.relationID, row)
}

func (db *Database) readPersistentRows(table *Table) ([][]value.Value, error) {
	types := make([]value.Type, len(table.Columns))
	for i, col := range table.Columns {
		types[i] = col.Type
	}
	rawRows, err := db.store.ReadAll(table.relationID)
	if err != nil {
		return nil, err
	}
	rows := make([][]value.Value, 0, len(rawRows))
	for _, raw := range rawRows {
		row, err := storage.DecodeRow(types, raw)
		if err != nil {
			return nil, fmt.Errorf("テーブルの行が不正です: %w", err)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (db *Database) deletePersistentRows(table *Table, condition rowCondition) error {
	types := make([]value.Type, len(table.Columns))
	for i, col := range table.Columns {
		types[i] = col.Type
	}
	_, err := db.store.DeleteWhere(table.relationID, func(raw []byte) (bool, error) {
		row, err := storage.DecodeRow(types, raw)
		if err != nil {
			return false, err
		}
		return condition == nil || condition.matches(row), nil
	})
	return err
}
