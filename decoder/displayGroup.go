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
)

// displayGroup displays a GroupNode with its fields, components, and nested groups.
func DisplayGroup(schema SchemaTree, g GroupNode, verbose bool, columnOutput bool, indent int) {
	printIndent(indent)

	fmt.Printf("Group: %s%s\n", g.Name, formatRequired(g.Required))

	for _, f := range g.Fields {
		printField(f, indent+4)

		if verbose && columnOutput {
			printEnumColumns(f.Field.Values, indent+6)
		} else if verbose {
			for _, val := range f.Field.Values {
				printEnum(val.Enum, val.Description, indent+6)
			}
		}
	}

	for _, c := range g.Components {
		DisplayComponent(schema, MessageNode{}, c, verbose, columnOutput, indent+4)
	}

	for _, sg := range g.Groups {
		DisplayGroup(schema, sg, verbose, columnOutput, indent+4)
	}
}

// printGroups prints all repeating groups of the message.
func printGroups(schema SchemaTree, msg MessageNode, verbose, column bool, indent int) {
	for _, g := range msg.Groups {
		DisplayGroup(schema, g, verbose, column, indent)
	}
}
