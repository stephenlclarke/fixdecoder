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
	"regexp"
	"strconv"
	"strings"
	"time"
)

func ValidateFixMessage(msg string, dict *FixTagLookup) []string {
	fields := ParseFix(msg)
	fieldMap, seenTags := buildFieldMap(fields)

	var errors []string

	msgTypeErrors, msgDef := validateMsgType(fieldMap, dict)
	errors = append(errors, msgTypeErrors...)
	if msgDef == nil {
		return errors // can't continue without a known MsgType
	}

	errors = append(errors, validateRequiredFields(msgDef.Required, seenTags, dict)...)
	errors = append(errors, validateFieldEnumsAndTypes(fields, dict)...)
	errors = append(errors, validateFieldOrdering(fields, msgDef.FieldOrder)...)
	errors = append(errors, validateChecksumField(msg, fieldMap)...)

	return errors
}

func buildFieldMap(fields []FieldValue) (map[int]string, map[int]bool) {
	fieldMap := make(map[int]string)
	seenTags := make(map[int]bool)
	for _, fv := range fields {
		fieldMap[fv.Tag] = fv.Value
		seenTags[fv.Tag] = true
	}
	return fieldMap, seenTags
}

func validateMsgType(fieldMap map[int]string, dict *FixTagLookup) ([]string, *MessageDef) {
	msgType, ok := fieldMap[35]
	if !ok {
		return []string{"Missing required tag 35 (MsgType)"}, nil
	}
	msgDef, ok := dict.Messages[msgType]
	if !ok {
		return []string{fmt.Sprintf("Unknown MsgType: %s", msgType)}, nil
	}
	return nil, &msgDef
}

func validateRequiredFields(required []int, seenTags map[int]bool, dict *FixTagLookup) []string {
	var errors []string
	for _, tag := range required {
		if !seenTags[tag] {
			errors = append(errors, fmt.Sprintf("Missing required tag %d (%s)", tag, dict.GetFieldName(tag)))
		}
	}
	return errors
}

func validateFieldEnumsAndTypes(fields []FieldValue, dict *FixTagLookup) []string {
	var errors []string
	for _, fv := range fields {
		tag := fv.Tag
		val := fv.Value

		// Enums
		if enumMap, found := dict.enumMap[tag]; found {
			if _, valid := enumMap[val]; !valid {
				errors = append(errors, fmt.Sprintf("Invalid enum value '%s' for tag %d", val, tag))
			}
		}

		// Types
		typ := dict.GetFieldType(tag)
		if typ != "" && !IsValidType(val, typ) {
			errors = append(errors, fmt.Sprintf("Invalid type for tag %d: expected %s, got '%s'", tag, typ, val))
		}
	}
	return errors
}

func validateFieldOrdering(fields []FieldValue, expectedOrder []int) []string {
	orderIndex := make(map[int]int)
	for i, tag := range expectedOrder {
		orderIndex[tag] = i
	}

	var errors []string
	lastIdx := -1
	for _, fv := range fields {
		if idx, ok := orderIndex[fv.Tag]; ok {
			if idx < lastIdx {
				errors = append(errors, fmt.Sprintf("Tag %d out of order", fv.Tag))
			}
			lastIdx = idx
		}
	}
	return errors
}

func validateChecksumField(msg string, fieldMap map[int]string) []string {
	checkVal, ok := fieldMap[10]
	if !ok {
		return []string{"Missing required checksum tag 10"}
	}
	expected := fmt.Sprintf("%03d", CalculateChecksum(msg))
	if checkVal != expected {
		return []string{fmt.Sprintf("Checksum mismatch: got %s, expected %s", checkVal, expected)}
	}
	return nil
}

func CalculateChecksum(msg string) int {
	const soh = "\x01"
	cutoff := strings.Index(msg, soh+"10=")
	if cutoff == -1 {
		// If 10= tag is missing, checksum cannot be validated
		return -1
	}

	fragment := msg[:cutoff+1] // Include the SOH before 10=
	sum := 0
	for i := 0; i < len(fragment); i++ {
		sum += int(fragment[i])
	}
	return sum % 256
}

func IsValidType(val string, typ string) bool {
	switch strings.ToUpper(typ) {
	case "INT", "LENGTH", "NUMINGROUP", "SEQNUM", "DAYOFMONTH":
		_, err := strconv.Atoi(val)
		return err == nil
	case "FLOAT", "QTY", "PRICE", "PRICEOFFSET", "AMT", "PERCENTAGE":
		_, err := strconv.ParseFloat(val, 64)
		return err == nil
	case "BOOLEAN":
		return val == "Y" || val == "N"
	case "CHAR":
		return len(val) == 1
	case "STRING", "DATA", "CURRENCY", "EXCHANGE", "COUNTRY", "MULTIPLEVALUESTRING", "MULTIPLESTRINGVALUE":
		return true
	case "UTCTIMESTAMP":
		layouts := []string{"20060102-15:04:05", "20060102-15:04:05.000"}
		for _, layout := range layouts {
			if _, err := time.Parse(layout, val); err == nil {
				return true
			}
		}
		return false
	case "UTCDATEONLY":
		_, err := time.Parse("20060102", val)
		return err == nil
	case "UTCTIMEONLY":
		layouts := []string{"15:04", "15:04:05", "15:04:05.000"}
		for _, layout := range layouts {
			if _, err := time.Parse(layout, val); err == nil {
				return true
			}
		}
		return false
	case "MONTHYEAR":
		return regexp.MustCompile(`^\d{6}([0-9]{2}|(-[0-9]{1,2})|(-?w[1-5]))?$`).MatchString(val)
	default:
		return true // assume valid for unknown/custom types
	}
}
