package arrow

import (
	"github.com/TuSKan/ggplot/pkg/dataset"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
)

type stringIterator struct {
	chunked  *arrow.Chunked
	chunkIdx int
	idx      int
}

// Next iterates chunked arrays wrapping Dictionary unpacking boundaries.
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

type float64Iterator struct {
	chunked  *arrow.Chunked
	chunkIdx int
	idx      int
}

func (s *float64Iterator) Next() (float64, bool, bool) {
	if s.chunkIdx >= len(s.chunked.Chunks()) {
		return 0, false, false
	}
	chunk := s.chunked.Chunk(s.chunkIdx)
	if s.idx >= chunk.Len() {
		s.chunkIdx++
		s.idx = 0
		return s.Next()
	}

	isNull := chunk.IsNull(s.idx)
	var val float64
	if !isNull {
		switch arr := chunk.(type) {
		case *array.Float64:
			val = arr.Value(s.idx)
		case *array.Int64:
			val = float64(arr.Value(s.idx))
		case *array.Float32:
			val = float64(arr.Value(s.idx))
		case *array.Int32:
			val = float64(arr.Value(s.idx))
		case *array.Dictionary:
			idx := arr.GetValueIndex(s.idx)
			switch dict := arr.Dictionary().(type) {
			case *array.Float64:
				val = dict.Value(idx)
			case *array.Int64:
				val = float64(dict.Value(idx))
			case *array.Float32:
				val = float64(dict.Value(idx))
			case *array.Int32:
				val = float64(dict.Value(idx))
			}
		}
	}
	s.idx++
	return val, isNull, true
}

func (c *TableColumn) Float64s() (dataset.Float64Iterator, error) {
	return &float64Iterator{chunked: c.chunked}, nil
}

type int64Iterator struct {
	chunked  *arrow.Chunked
	chunkIdx int
	idx      int
}

func (s *int64Iterator) Next() (int64, bool, bool) {
	if s.chunkIdx >= len(s.chunked.Chunks()) {
		return 0, false, false
	}
	chunk := s.chunked.Chunk(s.chunkIdx)
	if s.idx >= chunk.Len() {
		s.chunkIdx++
		s.idx = 0
		return s.Next()
	}

	isNull := chunk.IsNull(s.idx)
	var val int64
	if !isNull {
		switch arr := chunk.(type) {
		case *array.Int64:
			val = arr.Value(s.idx)
		case *array.Float64:
			val = int64(arr.Value(s.idx))
		case *array.Int32:
			val = int64(arr.Value(s.idx))
		case *array.Float32:
			val = int64(arr.Value(s.idx))
		case *array.Dictionary:
			idx := arr.GetValueIndex(s.idx)
			switch dict := arr.Dictionary().(type) {
			case *array.Int64:
				val = dict.Value(idx)
			case *array.Float64:
				val = int64(dict.Value(idx))
			}
		}
	}
	s.idx++
	return val, isNull, true
}

func (c *TableColumn) Int64s() (dataset.Int64Iterator, error) {
	return &int64Iterator{chunked: c.chunked}, nil
}

// Compile assertion verifying Iterator logic implements protocol.
var _ dataset.IterableColumn = (*TableColumn)(nil)
