package decoder

import (
	"fmt"
	"slices"
	"testing"
)

func TestCalculateChecksum(t *testing.T) {
	input := "8=FIX.4.4\x019=12\x0135=A\x0110=099\x01"
	expected := 226
	got := CalculateChecksum(input)

	if got != expected {
		t.Errorf("Expected checksum %d, got %d", expected, got)
	}
}

func TestCalculateChecksumMissingTag(t *testing.T) {
	input := "8=FIX.4.4\x019=12\x0135=A\x01"
	got := CalculateChecksum(input)

	if got != -1 {
		t.Errorf("Expected checksum -1 for missing tag, got %d", got)
	}
}

func TestIsValidTypeInt(t *testing.T) {
	valid := IsValidType("123", "INT")
	invalid := IsValidType("abc", "INT")

	if !valid || invalid {
		t.Errorf("INT validation failed")
	}
}

func TestIsValidTypeChar(t *testing.T) {
	valid := IsValidType("X", "CHAR")
	invalid := IsValidType("XY", "CHAR")

	if !valid || invalid {
		t.Errorf("CHAR validation failed")
	}
}

func TestIsValidTypeBoolean(t *testing.T) {
	cases := map[string]bool{
		"Y": true, "N": true, "X": false, "": false,
	}

	for input, expected := range cases {
		got := IsValidType(input, "BOOLEAN")

		if got != expected {
			t.Errorf("BOOLEAN type test failed for %q: expected %v", input, expected)
		}
	}
}

func TestIsValidTypeUTCTimestamp(t *testing.T) {
	valid1 := IsValidType("20230703-15:04:05", "UTCTIMESTAMP")
	valid2 := IsValidType("20230703-15:04:05.000", "UTCTIMESTAMP")
	invalid := IsValidType("invalid", "UTCTIMESTAMP")

	if !valid1 || !valid2 || invalid {
		t.Errorf("UTCTIMESTAMP validation failed")
	}
}

func TestIsValidTypeMonthYear(t *testing.T) {
	cases := map[string]bool{
		"202407":    true,
		"202407-w2": true,
		"20240709":  true,
		"07-2024":   false,
	}
	for input, expected := range cases {
		got := IsValidType(input, "MONTHYEAR")
		if got != expected {
			t.Errorf("MONTHYEAR test failed for %q: expected %v", input, expected)
		}
	}
}

func TestIsValidTypeGeneric(t *testing.T) {
	if !IsValidType("anything", "STRING") {
		t.Error("Expected STRING to be valid")
	}
	if !IsValidType("anything", "UNKNOWN_CUSTOM_TYPE") {
		t.Error("Expected unknown/custom types to be assumed valid")
	}
}

func setupTestDictionary() *FixTagLookup {
	return &FixTagLookup{
		tagToName: map[int]string{
			8:  "BeginString",
			9:  "BodyLength",
			10: "CheckSum",
			11: "ClOrdID",
			35: "MsgType",
			54: "Side",
		},
		fieldTypes: map[int]string{
			35: "STRING",
			11: "STRING",
			54: "CHAR",
		},
		enumMap: map[int]map[string]string{
			35: {"A": "Logon"},
			54: {"1": "Buy", "2": "Sell"},
		},
		Messages: map[string]MessageDef{
			"A": {
				Name:       "Logon",
				MsgType:    "A",
				FieldOrder: []int{35, 11, 54},
				Required:   []int{35, 11, 54},
			},
		},
	}
}

func TestValidateFixMessageValidMessage(t *testing.T) {
	dict := setupTestDictionary()

	base := "8=FIX.4.4\x019=23\x0135=A\x0111=ORDER123\x0154=1\x01"
	checksum := fmt.Sprintf("%03d", CalculateChecksum(base+"10=")) // Pass in fragment including SOH before 10=
	msg := base + "10=" + checksum + "\x01"

	errors := ValidateFixMessage(msg, dict)

	if len(errors) > 0 {
		t.Errorf("Expected no errors, got: %v", errors)
	}
}

func TestValidateFixMessageMissingRequiredField(t *testing.T) {
	dict := setupTestDictionary()

	base := "8=FIX.4.4\x019=23\x0135=A\x0154=1\x01" // Missing tag 11
	checksum := fmt.Sprintf("%03d", CalculateChecksum(base+"10="))
	msg := base + "10=" + checksum + "\x01"

	errors := ValidateFixMessage(msg, dict)
	expected := "Missing required tag 11 (ClOrdID)"
	found := slices.Contains(errors, expected)

	if !found {
		t.Errorf("Expected error %q, got: %v", expected, errors)
	}
}

func TestValidateFixMessageInvalidEnum(t *testing.T) {
	dict := setupTestDictionary()

	base := "8=FIX.4.4\x019=23\x0135=A\x0111=ORDER123\x0154=X\x01" // X is invalid enum
	checksum := fmt.Sprintf("%03d", CalculateChecksum(base+"10="))
	msg := base + "10=" + checksum + "\x01"

	errors := ValidateFixMessage(msg, dict)
	expected := "Invalid enum value 'X' for tag 54"
	found := slices.Contains(errors, expected)

	if !found {
		t.Errorf("Expected error %q, got: %v", expected, errors)
	}
}

func TestValidateFixMessageInvalidType(t *testing.T) {
	dict := setupTestDictionary()

	base := "8=FIX.4.4\x019=23\x0135=A\x0111=ORDER123\x0154=12\x01" // 12 is invalid CHAR
	checksum := fmt.Sprintf("%03d", CalculateChecksum(base+"10="))
	msg := base + "10=" + checksum + "\x01"

	errors := ValidateFixMessage(msg, dict)
	expected := "Invalid type for tag 54: expected CHAR, got '12'"
	found := slices.Contains(errors, expected)

	if !found {
		t.Errorf("Expected error %q, got: %v", expected, errors)
	}
}

func TestValidateFieldEnumsAndTypesInvalidEnum(t *testing.T) {
	fields := []FieldValue{
		{Tag: 54, Value: "X"}, // Invalid enum for tag 54
	}

	dict := setupTestDictionary()

	errors := validateFieldEnumsAndTypes(fields, dict)
	expected := "Invalid enum value 'X' for tag 54"

	if len(errors) == 0 || errors[0] != expected {
		t.Errorf("Expected error %q, got: %v", expected, errors)
	}
}

func TestValidateFixMessageTagOutOfOrder(t *testing.T) {
	dict := setupTestDictionary()

	base := "8=FIX.4.4\x019=23\x0135=A\x0154=1\x0111=ORDER123\x01" // Tag 11 comes after 54
	checksum := fmt.Sprintf("%03d", CalculateChecksum(base+"10="))
	msg := base + "10=" + checksum + "\x01"

	errors := ValidateFixMessage(msg, dict)
	expected := "Tag 11 out of order"
	found := slices.Contains(errors, expected)

	if !found {
		t.Errorf("Expected error %q, got: %v", expected, errors)
	}
}

func TestValidateChecksumFieldMissingTag10(t *testing.T) {
	msg := "8=FIX.4.4\x019=23\x0135=A\x0111=ORDER123\x0154=1\x01"
	fieldMap := map[int]string{
		8:  "FIX.4.4",
		9:  "23",
		35: "A",
		11: "ORDER123",
		54: "1",
		// tag 10 deliberately omitted
	}

	errors := validateChecksumField(msg, fieldMap)

	expected := "Missing required checksum tag 10"
	if len(errors) != 1 || errors[0] != expected {
		t.Errorf("Expected error %q, got: %v", expected, errors)
	}
}

func TestValidateChecksumFieldMismatch(t *testing.T) {
	msg := "8=FIX.4.4\x019=12\x0135=A\x01" // Note: intentionally incorrect checksum
	fieldMap := map[int]string{
		10: "000", // Invalid checksum value on purpose
	}

	errs := validateChecksumField(msg, fieldMap)

	if len(errs) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errs))
	}
	expected := fmt.Sprintf("%03d", CalculateChecksum(msg))
	expectedMsg := fmt.Sprintf("Checksum mismatch: got 000, expected %s", expected)
	if errs[0] != expectedMsg {
		t.Errorf("Unexpected error message:\nGot:  %s\nWant: %s", errs[0], expectedMsg)
	}
}

func TestValidateFixMessageMissingMsgType(t *testing.T) {
	dict := setupTestDictionary()

	msg := "8=FIX.4.4\x019=23\x0111=ORDER123\x0154=1\x0110=229\x01"
	errors := ValidateFixMessage(msg, dict)
	expected := "Missing required tag 35 (MsgType)"

	if len(errors) != 1 || errors[0] != expected {
		t.Errorf("Expected error %q, got: %v", expected, errors)
	}
}

func TestValidateMsgTypeMissingTag35(t *testing.T) {
	fieldMap := map[int]string{
		11: "ORDER123",
	}

	dict := setupTestDictionary()

	errors, def := validateMsgType(fieldMap, dict)

	if len(errors) != 1 || errors[0] != "Missing required tag 35 (MsgType)" {
		t.Errorf("Expected missing tag 35 error, got: %v", errors)
	}

	if def != nil {
		t.Error("Expected nil MessageDef when tag 35 is missing")
	}
}

func TestIsValidTypeFloatVariants(t *testing.T) {
	validInputs := []string{"123.45", "0", "-999.99"}
	invalidInputs := []string{"abc", "", "12.34.56"}

	types := []string{"FLOAT", "QTY", "PRICE", "PRICEOFFSET", "AMT", "PERCENTAGE"}

	for _, typ := range types {
		for _, val := range validInputs {
			if !IsValidType(val, typ) {
				t.Errorf("Expected %q to be valid for type %s", val, typ)
			}
		}
		for _, val := range invalidInputs {
			if IsValidType(val, typ) {
				t.Errorf("Expected %q to be invalid for type %s", val, typ)
			}
		}
	}
}

func TestIsValidTypeUTCDATEONLY(t *testing.T) {
	if !IsValidType("20250704", "UTCDATEONLY") {
		t.Error("Expected valid UTCDATEONLY format to pass")
	}
	if IsValidType("07-04-2025", "UTCDATEONLY") {
		t.Error("Expected invalid UTCDATEONLY format to fail")
	}
}

func TestIsValidTypeUTCTIMEONLY(t *testing.T) {
	valid := []string{"15:04", "15:04:05", "15:04:05.000"}
	invalid := []string{"3:04PM", "15:04:60", "invalid"}

	for _, v := range valid {
		if !IsValidType(v, "UTCTIMEONLY") {
			t.Errorf("Expected valid UTCTIMEONLY format: %s", v)
		}
	}
	for _, v := range invalid {
		if IsValidType(v, "UTCTIMEONLY") {
			t.Errorf("Expected invalid UTCTIMEONLY format to fail: %s", v)
		}
	}
}

func TestValidateMsgTypeUnknownType(t *testing.T) {
	fieldMap := map[int]string{
		35: "Z", // Unknown message type
	}
	dict := &FixTagLookup{
		Messages: map[string]MessageDef{
			"D": {MsgType: "D"},
		},
	}

	errs, def := validateMsgType(fieldMap, dict)

	if len(errs) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errs))
	}
	if errs[0] != "Unknown MsgType: Z" {
		t.Errorf("Unexpected error message: %s", errs[0])
	}
	if def != nil {
		t.Errorf("Expected nil MessageDef, got %+v", def)
	}
}
