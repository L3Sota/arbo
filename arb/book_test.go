package arb_test

import (
	"testing"

	"github.com/L3Sota/arbo/arb"
)

func BenchmarkGatherBooksP(b *testing.B) {
	arb.GatherBooksP()
}

func BenchmarkGatherBooks(b *testing.B) {
	arb.GatherBooks()
}
