package surveillance

import (
	"math"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	pkgsurv "github.com/wyfcoding/pkg/algos/surveillance"
)

func sampleSpoofingEvents(base time.Time) []MarketEvent {
	return []MarketEvent{
		{UserID: "u1", Type: "PLACE", Price: decimal.NewFromInt(100), Quantity: decimal.NewFromInt(120), Timestamp: base.Add(-50 * time.Second)},
		{UserID: "u1", Type: "PLACE", Price: decimal.NewFromInt(101), Quantity: decimal.NewFromInt(130), Timestamp: base.Add(-45 * time.Second)},
		{UserID: "u1", Type: "PLACE", Price: decimal.NewFromInt(102), Quantity: decimal.NewFromInt(110), Timestamp: base.Add(-40 * time.Second)},
		{UserID: "u1", Type: "CANCEL", Price: decimal.NewFromInt(100), Quantity: decimal.NewFromInt(120), Timestamp: base.Add(-30 * time.Second)},
		{UserID: "u1", Type: "CANCEL", Price: decimal.NewFromInt(101), Quantity: decimal.NewFromInt(130), Timestamp: base.Add(-20 * time.Second)},
		{UserID: "u1", Type: "CANCEL", Price: decimal.NewFromInt(102), Quantity: decimal.NewFromInt(110), Timestamp: base.Add(-10 * time.Second)},
	}
}

func toPkgSpoofingEvents(in []MarketEvent) []pkgsurv.MarketEvent {
	out := make([]pkgsurv.MarketEvent, 0, len(in))
	for _, e := range in {
		out = append(out, pkgsurv.MarketEvent{
			Price:     e.Price,
			Quantity:  e.Quantity,
			Timestamp: e.Timestamp,
			UserID:    e.UserID,
			Type:      e.Type,
		})
	}
	return out
}

func TestSpoofingEngineConsistency(t *testing.T) {
	base := time.Now()
	events := sampleSpoofingEvents(base)
	pkgEvents := toPkgSpoofingEvents(events)

	local := &Engine{Threshold: decimal.NewFromInt(100), Window: 5 * time.Minute}
	pkg := &pkgsurv.Engine{Threshold: decimal.NewFromInt(100), Window: 5 * time.Minute}

	localScore, localReason := local.Analyze(events)
	pkgScore, pkgReason := pkg.Analyze(pkgEvents)

	if math.Abs(localScore-pkgScore) > 1e-9 {
		t.Fatalf("score mismatch: local=%f pkg=%f", localScore, pkgScore)
	}
	if localReason != pkgReason {
		t.Fatalf("reason mismatch: local=%q pkg=%q", localReason, pkgReason)
	}

	emptyLocalScore, emptyLocalReason := local.Analyze(nil)
	emptyPkgScore, emptyPkgReason := pkg.Analyze(nil)
	if math.Abs(emptyLocalScore-emptyPkgScore) > 1e-9 {
		t.Fatalf("empty score mismatch: local=%f pkg=%f", emptyLocalScore, emptyPkgScore)
	}
	if emptyLocalReason != emptyPkgReason {
		t.Fatalf("empty reason mismatch: local=%q pkg=%q", emptyLocalReason, emptyPkgReason)
	}
}

func BenchmarkSpoofingLocalAnalyze(b *testing.B) {
	events := sampleSpoofingEvents(time.Now())
	engine := &Engine{Threshold: decimal.NewFromInt(100), Window: 5 * time.Minute}
	for i := 0; i < b.N; i++ {
		_, _ = engine.Analyze(events)
	}
}

func BenchmarkSpoofingPkgAnalyze(b *testing.B) {
	events := toPkgSpoofingEvents(sampleSpoofingEvents(time.Now()))
	engine := &pkgsurv.Engine{Threshold: decimal.NewFromInt(100), Window: 5 * time.Minute}
	for i := 0; i < b.N; i++ {
		_, _ = engine.Analyze(events)
	}
}
