package arrow

import (
	"github.com/TuSKan/ggplot/dataset"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
)

// --- Float64 Iterator ---

type float64Iterator struct {
	chunked  *arrow.Chunked
	chunkIdx int
	idx      int
}

func (it *float64Iterator) Next() (float64, bool, bool) {
	for it.chunkIdx < len(it.chunked.Chunks()) {
		chunk := it.chunked.Chunk(it.chunkIdx)
		if it.idx >= chunk.Len() {
			it.chunkIdx++
			it.idx = 0
			continue
		}
		isNull := chunk.IsNull(it.idx)
		var val float64
		if !isNull {
			switch arr := chunk.(type) {
			case *array.Float64:
				val = arr.Value(it.idx)
			case *array.Float32:
				val = float64(arr.Value(it.idx))
			case *array.Int64:
				val = float64(arr.Value(it.idx))
			case *array.Int32:
				val = float64(arr.Value(it.idx))
			case *array.Dictionary:
				idx := arr.GetValueIndex(it.idx)
				switch dict := arr.Dictionary().(type) {
				case *array.Float64:
					val = dict.Value(idx)
				case *array.Float32:
					val = float64(dict.Value(idx))
				case *array.Int64:
					val = float64(dict.Value(idx))
				case *array.Int32:
					val = float64(dict.Value(idx))
				}
			}
		}
		it.idx++
		return val, isNull, true
	}
	return 0, false, false
}

func (c *TableColumn) Float64s() (dataset.Float64Iter, error) {
	return &float64Iterator{chunked: c.chunked}, nil
}

// --- Int64 Iterator ---

type int64Iterator struct {
	chunked  *arrow.Chunked
	chunkIdx int
	idx      int
}

func (it *int64Iterator) Next() (int64, bool, bool) {
	for it.chunkIdx < len(it.chunked.Chunks()) {
		chunk := it.chunked.Chunk(it.chunkIdx)
		if it.idx >= chunk.Len() {
			it.chunkIdx++
			it.idx = 0
			continue
		}
		isNull := chunk.IsNull(it.idx)
		var val int64
		if !isNull {
			switch arr := chunk.(type) {
			case *array.Int64:
				val = arr.Value(it.idx)
			case *array.Int32:
				val = int64(arr.Value(it.idx))
			case *array.Float64:
				val = int64(arr.Value(it.idx))
			case *array.Float32:
				val = int64(arr.Value(it.idx))
			case *array.Dictionary:
				idx := arr.GetValueIndex(it.idx)
				switch dict := arr.Dictionary().(type) {
				case *array.Int64:
					val = dict.Value(idx)
				case *array.Int32:
					val = int64(dict.Value(idx))
				case *array.Float64:
					val = int64(dict.Value(idx))
				}
			}
		}
		it.idx++
		return val, isNull, true
	}
	return 0, false, false
}

func (c *TableColumn) Int64s() (dataset.Int64Iter, error) {
	return &int64Iterator{chunked: c.chunked}, nil
}

// --- String Iterator ---

type stringIterator struct {
	chunked  *arrow.Chunked
	chunkIdx int
	idx      int
}

func (it *stringIterator) Next() (string, bool, bool) {
	for it.chunkIdx < len(it.chunked.Chunks()) {
		chunk := it.chunked.Chunk(it.chunkIdx)
		if it.idx >= chunk.Len() {
			it.chunkIdx++
			it.idx = 0
			continue
		}
		isNull := chunk.IsNull(it.idx)
		var val string
		if !isNull {
			switch arr := chunk.(type) {
			case *array.String:
				val = arr.Value(it.idx)
			case *array.Dictionary:
				if dict, ok := arr.Dictionary().(*array.String); ok {
					val = dict.Value(arr.GetValueIndex(it.idx))
				}
			}
		}
		it.idx++
		return val, isNull, true
	}
	return "", false, false
}

func (c *TableColumn) Strings() (dataset.StringIter, error) {
	return &stringIterator{chunked: c.chunked}, nil
}
