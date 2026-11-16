package storage

import (
	"encoding/binary"
	"fmt"
)

const (
	PageSize           = 8192
	headerSize         = 8
	slotSize           = 4
	pageMagic          = 0x31474150 // "PAG1" をリトルエンディアンで格納する。
	deletedMask uint16 = 1 << 15
)

// Page は固定長ページ内の行の配置を管理する。
type Page struct {
	data [PageSize]byte
}

func NewPage() *Page {
	p := &Page{}
	binary.LittleEndian.PutUint32(p.data[0:4], pageMagic)
	binary.LittleEndian.PutUint16(p.data[4:6], headerSize)
	binary.LittleEndian.PutUint16(p.data[6:8], PageSize)
	return p
}

func ParsePage(data []byte) (*Page, error) {
	if len(data) != PageSize {
		return nil, fmt.Errorf("ページ長が不正です: %d", len(data))
	}
	p := &Page{}
	copy(p.data[:], data)
	if binary.LittleEndian.Uint32(p.data[0:4]) != pageMagic {
		return nil, fmt.Errorf("ページの識別子が不正です")
	}
	lower, upper := p.bounds()
	if lower < headerSize || (lower-headerSize)%slotSize != 0 || upper < lower || upper > PageSize {
		return nil, fmt.Errorf("ページの空き領域境界が不正です")
	}
	for i := 0; i < (lower-headerSize)/slotSize; i++ {
		offset, length := p.slot(i)
		if length == 0 || offset < upper || offset+length > PageSize {
			return nil, fmt.Errorf("ページの項目識別子 %d が不正です", i)
		}
	}
	return p, nil
}

func (p *Page) bounds() (int, int) {
	return int(binary.LittleEndian.Uint16(p.data[4:6])), int(binary.LittleEndian.Uint16(p.data[6:8]))
}

func (p *Page) slot(i int) (int, int) {
	start := headerSize + i*slotSize
	return int(binary.LittleEndian.Uint16(p.data[start : start+2])), int(binary.LittleEndian.Uint16(p.data[start+2:start+4]) &^ deletedMask)
}

func (p *Page) slotDeleted(i int) bool {
	start := headerSize + i*slotSize + 2
	return binary.LittleEndian.Uint16(p.data[start:start+2])&deletedMask != 0
}

func (p *Page) CanAppend(length int) bool {
	lower, upper := p.bounds()
	return length > 0 && length <= PageSize-headerSize-slotSize && upper-lower >= length+slotSize
}

func (p *Page) Append(row []byte) error {
	if !p.CanAppend(len(row)) {
		return fmt.Errorf("ページに行を格納できません: %d バイト", len(row))
	}
	lower, upper := p.bounds()
	offset := upper - len(row)
	copy(p.data[offset:upper], row)
	binary.LittleEndian.PutUint16(p.data[lower:lower+2], uint16(offset))
	binary.LittleEndian.PutUint16(p.data[lower+2:lower+4], uint16(len(row)))
	binary.LittleEndian.PutUint16(p.data[4:6], uint16(lower+slotSize))
	binary.LittleEndian.PutUint16(p.data[6:8], uint16(offset))
	return nil
}

func (p *Page) Rows() [][]byte {
	lower, _ := p.bounds()
	rows := make([][]byte, 0, (lower-headerSize)/slotSize)
	for i := 0; i < (lower-headerSize)/slotSize; i++ {
		if p.slotDeleted(i) {
			continue
		}
		offset, length := p.slot(i)
		row := make([]byte, length)
		copy(row, p.data[offset:offset+length])
		rows = append(rows, row)
	}
	return rows
}

// MarkDeleted は条件に合う行の項目識別子に削除フラグを立てる。
// 条件の評価が失敗した場合は、このページを変更しない。
func (p *Page) MarkDeleted(match func([]byte) (bool, error)) (int, error) {
	lower, _ := p.bounds()
	var targets []int
	for i := 0; i < (lower-headerSize)/slotSize; i++ {
		if p.slotDeleted(i) {
			continue
		}
		offset, length := p.slot(i)
		row := make([]byte, length)
		copy(row, p.data[offset:offset+length])
		matched, err := match(row)
		if err != nil {
			return 0, err
		}
		if matched {
			targets = append(targets, i)
		}
	}
	for _, i := range targets {
		start := headerSize + i*slotSize + 2
		length := binary.LittleEndian.Uint16(p.data[start : start+2])
		binary.LittleEndian.PutUint16(p.data[start:start+2], length|deletedMask)
	}
	return len(targets), nil
}

func (p *Page) Bytes() []byte { return p.data[:] }
