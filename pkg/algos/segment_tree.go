// Package algos - 线段树（Segment Tree）数据结构
package algos

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// SegmentTree 线段树
// 用于高效地处理区间查询和点更新问题
// 时间复杂度：构建 O(n)，查询 O(log n)，更新 O(log n)
type SegmentTree struct {
	tree []decimal.Decimal
	n    int
}

// NewSegmentTree 创建线段树
func NewSegmentTree(arr []decimal.Decimal) *SegmentTree {
	n := len(arr)
	st := &SegmentTree{
		tree: make([]decimal.Decimal, 4*n),
		n:    n,
	}
	st.build(arr, 0, 0, n-1)
	return st
}

// build 构建线段树
func (st *SegmentTree) build(arr []decimal.Decimal, node, start, end int) {
	if start == end {
		st.tree[node] = arr[start]
	} else {
		mid := (start + end) / 2
		leftChild := 2*node + 1
		rightChild := 2*node + 2
		st.build(arr, leftChild, start, mid)
		st.build(arr, rightChild, mid+1, end)
		st.tree[node] = st.tree[leftChild].Add(st.tree[rightChild])
	}
}

// Update 更新指定位置的值
func (st *SegmentTree) Update(index int, value decimal.Decimal) error {
	if index < 0 || index >= st.n {
		return fmt.Errorf("index out of range")
	}
	st.update(0, 0, st.n-1, index, value)
	return nil
}

// update 递归更新
func (st *SegmentTree) update(node, start, end, index int, value decimal.Decimal) {
	if start == end {
		st.tree[node] = value
	} else {
		mid := (start + end) / 2
		leftChild := 2*node + 1
		rightChild := 2*node + 2
		if index <= mid {
			st.update(leftChild, start, mid, index, value)
		} else {
			st.update(rightChild, mid+1, end, index, value)
		}
		st.tree[node] = st.tree[leftChild].Add(st.tree[rightChild])
	}
}

// Query 查询区间和
func (st *SegmentTree) Query(left, right int) (decimal.Decimal, error) {
	if left < 0 || right >= st.n || left > right {
		return decimal.Zero, fmt.Errorf("invalid range")
	}
	return st.query(0, 0, st.n-1, left, right), nil
}

// query 递归查询
func (st *SegmentTree) query(node, start, end, left, right int) decimal.Decimal {
	if right < start || end < left {
		return decimal.Zero
	}
	if left <= start && end <= right {
		return st.tree[node]
	}
	mid := (start + end) / 2
	leftChild := 2*node + 1
	rightChild := 2*node + 2
	leftSum := st.query(leftChild, start, mid, left, right)
	rightSum := st.query(rightChild, mid+1, end, left, right)
	return leftSum.Add(rightSum)
}

// RangeMaxSegmentTree 区间最大值线段树
type RangeMaxSegmentTree struct {
	tree []decimal.Decimal
	n    int
}

// NewRangeMaxSegmentTree 创建区间最大值线段树
func NewRangeMaxSegmentTree(arr []decimal.Decimal) *RangeMaxSegmentTree {
	n := len(arr)
	st := &RangeMaxSegmentTree{
		tree: make([]decimal.Decimal, 4*n),
		n:    n,
	}
	st.build(arr, 0, 0, n-1)
	return st
}

// build 构建线段树
func (st *RangeMaxSegmentTree) build(arr []decimal.Decimal, node, start, end int) {
	if start == end {
		st.tree[node] = arr[start]
	} else {
		mid := (start + end) / 2
		leftChild := 2*node + 1
		rightChild := 2*node + 2
		st.build(arr, leftChild, start, mid)
		st.build(arr, rightChild, mid+1, end)
		if st.tree[leftChild].GreaterThan(st.tree[rightChild]) {
			st.tree[node] = st.tree[leftChild]
		} else {
			st.tree[node] = st.tree[rightChild]
		}
	}
}

// Update 更新指定位置的值
func (st *RangeMaxSegmentTree) Update(index int, value decimal.Decimal) error {
	if index < 0 || index >= st.n {
		return fmt.Errorf("index out of range")
	}
	st.update(0, 0, st.n-1, index, value)
	return nil
}

// update 递归更新
func (st *RangeMaxSegmentTree) update(node, start, end, index int, value decimal.Decimal) {
	if start == end {
		st.tree[node] = value
	} else {
		mid := (start + end) / 2
		leftChild := 2*node + 1
		rightChild := 2*node + 2
		if index <= mid {
			st.update(leftChild, start, mid, index, value)
		} else {
			st.update(rightChild, mid+1, end, index, value)
		}
		if st.tree[leftChild].GreaterThan(st.tree[rightChild]) {
			st.tree[node] = st.tree[leftChild]
		} else {
			st.tree[node] = st.tree[rightChild]
		}
	}
}

// Query 查询区间最大值
func (st *RangeMaxSegmentTree) Query(left, right int) (decimal.Decimal, error) {
	if left < 0 || right >= st.n || left > right {
		return decimal.Zero, fmt.Errorf("invalid range")
	}
	return st.query(0, 0, st.n-1, left, right), nil
}

// query 递归查询
func (st *RangeMaxSegmentTree) query(node, start, end, left, right int) decimal.Decimal {
	if right < start || end < left {
		return decimal.NewFromInt(-9999999)
	}
	if left <= start && end <= right {
		return st.tree[node]
	}
	mid := (start + end) / 2
	leftChild := 2*node + 1
	rightChild := 2*node + 2
	leftMax := st.query(leftChild, start, mid, left, right)
	rightMax := st.query(rightChild, mid+1, end, left, right)
	if leftMax.GreaterThan(rightMax) {
		return leftMax
	}
	return rightMax
}
