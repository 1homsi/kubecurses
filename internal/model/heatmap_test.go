package model

import (
	"reflect"
	"testing"
)

func TestHeatmapPlanRows(t *testing.T) {
	tests := []struct {
		name    string
		n       int
		maxCols int
		want    []int
	}{
		{name: "zero nodes", n: 0, maxCols: 4, want: nil},
		{name: "single node", n: 1, maxCols: 4, want: []int{1}},
		{name: "exact single row", n: 4, maxCols: 4, want: []int{4}},
		{name: "n < maxCols single row", n: 3, maxCols: 5, want: []int{3}},
		{name: "n=7 compact shoulder", n: 7, maxCols: 4, want: []int{2, 3, 2}},
		{name: "n=10 avoids pyramid", n: 10, maxCols: 4, want: []int{2, 3, 3, 2}},
		{name: "n=12 balanced plateau", n: 12, maxCols: 4, want: []int{3, 3, 3, 3}},
		{name: "n=19 compact smooth", n: 19, maxCols: 5, want: []int{3, 4, 5, 4, 3}},
		{name: "n=2 two nodes", n: 2, maxCols: 4, want: []int{2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HeatmapPlanRows(tt.n, tt.maxCols)
			// Verify shape matches expected.
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HeatmapPlanRows(%d, %d) = %v, want %v", tt.n, tt.maxCols, got, tt.want)
			}
			// Verify capacity >= n.
			total := 0
			for _, r := range got {
				total += r
			}
			if total < tt.n {
				t.Errorf("HeatmapPlanRows(%d, %d): capacity %d < n", tt.n, tt.maxCols, total)
			}
			// Verify no row exceeds maxCols.
			for i, r := range got {
				if r > tt.maxCols {
					t.Errorf("HeatmapPlanRows(%d, %d): row %d width %d > maxCols", tt.n, tt.maxCols, i, r)
				}
			}
			if tt.n > 1 && len(got) > 1 && got[len(got)-1] == 1 {
				t.Errorf("HeatmapPlanRows(%d, %d): orphan last row %v", tt.n, tt.maxCols, got)
			}
		})
	}
}

func TestHeatmapPlanRowsMinNodeSilhouette(t *testing.T) {
	tests := []struct {
		n    int
		want []int
	}{
		{7, []int{2, 3, 2}},
		{8, []int{2, 2, 2, 2}},
		{9, []int{3, 3, 3}},
		{10, []int{2, 3, 3, 2}},
		{11, []int{2, 2, 3, 2, 2}},
		{12, []int{3, 3, 3, 3}},
		{13, []int{2, 3, 3, 3, 2}},
		{14, []int{2, 2, 3, 3, 2, 2}},
		{15, []int{2, 2, 2, 3, 2, 2, 2}},
	}

	for _, tt := range tests {
		got := HeatmapPlanRowsMin(tt.n, 4, 2)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("HeatmapPlanRowsMin(%d, 4, 2) = %v, want %v", tt.n, got, tt.want)
		}
	}
}

func TestHeatmapPlanSymmetry(t *testing.T) {
	for n := 1; n <= 50; n++ {
		for maxCols := 1; maxCols <= 8; maxCols++ {
			plan := HeatmapPlanRows(n, maxCols)
			if plan == nil && n > 0 {
				t.Errorf("HeatmapPlanRows(%d, %d) returned nil for n>0", n, maxCols)
				continue
			}
			// Check symmetry: plan[i] == plan[len-1-i]
			for i := 0; i < len(plan)/2; i++ {
				j := len(plan) - 1 - i
				if plan[i] != plan[j] {
					t.Errorf("HeatmapPlanRows(%d, %d) = %v: not symmetric (plan[%d]=%d != plan[%d]=%d)",
						n, maxCols, plan, i, plan[i], j, plan[j])
					break
				}
			}
			// Check monotone increase to center.
			mid := len(plan) / 2
			for i := 1; i <= mid; i++ {
				if plan[i] < plan[i-1] {
					t.Errorf("HeatmapPlanRows(%d, %d) = %v: not monotone increasing at index %d", n, maxCols, plan, i)
					break
				}
			}
		}
	}
}

func TestHeatmapNodeToRowColPlan(t *testing.T) {
	plan := []int{2, 3, 2} // 7 nodes total
	tests := []struct {
		nodeIdx int
		wantRow int
		wantCol int
	}{
		{0, 0, 0},
		{1, 0, 1},
		{2, 1, 0},
		{3, 1, 1},
		{4, 1, 2},
		{5, 2, 0},
		{6, 2, 1},
	}
	for _, tt := range tests {
		row, col := HeatmapNodeToRowColPlan(plan, tt.nodeIdx)
		if row != tt.wantRow || col != tt.wantCol {
			t.Errorf("HeatmapNodeToRowColPlan(plan, %d) = (%d,%d), want (%d,%d)",
				tt.nodeIdx, row, col, tt.wantRow, tt.wantCol)
		}
	}
}

func TestHeatmapRowColToNodePlan(t *testing.T) {
	plan := []int{2, 3, 2}
	tests := []struct {
		row, col int
		want     int
	}{
		{0, 0, 0},
		{0, 1, 1},
		{1, 0, 2},
		{1, 2, 4},
		{2, 0, 5},
		{2, 1, 6},
	}
	for _, tt := range tests {
		got := HeatmapRowColToNodePlan(plan, tt.row, tt.col)
		if got != tt.want {
			t.Errorf("HeatmapRowColToNodePlan(plan, %d, %d) = %d, want %d",
				tt.row, tt.col, got, tt.want)
		}
	}
}

func TestHeatmapPlanRoundTrip(t *testing.T) {
	// Verify NodeToRowCol and RowColToNode are inverses for a given plan.
	plan := HeatmapPlanRows(19, 5) // [3,4,5,4,3]
	total := 0
	for _, r := range plan {
		total += r
	}
	for nodeIdx := 0; nodeIdx < total; nodeIdx++ {
		row, col := HeatmapNodeToRowColPlan(plan, nodeIdx)
		got := HeatmapRowColToNodePlan(plan, row, col)
		if got != nodeIdx {
			t.Errorf("round-trip failed for nodeIdx=%d: got %d (row=%d,col=%d)", nodeIdx, got, row, col)
		}
	}
}
