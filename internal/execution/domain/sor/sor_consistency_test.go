package sor

import (
	"sort"
	"testing"

	"github.com/shopspring/decimal"
	pkgopt "github.com/wyfcoding/pkg/algos/optimization"
)

func sampleSORInput() (decimal.Decimal, []RouteInput) {
	totalQty := decimal.NewFromInt(120)
	inputs := []RouteInput{
		{VenueID: "EX_A", Price: decimal.NewFromInt(100), Quantity: decimal.NewFromInt(40), FeeRate: decimal.RequireFromString("0.001"), LatencyMs: 2.0},
		{VenueID: "EX_B", Price: decimal.NewFromInt(99), Quantity: decimal.NewFromInt(60), FeeRate: decimal.RequireFromString("0.002"), LatencyMs: 1.0},
		{VenueID: "EX_A", Price: decimal.NewFromInt(101), Quantity: decimal.NewFromInt(30), FeeRate: decimal.RequireFromString("0.001"), LatencyMs: 2.0},
		{VenueID: "EX_C", Price: decimal.NewFromInt(98), Quantity: decimal.NewFromInt(200), FeeRate: decimal.Zero, LatencyMs: 8.0},
	}
	return totalQty, inputs
}

func toPkgSORInput(in []RouteInput) []pkgopt.RouteInput {
	out := make([]pkgopt.RouteInput, 0, len(in))
	for _, i := range in {
		out = append(out, pkgopt.RouteInput{
			VenueID:   i.VenueID,
			Price:     i.Price,
			Quantity:  i.Quantity,
			FeeRate:   i.FeeRate,
			LatencyMs: i.LatencyMs,
		})
	}
	return out
}

func normalizeLocalSOR(in []RouteOutput) []RouteOutput {
	out := append([]RouteOutput(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i].VenueID < out[j].VenueID })
	return out
}

func normalizePkgSOR(in []pkgopt.RouteOutput) []pkgopt.RouteOutput {
	out := append([]pkgopt.RouteOutput(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i].VenueID < out[j].VenueID })
	return out
}

func assertSORResultEqual(t *testing.T, title string, local []RouteOutput, pkg []pkgopt.RouteOutput) {
	t.Helper()

	if len(local) != len(pkg) {
		t.Fatalf("%s len mismatch: local=%d pkg=%d", title, len(local), len(pkg))
	}
	for i := range local {
		if local[i].VenueID != pkg[i].VenueID {
			t.Fatalf("%s venue mismatch at %d: local=%s pkg=%s", title, i, local[i].VenueID, pkg[i].VenueID)
		}
		if !local[i].Quantity.Equal(pkg[i].Quantity) {
			t.Fatalf("%s qty mismatch at %d: local=%s pkg=%s", title, i, local[i].Quantity.String(), pkg[i].Quantity.String())
		}
		if !local[i].Price.Equal(pkg[i].Price) {
			t.Fatalf("%s price mismatch at %d: local=%s pkg=%s", title, i, local[i].Price.String(), pkg[i].Price.String())
		}
	}
}

func TestSORConsistency(t *testing.T) {
	totalQty, inputs := sampleSORInput()
	pkgInputs := toPkgSORInput(inputs)

	local := &SOROptimizer{LatencyFactor: 0.1}
	pkg := &pkgopt.SOROptimizer{LatencyFactor: 0.1}

	localBuy := normalizeLocalSOR(local.Optimize(totalQty, inputs, true))
	pkgBuy := normalizePkgSOR(pkg.Optimize(totalQty, pkgInputs, true))
	assertSORResultEqual(t, "buy", localBuy, pkgBuy)

	localSell := normalizeLocalSOR(local.Optimize(totalQty, inputs, false))
	pkgSell := normalizePkgSOR(pkg.Optimize(totalQty, pkgInputs, false))
	assertSORResultEqual(t, "sell", localSell, pkgSell)
}

func BenchmarkSORLocalOptimize(b *testing.B) {
	totalQty, inputs := sampleSORInput()
	opt := &SOROptimizer{LatencyFactor: 0.1}
	for i := 0; i < b.N; i++ {
		_ = opt.Optimize(totalQty, inputs, true)
	}
}

func BenchmarkSORPkgOptimize(b *testing.B) {
	totalQty, inputs := sampleSORInput()
	pkgInputs := toPkgSORInput(inputs)
	opt := &pkgopt.SOROptimizer{LatencyFactor: 0.1}
	for i := 0; i < b.N; i++ {
		_ = opt.Optimize(totalQty, pkgInputs, true)
	}
}
