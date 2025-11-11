package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// --- Test helpers (match simple style used elsewhere) ---

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func chdir(t *testing.T, dir string) func() {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	return func() { _ = os.Chdir(wd) }
}

// normalize newlines for cross-platform asserts
func normNL(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

// evalSymlink normalizes paths (fixes /private prefix on macOS temp dirs)
func evalSymlink(t *testing.T, p string) string {
	t.Helper()
	q, err := filepath.EvalSymlinks(p)
	if err != nil {
		// Fallback to Clean if evaluation fails (shouldn’t)
		return filepath.Clean(p)
	}
	return q
}

// captureOutput redirects stdout while f runs and returns the captured output.
// This keeps 'go test -json' output clean for report tools, if ever re-enabled.
func captureOutput(t *testing.T, f func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		_ = w.Close()
		os.Stdout = orig
	}()

	f()

	_ = w.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	return string(b)
}

// -----------------------------------------------------------------------------
// exists / isDir
// -----------------------------------------------------------------------------

func TestExistsAndIsDir(t *testing.T) {
	tmp := t.TempDir()

	// file
	f := filepath.Join(tmp, "a.txt")
	mustWriteFile(t, f, "x")
	if !exists(f) {
		t.Error("exists(file) = false, want true")
	}
	if isDir(f) {
		t.Error("isDir(file) = true, want false")
	}

	// dir
	if !isDir(tmp) {
		t.Error("isDir(dir) = false, want true")
	}

	// non-existent
	ne := filepath.Join(tmp, "nope")
	if exists(ne) {
		t.Error("exists(nonexistent) = true, want false")
	}
	if isDir(ne) {
		t.Error("isDir(nonexistent) = true, want false")
	}
}

// -----------------------------------------------------------------------------
// findRepoRoot
// -----------------------------------------------------------------------------

func TestFindRepoRootResourcesOnly(t *testing.T) {
	tmp := t.TempDir()
	res := filepath.Join(tmp, "resources")
	if err := os.MkdirAll(res, 0o755); err != nil {
		t.Fatalf("mkdir resources: %v", err)
	}
	runFrom := filepath.Join(tmp, "sub", "child")
	if err := os.MkdirAll(runFrom, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	defer chdir(t, runFrom)()

	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot error: %v", err)
	}
	// Normalize symlinks to avoid /private prefix mismatches on macOS
	got := evalSymlink(t, root)
	want := evalSymlink(t, tmp)
	if got != want {
		t.Errorf("findRepoRoot = %q, want %q", got, want)
	}
}

func TestFindRepoRootWithGoMod(t *testing.T) {
	tmp := t.TempDir()
	mustWriteFile(t, filepath.Join(tmp, "go.mod"), "module example.com/x\n")
	defer chdir(t, tmp)()

	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot error: %v", err)
	}
	got := evalSymlink(t, root)
	want := evalSymlink(t, tmp)
	if got != want {
		t.Errorf("findRepoRoot = %q, want %q", got, want)
	}
}

func TestFindRepoRootNotFound(t *testing.T) {
	tmp := t.TempDir()
	defer chdir(t, tmp)()

	if _, err := findRepoRoot(); err == nil {
		t.Error("expected error when no go.mod/resources present")
	}
}

// -----------------------------------------------------------------------------
// parseFixXML / loadAllFields
// -----------------------------------------------------------------------------

func TestParseFixXMLSuccess(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "fix44.xml")
	xml := `
<fix>
  <fields>
    <field number="49" name="SenderCompID" type="STRING"/>
    <field number="56" name="TargetCompID" type="STRING"/>
    <field number="1"  name="Account" type="STRING"/>
    <field number="0"  name="IgnoredZero" type="STRING"/>
    <field number="2"  name="" type="STRING"/>
  </fields>
</fix>`
	mustWriteFile(t, p, xml)

	m, err := parseFixXML(p)
	if err != nil {
		t.Fatalf("parseFixXML error: %v", err)
	}
	if len(m) != 3 {
		t.Errorf("len(m)=%d, want 3", len(m))
	}
	if m[49] != "SenderCompID" || m[56] != "TargetCompID" || m[1] != "Account" {
		t.Errorf("unexpected values: %#v", m)
	}
}

func TestParseFixXMLFileNotFound(t *testing.T) {
	if _, err := parseFixXML(filepath.Join(t.TempDir(), "nope.xml")); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestParseFixXMLInvalidXML(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "bad.xml")
	mustWriteFile(t, p, "<fix><fields><field></fix>")

	if _, err := parseFixXML(p); err == nil {
		t.Error("expected XML decode error")
	}
}

func TestLoadAllFieldsFirstWinsForDuplicates(t *testing.T) {
	tmp := t.TempDir()
	a := filepath.Join(tmp, "a.xml")
	b := filepath.Join(tmp, "b.xml")

	mustWriteFile(t, a, `<fix><fields><field number="100" name="Foo" type="STRING"/></fields></fix>`)
	mustWriteFile(t, b, `<fix><fields><field number="100" name="Bar" type="STRING"/><field number="200" name="Baz" type="STRING"/></fields></fix>`)

	got, err := loadAllFields([]string{a, b})
	if err != nil {
		t.Fatalf("loadAllFields error: %v", err)
	}
	if got[100] != "Foo" {
		t.Errorf("tag 100 = %q, want %q", got[100], "Foo")
	}
	if got[200] != "Baz" {
		t.Errorf("tag 200 = %q, want %q", got[200], "Baz")
	}
}

// -----------------------------------------------------------------------------
// filterSensitive
// -----------------------------------------------------------------------------

func TestFilterSensitiveMatchesCaseInsensitiveSubstrings(t *testing.T) {
	all := map[int]string{
		1: "Account",
		2: "UserName",
		3: "passwordHash",
		4: "SenderCompID",
		5: "TargetSubID",
		6: "locationIdOverride",
		7: "NotSensitive",
	}
	got := filterSensitive(all)

	want := []int{1, 2, 3, 4, 5, 6}
	if len(got) != len(want) {
		t.Errorf("len=%d, want %d; got=%v", len(got), len(want), got)
	}
	for _, k := range want {
		if _, ok := got[k]; !ok {
			t.Errorf("expected tag %d present", k)
		}
	}
	if _, ok := got[7]; ok {
		t.Error("did not expect tag 7 present")
	}
}

// -----------------------------------------------------------------------------
// writeGeneratedFile / writeHeader / writeMap
// -----------------------------------------------------------------------------

func TestWriteGeneratedFileFormatsAndSorts(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "fix", "sensitiveTagNames.go")

	in := map[int]string{
		56: "TargetCompID",
		1:  "Account",
		49: "SenderCompID",
	}
	if err := writeGeneratedFile(out, in); err != nil {
		t.Fatalf("writeGeneratedFile error: %v", err)
	}

	got := normNL(mustReadFile(t, out))
	if !strings.Contains(got, "package fix\n") {
		t.Error("missing package declaration")
	}
	if !strings.Contains(got, "// Code generated by generateSensitiveTagNames; DO NOT EDIT.") {
		t.Error("missing generated header")
	}

	// Ensure sorted by tag; allow gofmt spacing after ':'
	re1 := regexp.MustCompile(`\n\t1:\s+"Account",`)
	re49 := regexp.MustCompile(`\n\t49:\s+"SenderCompID",`)
	re56 := regexp.MustCompile(`\n\t56:\s+"TargetCompID",`)

	i1 := re1.FindStringIndex(got)
	i49 := re49.FindStringIndex(got)
	i56 := re56.FindStringIndex(got)
	if i1 == nil || i49 == nil || i56 == nil {
		t.Fatalf("missing map entries:\n%s", got)
	}
	if !(i1[0] < i49[0] && i49[0] < i56[0]) {
		t.Errorf("expected order 1 < 49 < 56; indexes %v, %v, %v", i1, i49, i56)
	}
}

func TestWriteHeaderAndMapDirect(t *testing.T) {
	var buf bytes.Buffer
	writeHeader(&buf)
	writeMap(&buf, map[int]string{
		56: "TargetCompID",
		49: "SenderCompID",
	})

	got := normNL(buf.String())
	if !strings.HasPrefix(got, "package fix\n\n// Code generated by generateSensitiveTagNames; DO NOT EDIT.\n") {
		t.Errorf("unexpected header:\n%s", got)
	}
	// Allow gofmt spacing
	wantBodyRe := regexp.MustCompile("\nvar SensitiveTagNames = map\\[int\\]string\\{\n\t49:\\s+\"SenderCompID\",\n\t56:\\s+\"TargetCompID\",\n\\}\n")
	if !wantBodyRe.MatchString(got) {
		t.Errorf("map body not as expected.\nGot:\n%s", got)
	}
}

// -----------------------------------------------------------------------------
// writeGeneratedFile error branches
// -----------------------------------------------------------------------------

// Test the mkdir-all failure branch: parent path exists as a file, not a directory.
func TestWriteGeneratedFileMkdirAllError(t *testing.T) {
	tmp := t.TempDir()

	// Create a file where the parent directory is expected to be.
	parentAsFile := filepath.Join(tmp, "not-a-dir")
	mustWriteFile(t, parentAsFile, "x") // this is a FILE

	// Target path whose parent is a FILE; MkdirAll should fail.
	target := filepath.Join(parentAsFile, "sensitiveTagNames.go")

	err := writeGeneratedFile(target, map[int]string{
		1:  "Account",
		49: "SenderCompID",
	})
	if err == nil {
		t.Fatalf("expected error from writeGeneratedFile when parent is a file, got nil")
	}
	if !strings.Contains(err.Error(), "mkdir") {
		t.Fatalf("expected mkdir error, got: %v", err)
	}
}

// Test the write-temp failure branch: pre-create a directory at <path>.tmp so
// os.WriteFile(<path>.tmp, ...) fails with "is a directory" (or similar).
func TestWriteGeneratedFileWriteTempError(t *testing.T) {
	tmp := t.TempDir()

	parent := filepath.Join(tmp, "fix")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatalf("mkdir parent: %v", err)
	}

	target := filepath.Join(parent, "sensitiveTagNames.go")
	// Pre-create a DIRECTORY at the temp path to force WriteFile failure.
	preventFile := target + ".tmp"
	if err := os.MkdirAll(preventFile, 0o755); err != nil {
		t.Fatalf("mkdir preventFile: %v", err)
	}

	err := writeGeneratedFile(target, map[int]string{
		1:  "Account",
		49: "SenderCompID",
	})
	if err == nil {
		t.Fatalf("expected error from writeGeneratedFile when temp path is a directory, got nil")
	}
	if !strings.Contains(err.Error(), "write temp") {
		t.Fatalf("expected 'write temp' error, got: %v", err)
	}
}

// -----------------------------------------------------------------------------
// formatSource indirection fallback
// -----------------------------------------------------------------------------

func TestWriteGeneratedFileFormatSourceErrorFallsBack(t *testing.T) {
	// Stub formatSource to force an error path and ensure it was invoked.
	old := formatSource
	defer func() { formatSource = old }()

	called := 0
	formatSource = func(b []byte) ([]byte, error) {
		called++
		return nil, errors.New("boom")
	}

	tmp := t.TempDir()
	target := filepath.Join(tmp, "fix", "sensitiveTagNames.go")

	tags := map[int]string{1: "Account", 49: "SenderCompID", 56: "TargetCompID"}

	if err := writeGeneratedFile(target, tags); err != nil {
		t.Fatalf("writeGeneratedFile error: %v", err)
	}
	if called != 1 {
		t.Fatalf("expected formatSource to be called once, got %d", called)
	}

	// The fallback should have written the *unformatted* buffer content.
	// Reproduce the buffer the function would have built and compare bytes.
	var buf bytes.Buffer
	writeHeader(&buf)
	writeMap(&buf, tags)

	got := normNL(mustReadFile(t, target))
	want := normNL(buf.String())
	if got != want {
		t.Fatalf("fallback content mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// -----------------------------------------------------------------------------
// relOrSame
// -----------------------------------------------------------------------------

func TestRelOrSame(t *testing.T) {
	tmp := t.TempDir()
	a := filepath.Join(tmp, "x", "y", "z.txt")
	mustWriteFile(t, a, "ok")

	rel := relOrSame(a, tmp)
	if filepath.IsAbs(rel) || strings.HasPrefix(rel, "..") {
		t.Errorf("expected relative path, got %q", rel)
	}

	// outside root => unchanged
	outside := filepath.Join(filepath.Dir(tmp), "q.txt")
	mustWriteFile(t, outside, "q")
	if got := relOrSame(outside, tmp); got != outside {
		t.Errorf("expected unchanged path, got %q", got)
	}
}

// -----------------------------------------------------------------------------
// run() end-to-end
// -----------------------------------------------------------------------------

func TestRunEndToEndSuccess(t *testing.T) {
	// repo:
	//   resources/fix44.xml → 1,49,56 sensitive; 999 not
	repo := t.TempDir()
	mustWriteFile(t, filepath.Join(repo, "go.mod"), "module example.com/fixdecoder\n")

	res := filepath.Join(repo, "resources")
	if err := os.MkdirAll(res, 0o755); err != nil {
		t.Fatalf("mkdir resources: %v", err)
	}
	xml := `
<fix>
  <fields>
    <field number="49" name="SenderCompID" type="STRING"/>
    <field number="56" name="TargetCompID" type="STRING"/>
    <field number="1"  name="Account" type="STRING"/>
    <field number="999" name="NotSensitive" type="STRING"/>
  </fields>
</fix>`
	mustWriteFile(t, filepath.Join(res, "fix44.xml"), xml)

	runFrom := filepath.Join(repo, "deep", "nest")
	if err := os.MkdirAll(runFrom, 0o755); err != nil {
		t.Fatalf("mkdir nest: %v", err)
	}
	defer chdir(t, runFrom)()

	var runErr error
	_ = captureOutput(t, func() {
		runErr = run()
	})
	if runErr != nil {
		t.Fatalf("run error: %v", runErr)
	}

	out := filepath.Join(repo, "fix", "sensitiveTagNames.go")
	data := normNL(mustReadFile(t, out))

	// Allow gofmt spacing after ':'
	re1 := regexp.MustCompile(`\n\t1:\s+"Account",`)
	re49 := regexp.MustCompile(`\n\t49:\s+"SenderCompID",`)
	re56 := regexp.MustCompile(`\n\t56:\s+"TargetCompID",`)

	if !re1.MatchString(data) {
		t.Fatalf("missing 1/Account in generated file:\n%s", data)
	}
	if !re49.MatchString(data) {
		t.Fatalf("missing 49/SenderCompID in generated file:\n%s", data)
	}
	if !re56.MatchString(data) {
		t.Fatalf("missing 56/TargetCompID in generated file:\n%s", data)
	}
	if strings.Contains(data, "NotSensitive") {
		t.Error("unexpected NotSensitive in generated file")
	}
}

func TestRunCannotLocateRepoRoot(t *testing.T) {
	tmp := t.TempDir()
	defer chdir(t, tmp)()

	var err error
	_ = captureOutput(t, func() {
		err = run()
	})
	if err == nil || !strings.Contains(err.Error(), "cannot locate repo root") {
		t.Errorf("want cannot locate repo root error, got: %v", err)
	}
}

func TestRunResourcesDirNotFound(t *testing.T) {
	repo := t.TempDir()
	mustWriteFile(t, filepath.Join(repo, "go.mod"), "module x\n")
	defer chdir(t, repo)()

	var err error
	_ = captureOutput(t, func() {
		err = run()
	})
	if err == nil || !strings.Contains(err.Error(), "resources directory not found") {
		t.Errorf("want resources not found error, got: %v", err)
	}
}

func TestRunNoXMLFiles(t *testing.T) {
	repo := t.TempDir()
	mustWriteFile(t, filepath.Join(repo, "go.mod"), "module x\n")
	res := filepath.Join(repo, "resources")
	if err := os.MkdirAll(res, 0o755); err != nil {
		t.Fatalf("mkdir resources: %v", err)
	}
	defer chdir(t, repo)()

	var err error
	_ = captureOutput(t, func() {
		err = run()
	})
	if err == nil || !strings.Contains(err.Error(), "no FIX XML files") {
		t.Errorf("want no FIX XML files error, got: %v", err)
	}
}

func TestRunNoSensitiveTagsFound(t *testing.T) {
	repo := t.TempDir()
	mustWriteFile(t, filepath.Join(repo, "go.mod"), "module x\n")
	res := filepath.Join(repo, "resources")
	if err := os.MkdirAll(res, 0o755); err != nil {
		t.Fatalf("mkdir resources: %v", err)
	}
	mustWriteFile(t, filepath.Join(res, "fix44.xml"), `<fix><fields><field number="10" name="CheckSum" type="STRING"/></fields></fix>`)
	defer chdir(t, repo)()

	var err error
	_ = captureOutput(t, func() {
		err = run()
	})
	if err == nil || !strings.Contains(err.Error(), "no sensitive tags found") {
		t.Errorf("want 'no sensitive tags found' error, got: %v", err)
	}
}

// -----------------------------------------------------------------------------
// run() additional error-path tests for Sonar coverage
// -----------------------------------------------------------------------------

func TestRunGlobError(t *testing.T) {
	repo := t.TempDir()
	mustWriteFile(t, filepath.Join(repo, "go.mod"), "module x\n")
	if err := os.MkdirAll(filepath.Join(repo, "resources"), 0o755); err != nil {
		t.Fatalf("mkdir resources: %v", err)
	}
	defer chdir(t, repo)()

	// Stub glob to force an error
	old := filepathGlob
	defer func() { filepathGlob = old }()
	filepathGlob = func(pattern string) ([]string, error) { return nil, errors.New("glob fail") }

	var err error
	_ = captureOutput(t, func() { err = run() })
	if err == nil || !strings.Contains(err.Error(), "glob resources") {
		t.Fatalf("want glob resources error, got: %v", err)
	}
}

func TestRunLoadAllFieldsError(t *testing.T) {
	repo := t.TempDir()
	mustWriteFile(t, filepath.Join(repo, "go.mod"), "module x\n")
	res := filepath.Join(repo, "resources")
	if err := os.MkdirAll(res, 0o755); err != nil {
		t.Fatalf("mkdir resources: %v", err)
	}
	// Bad XML to make parseFixXML fail inside loadAllFields
	bad := filepath.Join(res, "bad.xml")
	mustWriteFile(t, bad, "<fix><fields><field></fix>")
	defer chdir(t, repo)()

	var err error
	_ = captureOutput(t, func() { err = run() })
	if err == nil || !strings.Contains(err.Error(), "bad.xml") {
		t.Fatalf("want loadAllFields(parse bad.xml) error, got: %v", err)
	}
}

func TestRunWriteGeneratedFileError(t *testing.T) {
	// Set up valid inputs, but make repoRoot/fix a FILE so MkdirAll fails
	repo := t.TempDir()
	mustWriteFile(t, filepath.Join(repo, "go.mod"), "module x\n")

	res := filepath.Join(repo, "resources")
	if err := os.MkdirAll(res, 0o755); err != nil {
		t.Fatalf("mkdir resources: %v", err)
	}
	// Valid XML with sensitive fields
	xml := `
<fix>
  <fields>
    <field number="1"  name="Account" type="STRING"/>
    <field number="49" name="SenderCompID" type="STRING"/>
  </fields>
</fix>`
	mustWriteFile(t, filepath.Join(res, "fix44.xml"), xml)

	// Create a FILE named "fix" at repo root
	mustWriteFile(t, filepath.Join(repo, "fix"), "not a dir")

	defer chdir(t, repo)()

	var err error
	_ = captureOutput(t, func() { err = run() })
	if err == nil || !strings.Contains(err.Error(), "mkdir") {
		t.Fatalf("want mkdir error from writeGeneratedFile, got: %v", err)
	}
}
