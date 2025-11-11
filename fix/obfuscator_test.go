package fix

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// helper to quickly build a FIX line with SOH separators
func fixLine(pairs ...string) string {
	return strings.Join(pairs, soh) + soh
}

// capture writes to an io.Writer and returns the captured string
type capture struct{ bytes.Buffer }

func (c *capture) Write(p []byte) (int, error) { return c.Buffer.Write(p) }

func TestSplitOnce(t *testing.T) {
	type tc struct {
		in    string
		ok    bool
		left  string
		right string
	}
	cases := []tc{
		{"a=b=c", true, "a", "b=c"},
		{"=value", true, "", "value"},
		{"key=", true, "key", ""},
		{"novalue", false, "", ""},
		{"a\x01b", true, "a", "b"},
	}
	for _, c := range cases {
		l, r, ok := splitOnce(c.in)
		if ok != c.ok || (ok && (l != c.left || r != c.right)) {
			t.Fatalf("splitOnce(%q)=(%q,%q,%v), want (%q,%q,%v)", c.in, l, r, ok, c.left, c.right, c.ok)
		}
	}
}

func TestObfuscatorDisabledReturnsUnchanged(t *testing.T) {
	o := CreateObfuscator(nil, false)
	in := fixLine("8=FIX.4.4", "49=ABC", "56=DEF", "1=ACC")
	out := o.ObfuscateLine(in, nil)
	if out != in {
		t.Fatalf("disabled obfuscator changed input:\n got: %q\nwant: %q", out, in)
	}
}

func TestObfuscatorNoSensitiveTagsReturnsUnchanged(t *testing.T) {
	o := CreateObfuscator(map[int]string{}, true) // enabled, but no sensitive tags
	in := fixLine("8=FIX.4.4", "11=OID1", "38=100", "40=2")
	out := o.ObfuscateLine(in, nil)
	if out != in {
		t.Fatalf("no-sensitive obfuscator changed input:\n got: %q\nwant: %q", out, in)
	}
}

func TestObfuscatorObfuscatesSensitiveValuesWithStableAliases(t *testing.T) {
	sensitive := map[int]string{
		49: "SenderCompID",
		56: "TargetCompID",
		1:  "Account",
	}
	o := CreateObfuscator(sensitive, true)

	// First line: create aliases
	in1 := fixLine("8=FIX.4.4", "49=ABC", "56=DEF", "1=ACC123", "11=OID1")
	var stderr1 capture
	out1 := o.ObfuscateLine(in1, &stderr1)

	// Expect values to be replaced with Name0001 per tag, others unchanged
	if !strings.Contains(out1, "49=SenderCompID0001"+soh) ||
		!strings.Contains(out1, "56=TargetCompID0001"+soh) ||
		!strings.Contains(out1, "1=Account0001"+soh) ||
		!strings.Contains(out1, "11=OID1"+soh) {
		t.Fatalf("unexpected obfuscation result:\n%s", repr(out1))
	}

	// Second line: same values should reuse the same aliases; new values bump counters
	in2 := fixLine("49=ABC", "56=NEWDEF", "1=ACC999", "11=OID2")
	var stderr2 capture
	out2 := o.ObfuscateLine(in2, &stderr2)

	if !strings.Contains(out2, "49=SenderCompID0001"+soh) { // reused
		t.Fatalf("expected reuse of alias for 49=ABC; got:\n%s", repr(out2))
	}
	if !strings.Contains(out2, "56=TargetCompID0002"+soh) { // new value => next counter
		t.Fatalf("expected incremented alias for 56=NEWDEF; got:\n%s", repr(out2))
	}
	if !strings.Contains(out2, "1=Account0002"+soh) { // new account value
		t.Fatalf("expected incremented alias for 1=ACC999; got:\n%s", repr(out2))
	}
	if !strings.Contains(out2, "11=OID2"+soh) { // unaffected field
		t.Fatalf("expected non-sensitive field unchanged; got:\n%s", repr(out2))
	}

	// Ensure stderr writers were accepted; we don't assert exact text
	if stderr1.Len() == 0 || stderr2.Len() == 0 {
		t.Fatalf("expected activity logged to stderr writers")
	}
}

func TestObfuscatorIgnoresMalformedAndNonNumericTags(t *testing.T) {
	sensitive := map[int]string{49: "SenderCompID"}
	o := CreateObfuscator(sensitive, true)

	// Malformed pairs and non-numeric tags should be left as-is
	in := strings.Join([]string{
		"8=FIX.4.4",
		"=NOVALUE", // no key
		"NOEQUALS", // no '='
		"ABC=XYZ",  // non-numeric tag
		"49=",      // empty value (still sensitive; alias should be generated)
		"49=REAL",  // normal sensitive
	}, soh) + soh

	out := o.ObfuscateLine(in, io.Discard)

	if !strings.Contains(out, soh+"=NOVALUE"+soh) || !strings.Contains(out, soh+"NOEQUALS"+soh) || !strings.Contains(out, soh+"ABC=XYZ"+soh) {
		t.Fatalf("expected malformed/non-numeric pairs left intact; got:\n%s", repr(out))
	}

	// For 49= (empty), we expect an alias generated (SenderCompID0001)
	if !strings.Contains(out, soh+"49=SenderCompID0001"+soh) {
		t.Fatalf("expected alias for empty sensitive value; got:\n%s", repr(out))
	}
	// For 49=REAL we expect second alias (SenderCompID0002)
	if !strings.Contains(out, soh+"49=SenderCompID0002"+soh) {
		t.Fatalf("expected incremented alias for second 49 value; got:\n%s", repr(out))
	}
}

// repr provides a human-friendly escaped string for diagnostics
func repr(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\x01' {
			b.WriteString("|SOH|")
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func TestEnabledReturnsUnchangedWhenDisabled(t *testing.T) {
	// Ensure the Enabled wrapper returns the original line when the obfuscator is disabled
	o := CreateObfuscator(nil, false)
	in := fixLine("8=FIX.4.4", "49=ABC", "56=DEF")
	var stderr capture
	out := o.Enabled(in, &stderr)
	if out != in {
		t.Fatalf("Enabled() altered line when disabled:\n got: %q\nwant: %q", out, in)
	}
}
