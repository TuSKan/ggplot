package arrow

import (
	"github.com/TuSKan/ggplot/internal/dataset"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
)

type stringIterator struct {
	chunked  *arrow.Chunked
	chunkIdx int
	idx      int
}

// Next iterates chunked arrays safely wrapping Dictionary unpacking boundaries.
func (s *stringIterator) Next() (string, bool, bool) {
	if s.chunkIdx >= len(s.chunked.Chunks()) {
		return "", false, false
	}
	chunk := s.chunked.Chunk(s.chunkIdx)
	if s.idx >= chunk.Len() {
		s.chunkIdx++
		s.idx = 0
		return s.Next() // Recursively jump boundaries
	}

	isNull := chunk.IsNull(s.idx)
	var val string
	if !isNull {
		switch arr := chunk.(type) {
		case *array.String:
			val = arr.Value(s.idx)
		case *array.Dictionary:
			if dict, ok := arr.Dictionary().(*array.String); ok {
				val = dict.Value(arr.GetValueIndex(s.idx))
			} else {
				val = "unsupported_dict_value"
			}
		default:
			val = "unsupported_string_val"
		}
	}
	s.idx++
	return val, isNull, true
}

func (c *TableColumn) Strings() (dataset.StringIterator, error) {
	return &stringIterator{chunked: c.chunked}, nil
}

// Compile assertion verifying Iterator logic implements protocol natively.
var _ dataset.IterableColumn = (*TableColumn)(nil)
