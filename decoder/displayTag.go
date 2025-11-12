// display.go
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
	"fmt"
	"sort"
)

var printStringColumns = PrintStringColumns

// listAllTags prints every tag number, name, and type.
func ListAllTags(schema SchemaTree) {
	fields := make([]Field, 0, len(schema.Fields))
	for _, f := range schema.Fields {
		fields = append(fields, f)
	}

	sort.Slice(fields, func(i, j int) bool { return fields[i].Number < fields[j].Number })
	for _, field := range fields {
		fmt.Printf("%-4d: %s (%s)\n", field.Number, field.Name, field.Type)
	}
}

// printTagDetails prints a field's header and, if verbose, its enum values.
func PrintTagDetails(field Field, verbose, column bool) {
	fmt.Printf("%-4d: %s (%s)\n", field.Number, field.Name, field.Type)

	if verbose {
		if column {
			printEnumColumns(field.Values, 4)
		} else {
			for _, v := range field.Values {
				printEnum(v.Enum, v.Description, 4)
			}
		}
	}
}

func PrintTagsInColumns(schema SchemaTree) {
	fs := make([]Field, 0, len(schema.Fields))
	for _, f := range schema.Fields {
		fs = append(fs, f)
	}

	sort.Slice(fs, func(i, j int) bool {
		return fs[i].Number < fs[j].Number
	})

	lines := make([]string, len(fs))
	for i, f := range fs {
		lines[i] = fmt.Sprintf("%-4d: %s (%s)", f.Number, f.Name, f.Type)
	}

	printStringColumns(lines)
}
