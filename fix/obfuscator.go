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
	"fmt"
	"io"
	"maps"
	"strconv"
	"strings"
	"sync"
)

const soh = "\x01"

// Obfuscator replaces values of sensitive FIX tags with stable aliases.
// It is safe for concurrent use.
type Obfuscator struct {
	enabled  bool              // global enable/disable flag
	tags     map[int]string    // tag -> name (provided by SensitiveTags)
	mu       sync.Mutex        // protects aliasMap and counter
	aliasMap map[string]string // "tag=value" -> alias
	counter  map[int]int       // per-tag, for zero-padded suffixes
}

// CreateObfuscator constructs an Obfuscator using the given tag map.
// If enabled is false, all calls to Enabled() will return the line unchanged.
func CreateObfuscator(tags map[int]string, enabled bool) *Obfuscator {
	cp := make(map[int]string, len(tags))
	maps.Copy(cp, tags)

	return &Obfuscator{
		enabled:  enabled,
		tags:     cp,
		aliasMap: make(map[string]string),
		counter:  make(map[int]int),
	}
}

// Enabled returns the original line if obfuscation is disabled,
// otherwise returns the obfuscated version and logs first-use events to stderr (if non-nil).
func (o *Obfuscator) Enabled(line string, stderr io.Writer) string {
	if !o.enabled {
		return line
	}
	return o.ObfuscateLine(line, stderr)
}

// ObfuscateLine rewrites a single SOH-delimited FIX line, replacing values for sensitive tags.
// On first occurrence of any tag=value pair, it logs to stderr (if provided).
func (o *Obfuscator) ObfuscateLine(line string, stderr io.Writer) string {
	fields := strings.Split(line, soh)

	for i, f := range fields {
		tagStr, val, ok := splitOnce(f)
		if !ok {
			continue
		}

		tagNum, err := strconv.Atoi(tagStr)
		if err != nil {
			continue
		}

		name, sensitive := o.tags[tagNum]
		if !sensitive {
			continue
		}

		key := tagStr + "=" + val

		o.mu.Lock()
		alias, exists := o.aliasMap[key]
		if !exists {
			o.counter[tagNum]++
			alias = fmt.Sprintf("%s%04d", name, o.counter[tagNum])
			o.aliasMap[key] = alias

			if stderr != nil {
				fmt.Fprintf(stderr, "first use: tag %d (%s) value [%s] → [%s]\n",
					tagNum, name, val, alias)
			}
		}
		o.mu.Unlock()

		fields[i] = tagStr + "=" + alias
	}

	return strings.Join(fields, soh)
}

// ---- small helpers (keep complexity low) ----

func splitOnce(s string) (left, right string, ok bool) {
	// Accept empty left or right and split on first occurrence of '=' or SOH.
	// This allows handling fragments that may still include SOH.
	idx := strings.IndexAny(s, "=\x01")
	if idx < 0 {
		return "", "", false
	}
	return s[:idx], s[idx+1:], true
}
