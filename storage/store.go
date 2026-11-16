package storage

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

const controlName = "CONTROL"

var controlMagic = [8]byte{'S', 'R', 'D', 'B', 'M', 'S', '0', '1'}

// Store は1つのDBディレクトリを扱う。各操作でファイルを開き、更新後に同期する。
type Store struct {
	path   string
	closed bool
}

func Open(path string) (*Store, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		if err := os.Mkdir(path, 0700); err != nil {
			return nil, fmt.Errorf("DBディレクトリを作れません: %w", err)
		}
		if err := os.Mkdir(filepath.Join(path, "relations"), 0700); err != nil {
			return nil, fmt.Errorf("relationsディレクトリを作れません: %w", err)
		}
		control := make([]byte, 16)
		copy(control[:8], controlMagic[:])
		binary.LittleEndian.PutUint32(control[8:12], 1)
		binary.LittleEndian.PutUint32(control[12:16], PageSize)
		f, err := os.OpenFile(filepath.Join(path, controlName), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return nil, fmt.Errorf("CONTROLを作れません: %w", err)
		}
		_, writeErr := f.Write(control)
		if writeErr == nil {
			writeErr = f.Sync()
		}
		closeErr := f.Close()
		if writeErr != nil {
			return nil, errors.Join(fmt.Errorf("CONTROLを書けません: %w", writeErr), closeErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("CONTROLを閉じられません: %w", closeErr)
		}
		s := &Store{path: path}
		for _, id := range []int64{1, 2} {
			if err := s.CreateRelation(id); err != nil {
				return nil, err
			}
		}
		if err := syncDir(filepath.Join(path, "relations")); err != nil {
			return nil, err
		}
		if err := syncDir(path); err != nil {
			return nil, err
		}
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("DBディレクトリを確認できません: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("DBのパスはディレクトリではありません")
	}
	control, err := os.ReadFile(filepath.Join(path, controlName))
	if err != nil {
		return nil, fmt.Errorf("CONTROLを読めません: %w", err)
	}
	if len(control) != 16 || string(control[:8]) != string(controlMagic[:]) || binary.LittleEndian.Uint32(control[8:12]) != 1 || binary.LittleEndian.Uint32(control[12:16]) != PageSize {
		return nil, fmt.Errorf("CONTROLの形式またはバージョンが不正です")
	}
	return &Store{path: path}, nil
}

func (s *Store) relationPath(id int64) string {
	return filepath.Join(s.path, "relations", strconv.FormatInt(id, 10))
}

func (s *Store) checkOpen() error {
	if s.closed {
		return fmt.Errorf("DBは既に閉じています")
	}
	return nil
}

func (s *Store) CreateRelation(id int64) error {
	if err := s.checkOpen(); err != nil {
		return err
	}
	if id < 1 {
		return fmt.Errorf("relation番号が不正です: %d", id)
	}
	f, err := os.OpenFile(s.relationPath(id), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("relation %d を作れません: %w", id, err)
	}
	if err := f.Sync(); err != nil {
		return errors.Join(fmt.Errorf("relation %d を同期できません: %w", id, err), f.Close())
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("relation %d を閉じられません: %w", id, err)
	}
	return syncDir(filepath.Join(s.path, "relations"))
}

func (s *Store) ReadAll(id int64) (rows [][]byte, retErr error) {
	if err := s.checkOpen(); err != nil {
		return nil, err
	}
	f, err := os.Open(s.relationPath(id))
	if err != nil {
		return nil, fmt.Errorf("relation %d を開けません: %w", id, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("relation %d を閉じられません: %w", id, err))
		}
	}()
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("relation %d を調べられません: %w", id, err)
	}
	if info.Size()%PageSize != 0 {
		return nil, fmt.Errorf("relation %d のサイズがページ長の倍数ではありません", id)
	}
	buf := make([]byte, PageSize)
	for offset := int64(0); offset < info.Size(); offset += PageSize {
		if _, err := io.ReadFull(f, buf); err != nil {
			return nil, fmt.Errorf("relation %d のページを読めません: %w", id, err)
		}
		page, err := ParsePage(buf)
		if err != nil {
			return nil, fmt.Errorf("relation %d のページ %d: %w", id, offset/PageSize, err)
		}
		rows = append(rows, page.Rows()...)
	}
	return rows, nil
}

// ValidateRelation は行を読み込まず、ファイルの存在とページ長を確認する。
func (s *Store) ValidateRelation(id int64) error {
	if err := s.checkOpen(); err != nil {
		return err
	}
	info, err := os.Stat(s.relationPath(id))
	if err != nil {
		return fmt.Errorf("relation %d を調べられません: %w", id, err)
	}
	if !info.Mode().IsRegular() || info.Size()%PageSize != 0 {
		return fmt.Errorf("relation %d のファイル形式が不正です", id)
	}
	return nil
}

func (s *Store) Append(id int64, row []byte) (retErr error) {
	if err := s.checkOpen(); err != nil {
		return err
	}
	if len(row) == 0 || len(row) > PageSize-headerSize-slotSize {
		return fmt.Errorf("行が1ページに収まりません: %d バイト", len(row))
	}
	f, err := os.OpenFile(s.relationPath(id), os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("relation %d を開けません: %w", id, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("relation %d を閉じられません: %w", id, err))
		}
	}()
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("relation %d を調べられません: %w", id, err)
	}
	if info.Size()%PageSize != 0 {
		return fmt.Errorf("relation %d のサイズがページ長の倍数ではありません", id)
	}
	offset := info.Size()
	page := NewPage()
	if offset > 0 {
		buf := make([]byte, PageSize)
		if _, err := f.ReadAt(buf, offset-PageSize); err != nil {
			return fmt.Errorf("relation %d の末尾ページを読めません: %w", id, err)
		}
		page, err = ParsePage(buf)
		if err != nil {
			return fmt.Errorf("relation %d の末尾ページが不正です: %w", id, err)
		}
		if page.CanAppend(len(row)) {
			offset -= PageSize
		} else {
			page = NewPage()
		}
	}
	if err := page.Append(row); err != nil {
		return err
	}
	if n, err := f.WriteAt(page.Bytes(), offset); err != nil {
		return fmt.Errorf("relation %d のページを書けません: %w", id, err)
	} else if n != PageSize {
		return fmt.Errorf("relation %d のページを最後まで書けません", id)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("relation %d を同期できません: %w", id, err)
	}
	return nil
}

// DeleteWhere は全ページの条件評価を終えてから、該当ページを書き戻す。
// 書き戻し途中のI/O失敗やクラッシュによる部分更新は、この段階では復旧しない。
func (s *Store) DeleteWhere(id int64, match func([]byte) (bool, error)) (deleted int, retErr error) {
	if err := s.checkOpen(); err != nil {
		return 0, err
	}
	f, err := os.OpenFile(s.relationPath(id), os.O_RDWR, 0)
	if err != nil {
		return 0, fmt.Errorf("relation %d を開けません: %w", id, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("relation %d を閉じられません: %w", id, err))
		}
	}()
	info, err := f.Stat()
	if err != nil {
		return 0, fmt.Errorf("relation %d を調べられません: %w", id, err)
	}
	if info.Size()%PageSize != 0 {
		return 0, fmt.Errorf("relation %d のサイズがページ長の倍数ではありません", id)
	}
	type changedPage struct {
		offset int64
		page   *Page
	}
	var changed []changedPage
	for offset := int64(0); offset < info.Size(); offset += PageSize {
		buf := make([]byte, PageSize)
		if _, err := f.ReadAt(buf, offset); err != nil {
			return 0, fmt.Errorf("relation %d のページを読めません: %w", id, err)
		}
		page, err := ParsePage(buf)
		if err != nil {
			return 0, fmt.Errorf("relation %d のページ %d: %w", id, offset/PageSize, err)
		}
		count, err := page.MarkDeleted(match)
		if err != nil {
			return 0, fmt.Errorf("relation %d のページ %d: %w", id, offset/PageSize, err)
		}
		if count > 0 {
			deleted += count
			changed = append(changed, changedPage{offset: offset, page: page})
		}
	}
	for _, item := range changed {
		if n, err := f.WriteAt(item.page.Bytes(), item.offset); err != nil {
			return 0, fmt.Errorf("relation %d のページを書けません: %w", id, err)
		} else if n != PageSize {
			return 0, fmt.Errorf("relation %d のページを最後まで書けません", id)
		}
	}
	if len(changed) > 0 {
		if err := f.Sync(); err != nil {
			return 0, fmt.Errorf("relation %d を同期できません: %w", id, err)
		}
	}
	return deleted, nil
}

func (s *Store) Close() error {
	if err := s.checkOpen(); err != nil {
		return err
	}
	s.closed = true
	return nil
}

func syncDir(path string) (retErr error) {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("ディレクトリを開けません: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("ディレクトリを閉じられません: %w", err))
		}
	}()
	if err := f.Sync(); err != nil {
		return fmt.Errorf("ディレクトリを同期できません: %w", err)
	}
	return nil
}
