package update

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// buildZipWithEntry builds an in-memory archive containing a single file with
// the given name and content.
func buildZipWithEntry(t *testing.T, name, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create(name)
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := f.Write([]byte(content)); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

// TestUnzipCannotOverwriteRunningExecutable documents why UpdateCore retires
// the executable before unzipping on Windows: the image file of a running
// process is held open without write sharing, so extracting the update
// archive over it fails with "Access is denied" (issue #347).
func TestUnzipCannotOverwriteRunningExecutable(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only behavior")
	}
	execPath, err := os.Executable()
	if err != nil {
		t.Fatalf("executable: %v", err)
	}
	data := buildZipWithEntry(t, filepath.Base(execPath), "replacement")
	if err := unzip(data, filepath.Dir(execPath)); err == nil {
		// If overwriting the running binary suddenly works, the retire step
		// is no longer needed; fail so this can be revisited.
		t.Fatal("unzip over the running executable unexpectedly succeeded")
	}
}

// TestRetireThenUnzipSucceeds verifies the fix for issue #347: the running
// executable is renamed aside first, the archive then extracts cleanly onto
// the original path, and restoring works.
func TestRetireThenUnzipSucceeds(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only behavior")
	}
	execPath, err := os.Executable()
	if err != nil {
		t.Fatalf("executable: %v", err)
	}
	orig, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatalf("read executable: %v", err)
	}
	if len(orig) == 0 {
		t.Fatal("empty executable")
	}

	retired, err := retireExecutable(execPath)
	if err != nil {
		t.Fatalf("retire executable: %v", err)
	}
	if _, err := os.Stat(retired); err != nil {
		t.Fatalf("retired binary missing: %v", err)
	}

	restore := func() {
		os.Remove(execPath)
		if err := os.Rename(retired, execPath); err != nil {
			t.Fatalf("restore executable: %v", err)
		}
	}
	defer restore()

	data := buildZipWithEntry(t, filepath.Base(execPath), "replacement")
	if err := unzip(data, filepath.Dir(execPath)); err != nil {
		t.Fatalf("unzip after retire: %v", err)
	}
	got, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatalf("read new executable: %v", err)
	}
	if string(got) != "replacement" {
		t.Fatalf("unexpected content %q", got)
	}
}

// TestRetireExecutableReplacesStaleCopy ensures a leftover .old file from an
// interrupted update does not block the next self-update.
func TestRetireExecutableReplacesStaleCopy(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "octopus.exe")
	if err := os.WriteFile(execPath, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}
	stale := execPath + retiredExecutableSuffix
	if err := os.WriteFile(stale, []byte("stale"), 0o755); err != nil {
		t.Fatal(err)
	}
	retired, err := retireExecutable(execPath)
	if err != nil {
		t.Fatalf("retire: %v", err)
	}
	if retired != stale {
		t.Fatalf("expected retired path %q, got %q", stale, retired)
	}
	got, err := os.ReadFile(stale)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("stale copy not replaced, content %q", got)
	}
}

// TestCleanupRetiredExecutable covers the startup cleanup path.
func TestCleanupRetiredExecutable(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "octopus.exe")
	if err := os.WriteFile(execPath, []byte("current"), 0o755); err != nil {
		t.Fatal(err)
	}
	// point the cleanup at our temp copy via a saved cwd switch: os.Executable
	// returns the test binary, so exercise the remove-only path instead.
	retired, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	retired += retiredExecutableSuffix
	if err := os.WriteFile(retired, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	CleanupRetiredExecutable()
	if _, err := os.Stat(retired); !os.IsNotExist(err) {
		t.Fatalf("retired executable not cleaned up: %v", err)
	}
	// second run must be a no-op when nothing is left over
	CleanupRetiredExecutable()
	_ = execPath
}
