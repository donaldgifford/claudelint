package engine_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/config"
	"github.com/donaldgifford/claudelint/internal/engine"
	_ "github.com/donaldgifford/claudelint/internal/rules/all"
)

// benchArtifact implements artifact.Artifact with in-memory source
// bytes. Benchmarks avoid disk I/O by synthesizing artifacts directly;
// the aim is to measure engine throughput, not filesystem speed.
type benchArtifact struct {
	path   string
	kind   artifact.ArtifactKind
	source []byte
}

func (b *benchArtifact) Kind() artifact.ArtifactKind { return b.kind }
func (b *benchArtifact) Path() string                { return b.path }
func (b *benchArtifact) Source() []byte              { return b.source }

func synthArtifacts(n int) []artifact.Artifact {
	body := []byte(strings.Repeat("some markdown body text filling a page.\n", 20))
	out := make([]artifact.Artifact, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, &benchArtifact{
			path:   fmt.Sprintf("gen/%d/CLAUDE.md", i),
			kind:   artifact.KindClaudeMD,
			source: body,
		})
	}
	return out
}

// BenchmarkRunCLAUDEMDSmall measures a realistic small-repo run.
func BenchmarkRunCLAUDEMDSmall(b *testing.B) {
	arts := synthArtifacts(100)
	cfg := config.Resolve(nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.New(cfg).Run(arts, nil)
	}
}

// BenchmarkRunCLAUDEMDLarge measures a larger run that should expose
// contention in the worker pool. 10k files mirrors the upper-bound
// target from IMPL-0001 Phase 1.8.
func BenchmarkRunCLAUDEMDLarge(b *testing.B) {
	arts := synthArtifacts(10000)
	cfg := config.Resolve(nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.New(cfg).Run(arts, nil)
	}
}

// BenchmarkRunWorkerScaling compares 1-worker vs GOMAXPROCS so we can
// detect scheduler regressions that would show up as flat scaling.
func BenchmarkRunWorkerScaling(b *testing.B) {
	arts := synthArtifacts(1000)
	cfg := config.Resolve(nil)

	b.Run("workers=1", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = engine.New(cfg, engine.WithWorkers(1)).Run(arts, nil)
		}
	})
	b.Run("workers=default", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = engine.New(cfg).Run(arts, nil)
		}
	})
}
