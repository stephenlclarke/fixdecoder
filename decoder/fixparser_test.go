// fixParser_test.go
/*
fixdecoder — FIX protocol decoder tools
Copyright (C) 2025 Steve Clarke <stephenlclarke@mac.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.

In accordance with section 13 of the AGPL, if you modify this program,
your modified version must prominently offer all users interacting with it
remotely through a computer network an opportunity to receive the source
code of your version.
*/
package decoder

import (
	"reflect"
	"testing"
)

func TestParseFixValidFields(t *testing.T) {
	msg := "8=FIX.4.4\x019=112\x0135=A\x01"
	got := ParseFix(msg)

	want := []FieldValue{
		{Tag: 8, Value: "FIX.4.4"},
		{Tag: 9, Value: "112"},
		{Tag: 35, Value: "A"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseFix() = %v, want %v", got, want)
	}
}

func TestParseFixNoSOH(t *testing.T) {
	msg := "8=FIX.4.49=11235=A"

	if got := ParseFix(msg); got != nil {
		t.Errorf("Expected nil when no SOH, got %v", got)
	}
}

func TestParseFixEmptyFields(t *testing.T) {
	msg := "\x01\x01\x01" // only delimiters, no data

	got := ParseFix(msg)
	if len(got) != 0 {
		t.Errorf("Expected 0 parsed fields, got %d", len(got))
	}
}

func TestParseFixFieldWithoutEquals(t *testing.T) {
	msg := "8=FIX.4.4\x01BADFIELD\x0135=A\x01"
	got := ParseFix(msg)

	want := []FieldValue{
		{Tag: 8, Value: "FIX.4.4"},
		{Tag: 35, Value: "A"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Expected valid fields only, got %v", got)
	}
}

func TestParseFixInvalidTagNumber(t *testing.T) {
	msg := "abc=value\x018=FIX.4.4\x01"
	got := ParseFix(msg)

	want := []FieldValue{
		{Tag: 8, Value: "FIX.4.4"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Expected valid numeric tags only, got %v", got)
	}
}
