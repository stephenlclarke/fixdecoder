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
package fix

import (
	"strings"
	"testing"
)

// Mock FIX XML values — in real test environments these would be imported from respective fix packages
const (
	mockFIX40    = "\n<fix type='FIX' major='4' minor='0' servicepack='0'>"
	mockFIX41    = "\n<fix type='FIX' major='4' minor='1' servicepack='0'>"
	mockFIX42    = "\n<fix type='FIX' major='4' minor='2' servicepack='0'>"
	mockFIX43    = "\n<fix type='FIX' major='4' minor='3' servicepack='0'>"
	mockFIX44    = "\n<fix type='FIX' major='4' minor='4' servicepack='0'>"
	mockFIX50    = "\n<fix type='FIX' major='5' minor='0' servicepack='0'>"
	mockFIX50SP1 = "\n<fix type='FIX' major='5' minor='0' servicepack='1'>"
	mockFIX50SP2 = "\n<fix type='FIX' major='5' minor='0' servicepack='2'>"
	mockFIXT11   = "\n<fix type='FIXT' major='1' minor='1' servicepack='0'>"
)

func TestChooseEmbeddedXML(t *testing.T) {
	tests := []struct {
		version       string
		expectedStart string
	}{
		{"40", mockFIX40},
		{"41", mockFIX41},
		{"42", mockFIX42},
		{"43", mockFIX43},
		{"44", mockFIX44},
		{"50", mockFIX50},
		{"50SP1", mockFIX50SP1},
		{"50SP2", mockFIX50SP2},
		{"T11", mockFIXT11},
		{"unknown", mockFIX44}, // default fallback
	}

	for _, tt := range tests {
		result := ChooseEmbeddedXML(tt.version)

		// Only check that the returned XML starts with the expected string
		// (we avoid comparing huge XML blobs directly)
		if !strings.HasPrefix(result, tt.expectedStart) {
			t.Errorf("ChooseEmbeddedXML(%q) = %q, want prefix %q", tt.version, result[:50], tt.expectedStart)
		}
	}
}

func TestSupportedFixVersions(t *testing.T) {
	got := SupportedFixVersions()
	expected := "40,41,42,43,44,50,50SP1,50SP2,T11"

	if got != expected {
		t.Errorf("SupportedFixVersions() = %q, want %q", got, expected)
	}
}
