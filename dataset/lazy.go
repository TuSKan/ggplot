// lazy.go implements the deferred execution infrastructure for Dataset.
//
// Every Dataset verb builds a linked chain of op nodes instead of executing
// immediately. Materialisation happens only when Collect(ctx) is called.
// The executeOps function walks the chain and dispatches to exec* methods
// that contain the actual engine logic (relocated from the old eager verbs).
package dataset

import (
	"context"
	"fmt"
)

// opKind identifies the type of operation in a lazy Dataset chain.
type opKind int

const (
	opNone        opKind = iota // root node — wraps a materialised Table
	opSelect                    // keep named columns
	opRename                    // rename one column
	opFilter                    // mask-based row filtering
	opArrange                   // sort by columns
	opHead                      // first N rows
	opTail                      // last N rows
	opSlice                     // row range [start, end)
	opDistinct                  // deduplicate
	opJoin                      // any join type
	opPivotLonger               // wide → long
	opPivotWider                // long → wide
	opSeparate                  // split column
	opFill                      // fill missing values
	opDropNA                    // drop rows with NA
	opStack                     // row-bind
	opCombine                   // column-bind
	opGroupBy                   // group-by + summarize (compound)
	opMutate                    // add/replace column
	opReplaceCol                // replace column with float64 values
)

// op holds the parameters for a single lazy operation.
// This is a union struct — only the fields relevant to the opKind are used.
type op struct {
	kind opKind

	// Shared
	cols []string // select, arrange, distinct, groupby, dropNA

	// Filter
	mask Masker

	// Head / Tail
	n int

	// Slice
	start, end int

	// Join
	joinOther Table
	joinSpec  JoinSpec

	// Reshape
	pivotL PivotLongerSpec
	pivotW PivotWiderSpec
	sepCol string
	into   []string
	sep    string

	// Fill
	fillCol string
	fillDir FillDirection

	// Stack / Combine
	others []Table

	// GroupBy + Summarize
	aggSpecs []AggSpec

	// Mutate
	mutName string
	mutFn   MutateFunc

	// Rename
	renameOld string
	renameNew string

	// ReplaceCol
	replaceCol  string
	replaceVals []float64
}

// root walks the parent chain to find the root Dataset (the one with a Table).
func (f *Dataset) root() *Dataset {
	cur := f
	for cur.parent != nil {
		cur = cur.parent
	}
	return cur
}

// flatten walks the parent chain and returns the operations in execution
// order (root-first). The root node (opNone) is excluded.
// Pre-counts chain length for exact capacity; builds in-order to skip reversal.
func (f *Dataset) flatten() []op {
	// Count chain length.
	n := 0
	for cur := f; cur != nil && cur.op.kind != opNone; cur = cur.parent {
		n++
	}
	if n == 0 {
		return nil
	}
	// Fill from tail to head (root-first order).
	chain := make([]op, n)
	i := n - 1
	for cur := f; cur != nil && cur.op.kind != opNone; cur = cur.parent {
		chain[i] = cur.op
		i--
	}
	return chain
}

// Collected reports whether the Dataset has been materialised.
func (f Dataset) Collected() bool {
	return f.tbl != nil
}

// Collect materialises the lazy operation chain, returning a new Dataset
// with the result Table populated. If already materialised, returns self.
//
// This is the single materialisation boundary — all data access must go
// through a collected Dataset.
func (f Dataset) Collect(ctx context.Context) (Dataset, error) {
	if f.tbl != nil {
		return f, f.err
	}
	if f.err != nil {
		return f, f.err
	}

	// Find root table.
	root := f.root()
	if root.tbl == nil {
		return f, fmt.Errorf("dataset: no root table in lazy chain")
	}

	// Flatten verb chain.
	ops := f.flatten()
	if len(ops) == 0 {
		return Dataset{eng: f.eng, tbl: root.tbl}, nil
	}

	// Let engine optimise if it supports it.
	eng := f.eng
	if opt, ok := eng.(Optimizer); ok {
		ops = opt.Optimize(ops)
	}

	// Execute operations sequentially.
	tbl, err := executeOps(ctx, eng, root.tbl, ops)
	if err != nil {
		return Dataset{eng: eng, err: err}, err
	}
	return Dataset{eng: eng, tbl: tbl}, nil
}

// executeOps runs a sequence of operations against a starting Table.
func executeOps(ctx context.Context, eng Engine, tbl Table, ops []op) (Table, error) {
	cur := Dataset{eng: eng, tbl: tbl}
	for _, o := range ops {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		switch o.kind {
		case opSelect:
			cur = cur.execSelect(o.cols)
		case opRename:
			cur = cur.execRename(o.renameOld, o.renameNew)
		case opFilter:
			cur = cur.execFilter(o.mask)
		case opArrange:
			cur = cur.execArrange(o.cols)
		case opHead:
			cur = cur.execHead(o.n)
		case opTail:
			cur = cur.execTail(o.n)
		case opSlice:
			cur = cur.execSlice(o.start, o.end)
		case opDistinct:
			cur = cur.execDistinct(o.cols)
		case opJoin:
			cur = cur.execJoin(o.joinOther, o.joinSpec)
		case opPivotLonger:
			cur = cur.execPivotLonger(o.pivotL)
		case opPivotWider:
			cur = cur.execPivotWider(o.pivotW)
		case opSeparate:
			cur = cur.execSeparate(o.sepCol, o.into, o.sep)
		case opFill:
			cur = cur.execFill(o.fillCol, o.fillDir)
		case opDropNA:
			cur = cur.execDropNA(o.cols)
		case opStack:
			cur = cur.execStack(o.others)
		case opCombine:
			cur = cur.execCombine(o.others)
		case opGroupBy:
			cur = cur.execGroupBy(o.cols, o.aggSpecs)
		case opMutate:
			cur = cur.execMutate(o.mutName, o.mutFn)
		case opReplaceCol:
			cur = cur.execReplaceCol(o.replaceCol, o.replaceVals)
		}
		if cur.err != nil {
			return nil, cur.err
		}
	}
	return cur.tbl, nil
}
