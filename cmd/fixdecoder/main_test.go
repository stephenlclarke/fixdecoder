package main

import (
	"encoding/xml"
	"os"
	"strings"
	"testing"

	"bitbucket.org/edgewater/fixdecoder/decoder"
)

const (
	expectedSetValueMsg = "Expected String to return the set value"
	defaultFixFlag      = "-fix=44"
)

func TestTagFlagSet(t *testing.T) {
	var f tagFlag
	err := f.Set("35")
	if err != nil || f.value != "35" || !f.isSet {
		t.Error("Expected tagFlag to set correctly")
	}
	if !f.IsBoolFlag() {
		t.Error("Expected tagFlag to report IsBoolFlag true")
	}
	if f.String() != "35" {
		t.Error(expectedSetValueMsg)
	}
}

func TestComponentFlagSet(t *testing.T) {
	var f componentFlag
	err := f.Set("Header")
	if err != nil || f.value != "Header" || !f.isSet {
		t.Error("Expected componentFlag to set correctly")
	}
	if !f.IsBoolFlag() {
		t.Error("Expected componentFlag to report IsBoolFlag true")
	}
	if f.String() != "Header" {
		t.Error(expectedSetValueMsg)
	}
}

func TestMessageFlagSet(t *testing.T) {
	var f messageFlag
	err := f.Set("Logon")
	if err != nil || f.value != "Logon" || !f.isSet {
		t.Error("Expected messageFlag to set correctly")
	}
	if !f.IsBoolFlag() {
		t.Error("Expected messageFlag to report IsBoolFlag true")
	}
	if f.String() != "Logon" {
		t.Error(expectedSetValueMsg)
	}
}

func TestParseFlagsArgsDefaults(t *testing.T) {
	args := []string{defaultFixFlag, "-verbose", "-header", "-trailer", "-column", "-info"}
	opts := parseFlagsArgs(args)

	if opts.FixVersion != "44" || !opts.Verbose || !opts.IncludeHeader || !opts.IncludeTrailer || !opts.ColumnOutput || !opts.Info {
		t.Error("Expected flags to parse correctly with defaults")
	}
}

func TestParseFlagsArgsWithMessageComponentTag(t *testing.T) {
	args := []string{"-message=Logon", "-component=Header", "-tag=35"}
	opts := parseFlagsArgs(args)

	if opts.Message.value != "Logon" || opts.Component.value != "Header" || opts.Tag.value != "35" {
		t.Error("Expected flags to capture correct values")
	}
	if !opts.Message.isSet || !opts.Component.isSet || !opts.Tag.isSet {
		t.Error("Expected flags to mark isSet true")
	}
}

func TestPrintUsage(t *testing.T) {
	out := captureOutput(func() {
		PrintUsage()
	})

	expectedStrings := []string{
		"fixdecoder",        // version info
		"git clone",         // Git URL
		"Usage: fixdecoder", // usage line
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(out, expected) {
			t.Errorf("Expected output to include %q, but it did not.\nFull output:\n%s", expected, out)
		}
	}
}

func TestLoadSchemaSuccess(t *testing.T) {
	sample := `<?xml version="1.0"?><fix major="4" minor="4"><header/><messages/><trailer/></fix>`
	tmp := "test_fix.xml"
	os.WriteFile(tmp, []byte(sample), 0644)
	defer os.Remove(tmp)

	schema, err := loadSchema(tmp)
	if err != nil {
		t.Errorf("Expected successful schema load, got error: %v", err)
	}
	expectedVersion := "4.4" // Adjusted to match actual implementation
	if schema.Version != expectedVersion {
		t.Errorf("Expected schema version %s, got: %s", expectedVersion, schema.Version)
	}
}

func TestLoadSchemaReadError(t *testing.T) {
	_, err := loadSchema("nonexistent.xml")
	if err == nil {
		t.Error("Expected error for missing file")
	}
}

func TestLoadSchemaUnmarshalError(t *testing.T) {
	tmp := "bad_fix.xml"
	os.WriteFile(tmp, []byte("<not valid xml"), 0644)
	defer os.Remove(tmp)

	_, err := loadSchema(tmp)
	if err == nil {
		t.Error("Expected unmarshal error for bad XML")
	}
}

func TestExtractFileArgsOrStdinWithFiles(t *testing.T) {
	files := extractFileArgsOrStdin([]string{"input1.txt", "-v", "input2.txt"})
	if len(files) != 2 || files[0] != "input1.txt" || files[1] != "input2.txt" {
		t.Error("Expected file arguments extracted correctly")
	}
}

func TestExtractFileArgsOrStdinDefaultToStdin(t *testing.T) {
	files := extractFileArgsOrStdin([]string{"-v", "--flag"})
	if len(files) != 1 || files[0] != "-" {
		t.Error("Expected fallback to '-' for stdin")
	}
}

func TestRunHandlersWithValidSchema(t *testing.T) {
	xmlData := `<fix major="4" minor="4"></fix>`
	var dict decoder.FixDictionary
	err := xml.Unmarshal([]byte(xmlData), &dict)
	if err != nil {
		t.Fatalf("Failed to unmarshal XML: %v", err)
	}
	schema := decoder.BuildSchema(dict)

	opts := CLIOptions{
		Message:    messageFlag{value: "A", isSet: true},
		FixVersion: "4.4",
	}

	ok := runHandlers(opts, schema)
	if !ok {
		t.Error("Expected runHandlers to succeed with valid schema and message")
	}
}

func TestLoadSchemaFromOptsEmbeddedSuccess(t *testing.T) {
	opts := CLIOptions{FixVersion: "4.4"}
	schema, err := loadSchemaFromOpts(opts)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(schema.Messages) == 0 {
		t.Errorf("Expected non-empty schema messages, got: %+v", schema)
	}
}
func TestProcessWithValidArgs(t *testing.T) {
	tmp, _ := os.CreateTemp("", "log*.txt")
	defer os.Remove(tmp.Name())
	_ = os.WriteFile(tmp.Name(), []byte("foo=bar"), 0644)

	args := []string{defaultFixFlag, tmp.Name()}
	var out, errOut strings.Builder

	code := Process(args, &out, &errOut)
	if code != 0 {
		t.Errorf("Expected success, got code=%d, err=%s", code, errOut.String())
	}
}

func TestLoadSchemaFromOptsExternalXML(t *testing.T) {
	xml := []byte(`<fix major="4" minor="4"></fix>`)
	tmp, _ := os.CreateTemp("", "fix*.xml")
	defer os.Remove(tmp.Name())
	_ = os.WriteFile(tmp.Name(), xml, 0644)

	opts := CLIOptions{XMLPath: tmp.Name()}
	schema, err := loadSchemaFromOpts(opts)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(schema.Messages) != 0 {
		t.Errorf("Expected empty schema messages for minimal XML, got: %d", len(schema.Messages))
	}
}

func TestProcessRunHandlersPath(t *testing.T) {
	var out, errOut strings.Builder
	code := Process([]string{defaultFixFlag, "-message=A"}, &out, &errOut)
	if code != 0 {
		t.Errorf("Expected 0 code from runHandlers path, got %d", code)
	}
}

func TestProcessPrettifyFilesPath(t *testing.T) {
	// Create a dummy log file
	tmp, _ := os.CreateTemp("", "test*.log")
	defer os.Remove(tmp.Name())
	_ = os.WriteFile(tmp.Name(), []byte("some data"), 0644)

	var out, errOut strings.Builder
	code := Process([]string{defaultFixFlag, tmp.Name()}, &out, &errOut)
	if code != 0 {
		t.Errorf("Expected 0 code from PrettifyFiles path, got %d", code)
	}
}

func TestLoadSchemaFromOptsExternalUnmarshalError(t *testing.T) {
	// Write invalid XML to a temp file
	tmp, _ := os.CreateTemp("", "bad*.xml")
	defer os.Remove(tmp.Name())
	_ = os.WriteFile(tmp.Name(), []byte("<bad"), 0644)

	opts := CLIOptions{XMLPath: tmp.Name()}
	_, err := loadSchemaFromOpts(opts)

	if err == nil || !strings.Contains(err.Error(), "XML syntax error") {
		t.Errorf("Expected unmarshalling error, got: %v", err)
	}
}

func TestLoadSchemaFromOptsXMLUnmarshalError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "bad*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	defer os.Remove(tmpFile.Name())

	// Write malformed XML content
	if _, err := tmpFile.WriteString("<fix>"); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	_ = tmpFile.Close()

	opts := CLIOptions{XMLPath: tmpFile.Name()}
	_, err = loadSchemaFromOpts(opts)
	if err == nil {
		t.Fatal("Expected error due to malformed XML, got nil")
	}

	if !strings.Contains(err.Error(), "XML syntax error") {
		t.Errorf("Expected XML syntax error, got: %v", err)
	}
}
