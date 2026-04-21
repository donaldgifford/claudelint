package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
)

// profileSession owns every profiling file handle claudelint opens
// for a run. Callers take one session at the start of a run, defer
// close(), and never have to juggle individual pprof.StopCPUProfile /
// WriteHeapProfile calls themselves.
//
// cpu.pprof is captured for the whole run (started on session open,
// stopped on close). heap.pprof, block.pprof, and mutex.pprof are
// snapshots taken at close time so they reflect end-of-run state,
// which for a short-lived linter is more useful than mid-run sampling.
type profileSession struct {
	dir    string
	cpuFP  *os.File
	closed bool
}

// startProfileSession opens every pprof output file under dir and
// begins CPU profiling. dir is created if it does not exist. Returns
// a session whose Close() must run to produce valid pprof files —
// the caller should defer close right after successful construction.
func startProfileSession(dir string) (*profileSession, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create profile dir: %w", err)
	}
	cpuPath := filepath.Join(dir, "cpu.pprof")
	cpuFP, err := os.Create(cpuPath) // #nosec G304 — user-provided output path is the point.
	if err != nil {
		return nil, fmt.Errorf("create cpu.pprof: %w", err)
	}
	if err := pprof.StartCPUProfile(cpuFP); err != nil {
		if cerr := cpuFP.Close(); cerr != nil {
			err = fmt.Errorf("%w (also: close cpu.pprof: %w)", err, cerr)
		}
		return nil, fmt.Errorf("start cpu profile: %w", err)
	}
	// Turn on block and mutex profiling so the session captures them
	// at close. Rates mirror the defaults in
	// https://pkg.go.dev/runtime#SetBlockProfileRate.
	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)
	return &profileSession{dir: dir, cpuFP: cpuFP}, nil
}

// Close stops CPU profiling and writes heap, block, and mutex
// snapshots. The returned error, if any, is the first failure
// encountered — later steps still run so partial output is still
// usable.
func (p *profileSession) Close() error {
	if p.closed {
		return nil
	}
	p.closed = true

	pprof.StopCPUProfile()
	var firstErr error
	if err := p.cpuFP.Close(); err != nil {
		firstErr = fmt.Errorf("close cpu.pprof: %w", err)
	}
	for _, name := range []struct{ file, prof string }{
		{"heap.pprof", "heap"},
		{"block.pprof", "block"},
		{"mutex.pprof", "mutex"},
	} {
		if err := writeNamedProfile(p.dir, name.file, name.prof); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// writeNamedProfile snapshots a pprof-registered profile into dir/name.
// Returns an error if the profile is unknown or the file cannot be
// written. The caller decides whether the failure is fatal.
func writeNamedProfile(dir, filename, profileName string) error {
	prof := pprof.Lookup(profileName)
	if prof == nil {
		return fmt.Errorf("pprof profile %q not registered", profileName)
	}
	path := filepath.Join(dir, filename)
	fp, err := os.Create(path) // #nosec G304 — user-provided output path is the point.
	if err != nil {
		return fmt.Errorf("create %s: %w", filename, err)
	}
	if werr := prof.WriteTo(fp, 0); werr != nil {
		if cerr := fp.Close(); cerr != nil {
			return fmt.Errorf("write %s: %w (close: %w)", filename, werr, cerr)
		}
		return fmt.Errorf("write %s: %w", filename, werr)
	}
	if cerr := fp.Close(); cerr != nil {
		return fmt.Errorf("close %s: %w", filename, cerr)
	}
	return nil
}
