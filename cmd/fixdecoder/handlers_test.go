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
package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stephenlclarke/fixdecoder/decoder"
)

func captureOutput(f func()) string {
	r, w, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = stdout
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

var fullSchema = decoder.SchemaTree{
	Version:     "FIX.4.4",
	ServicePack: "SP2",
	Messages: map[string]decoder.MessageNode{
		"Logon": {
			Name:    "Logon",
			MsgType: "A",
			MsgCat:  "admin",
		},
	},
	Components: map[string]decoder.ComponentNode{
		"Header": {},
	},
	Fields: map[string]decoder.Field{
		"35": {Name: "MsgType", Number: 35, Type: "STRING"},
	},
}

func TestHandleMessageListAllMessages(t *testing.T) {
	opts := CLIOptions{
		Message:      messageFlag{isSet: true, value: "true"},
		ColumnOutput: false,
	}
	out := captureOutput(func() {
		handleMessage(opts, fullSchema)
	})
	if !strings.Contains(out, "Logon") {
		t.Error("Expected message listing")
	}
}

func TestHandleTagColumnOutput(t *testing.T) {
	opts := CLIOptions{
		Tag:          tagFlag{isSet: true, value: "true"},
		ColumnOutput: true,
	}
	out := captureOutput(func() {
		handleTag(opts, fullSchema)
	})
	if out == "" {
		t.Error("Expected tag columns to be printed")
	}
}

func TestHandleTagInvalidNumber(t *testing.T) {
	opts := CLIOptions{Tag: tagFlag{isSet: true, value: "notanumber"}}
	out := captureOutput(func() {
		handleTag(opts, fullSchema)
	})
	if !strings.Contains(out, "Invalid tag") {
		t.Error("Expected invalid tag message")
	}
}

func TestHandleTagNotFound(t *testing.T) {
	opts := CLIOptions{Tag: tagFlag{isSet: true, value: "99"}}
	out := captureOutput(func() {
		handleTag(opts, fullSchema)
	})
	if !strings.Contains(out, "Tag not found") {
		t.Error("Expected tag not found message")
	}
}

func TestHandleComponentAllPaths(t *testing.T) {
	// bare
	out := captureOutput(func() {
		handleComponent(CLIOptions{
			Component:    componentFlag{isSet: true, value: "true"},
			ColumnOutput: false,
		}, fullSchema)
	})
	if out == "" {
		t.Error("Expected component listing")
	}

	// empty
	out = captureOutput(func() {
		handleComponent(CLIOptions{
			Component: componentFlag{isSet: true, value: ""},
		}, fullSchema)
	})
	if !strings.Contains(out, "Usage") {
		t.Error("Expected usage for empty component flag")
	}

	// named - not found
	out = captureOutput(func() {
		handleComponent(CLIOptions{
			Component: componentFlag{isSet: true, value: "Unknown"},
		}, fullSchema)
	})
	if !strings.Contains(out, "Component not found") {
		t.Error("Expected not found component message")
	}
}

func TestRunHandlersAllTrue(t *testing.T) {
	opts := CLIOptions{
		Info:      true,
		Message:   messageFlag{isSet: true, value: "Logon"},
		Tag:       tagFlag{isSet: true, value: "35"},
		Component: componentFlag{isSet: true, value: "Header"},
	}
	result := runHandlers(opts, fullSchema)
	if !result {
		t.Error("Expected runHandlers to return true")
	}
}

func TestHandleInfoOff(t *testing.T) {
	opts := CLIOptions{Info: false}
	result := handleInfo(opts, fullSchema)
	if result {
		t.Error("Expected handleInfo to return false when Info is false")
	}
}

func TestHandleMessageNotSet(t *testing.T) {
	opts := CLIOptions{Message: messageFlag{isSet: false}}
	result := handleMessage(opts, fullSchema)
	if result {
		t.Error("Expected handleMessage to return false when flag not set")
	}
}

func TestHandleMessageSpecificMismatch(t *testing.T) {
	opts := CLIOptions{Message: messageFlag{isSet: true, value: "Unknown"}}
	out := captureOutput(func() {
		handleMessage(opts, fullSchema)
	})
	if !strings.Contains(out, "Message not found") {
		t.Error("Expected 'Message not found' output")
	}
}

func TestHandleComponentNotSet(t *testing.T) {
	opts := CLIOptions{Component: componentFlag{isSet: false}}
	result := handleComponent(opts, fullSchema)
	if result {
		t.Error("Expected handleComponent to return false when not set")
	}
}

func TestHandleBareComponentOutputOrdering(t *testing.T) {
	schema := fullSchema // Lines 150-157
	schema.Components = map[string]decoder.ComponentNode{
		"ZComp": {}, "AComp": {}, "MComp": {},
	}
	opts := CLIOptions{ColumnOutput: true}
	out := captureOutput(func() {
		handleBareComponent(opts, schema)
	})
	if !(strings.Contains(out, "AComp") && strings.Contains(out, "MComp") && strings.Contains(out, "ZComp")) {
		t.Error("Expected sorted component names in output")
	}
}

func TestHandleMessageEmptyMessage(t *testing.T) {
	opts := CLIOptions{Message: messageFlag{isSet: true, value: ""}}
	out := captureOutput(func() {
		handleMessage(opts, fullSchema)
	})
	if !strings.Contains(out, "Usage") {
		t.Error("Expected usage help for empty message flag")
	}
}

func TestHandleTagEmptyValue(t *testing.T) {
	opts := CLIOptions{Tag: tagFlag{isSet: true, value: ""}}
	out := captureOutput(func() {
		handleTag(opts, fullSchema)
	})
	if !strings.Contains(out, "Usage") {
		t.Error("Expected usage message for empty tag")
	}
}

func TestHandleTagValidMatch(t *testing.T) {
	opts := CLIOptions{Tag: tagFlag{isSet: true, value: "35"}}
	out := captureOutput(func() {
		handleTag(opts, fullSchema)
	})
	if !strings.Contains(out, "MsgType") {
		t.Error("Expected tag output for tag 35")
	}
}

func TestHandleMessageSpecificMatch(t *testing.T) {
	opts := CLIOptions{Message: messageFlag{isSet: true, value: "Logon"}, Verbose: true}
	result := handleMessage(opts, fullSchema)
	if !result {
		t.Error("Expected handleMessage to return true for valid message")
	}
}

func TestHandleTagNotSet(t *testing.T) {
	opts := CLIOptions{Tag: tagFlag{isSet: false}}
	result := handleTag(opts, fullSchema)
	if result {
		t.Error("Expected handleTag to return false when not set")
	}
}

func TestHandleXMLEmptyPath(t *testing.T) {
	opts := CLIOptions{XMLPath: ""}
	result := handleXML(opts, fullSchema)
	if result {
		t.Error("Expected handleXML to return false when XMLPath is empty")
	}
}

func TestHandleMessageByExactNameMatch(t *testing.T) {
	opts := CLIOptions{
		Message: messageFlag{isSet: true, value: "Logon"},
		Verbose: true,
	}
	out := captureOutput(func() {
		result := handleMessage(opts, fullSchema)
		if !result {
			t.Error("Expected handleMessage to return true for exact message match")
		}
	})
	if !strings.Contains(out, "Logon") {
		t.Error("Expected output to contain matched message name")
	}
}

func TestHandleTagNotSetFlag(t *testing.T) {
	opts := CLIOptions{Tag: tagFlag{isSet: false}}
	result := handleTag(opts, fullSchema)
	if result {
		t.Error("Expected handleTag to return false when flag is not set")
	}
}

func TestHandleXMLWithPath(t *testing.T) {
	opts := CLIOptions{XMLPath: "/tmp/fix.xml"}
	out := captureOutput(func() {
		result := handleXML(opts, fullSchema)
		if !result {
			t.Error("Expected handleXML to return true when XMLPath is set")
		}
	})
	if !strings.Contains(out, "Dictionary loaded from") {
		t.Error("Expected dictionary load message in output")
	}
}

func TestHandleMessageColumnOutputFormat(t *testing.T) {
	opts := CLIOptions{
		Message:      messageFlag{isSet: true, value: "true"},
		ColumnOutput: true,
	}
	out := captureOutput(func() {
		result := handleMessage(opts, fullSchema)
		if !result {
			t.Error("Expected handleMessage to return true for -message=true with ColumnOutput")
		}
	})
	if !strings.Contains(out, "Logon") {
		t.Error("Expected formatted message in column output")
	}
}

func TestHandleTagListAllTags(t *testing.T) {
	opts := CLIOptions{
		Tag:          tagFlag{isSet: true, value: "true"},
		ColumnOutput: false,
	}
	out := captureOutput(func() {
		result := handleTag(opts, fullSchema)
		if !result {
			t.Error("Expected handleTag to return true for bare -tag")
		}
	})
	if !strings.Contains(out, "MsgType") {
		t.Error("Expected tag listing in output")
	}
}
