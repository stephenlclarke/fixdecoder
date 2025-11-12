// fixparser.go
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
	"strconv"
	"strings"
)

type FieldValue struct {
	Tag   int
	Value string
}

func ParseFix(msg string) []FieldValue {
	// If there's no SOH delimiter, assume no valid fields
	if !strings.Contains(msg, "\x01") {
		return nil
	}

	parts := strings.Split(msg, "\x01")
	out := make([]FieldValue, 0, len(parts))

	for _, p := range parts {
		if p == "" {
			continue
		}

		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			continue
		}

		tag, err := strconv.Atoi(kv[0])
		if err != nil {
			continue
		}

		out = append(out, FieldValue{Tag: tag, Value: kv[1]})
	}

	return out
}
