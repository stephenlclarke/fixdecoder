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

// listAllMessages prints all messages in sorted order by MsgType.
func ListAllMessages(schema SchemaTree) {
	var msgs []MessageNode
	for _, m := range schema.Messages {
		msgs = append(msgs, m)
	}

	sort.Slice(msgs, func(i, j int) bool { return msgs[i].MsgType < msgs[j].MsgType })
	for _, m := range msgs {
		fmt.Printf("%-4s: %s (%s)\n", m.MsgType, m.Name, m.MsgCat)
	}
}

// printMessageStart prints the “Message: Name (Type)” header.
func printMessageStart(msg MessageNode) {
	fmt.Printf("Message: %s (%s)\n", msg.Name, msg.MsgType)
}

// displayMessageStructureWithOptions orchestrates the above helpers.
func DisplayMessageStructureWithOptions(
	schema SchemaTree,
	msg MessageNode,
	verbose, includeHeader, includeTrailer, column bool,
	indent int,
) {
	printMessageStart(msg)
	printHeader(schema, msg, includeHeader, verbose, column, indent)
	printFields(msg, verbose, column, indent)
	printComponents(schema, msg, verbose, column, indent)
	printGroups(schema, msg, verbose, column, indent)
	printTrailer(schema, msg, includeTrailer, verbose, column, indent)
}
