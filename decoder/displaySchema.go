// displaySchema.go
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

import "fmt"

// PrintSchemaSummary writes a one-line overview of the dictionary that was
// just loaded.
func PrintSchemaSummary(schema SchemaTree) {
	fields := len(schema.Fields)
	components := len(schema.Components)
	messages := len(schema.Messages)
	version := schema.Version
	servicePack := schema.ServicePack

	fmt.Printf("Fields: %d   Components: %d   Messages: %d   Version: %s  Service Pack: %s\n",
		fields, components, messages, version, servicePack)
}
