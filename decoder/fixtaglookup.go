// fixtaglookup.go
package decoder

import (
	"encoding/xml"
	"strconv"
	"strings"
	"sync"

	"bitbucket.org/edgewater/fixdecoder/fix"
	"golang.org/x/net/html/charset"
)

var chooseEmbeddedXML = fix.ChooseEmbeddedXML

type rawFix struct {
	Fields []struct {
		XMLName xml.Name `xml:"field"`
		Name    string   `xml:"name,attr"`
		Tag     int      `xml:"number,attr"`
		Type    string   `xml:"type,attr"`

		Values []struct {
			Enum        string `xml:"enum,attr"`
			Description string `xml:"description,attr"`
		} `xml:"value"`

		ValuesWrapper []struct {
			Enum        string `xml:"enum,attr"`
			Description string `xml:"description,attr"`
		} `xml:"values>value"`
	} `xml:"fields>field"`

	Messages []struct {
		XMLName xml.Name `xml:"message"`
		Name    string   `xml:"name,attr"`
		MsgType string   `xml:"msgtype,attr"`

		Fields []struct {
			Name     string `xml:"name,attr"`
			Required string `xml:"required,attr"` // "Y" or "N"
		} `xml:"field"`
	} `xml:"messages>message"`

	Groups []struct {
		NumInGroup int   `xml:"numInGroup,attr"`
		Tags       []int `xml:"field"`
	} `xml:"groups>group"`
}

type MessageDef struct {
	Name       string
	MsgType    string
	FieldOrder []int
	Required   []int
}

type GroupDef struct {
	NumInGroupTag int
	FieldOrder    []int
}

type FixTagLookup struct {
	tagToName   map[int]string
	enumMap     map[int]map[string]string
	fieldTypes  map[int]string
	groupCounts map[int]bool
	groupOwners map[int]int
	groupDefs   map[int]GroupDef
	Messages    map[string]MessageDef
}

func parseDictionary(xmlData string) (*FixTagLookup, error) {
	dec := xml.NewDecoder(strings.NewReader(xmlData))
	dec.CharsetReader = charset.NewReaderLabel

	var raw rawFix
	if err := dec.Decode(&raw); err != nil {
		return nil, err
	}

	d := &FixTagLookup{
		tagToName:   make(map[int]string, len(raw.Fields)),
		enumMap:     make(map[int]map[string]string, len(raw.Fields)),
		fieldTypes:  make(map[int]string, len(raw.Fields)),
		groupCounts: make(map[int]bool),
		groupOwners: make(map[int]int),
		groupDefs:   make(map[int]GroupDef),
		Messages:    make(map[string]MessageDef),
	}

	parseFields(&raw, d)
	parseMessages(&raw, d)
	parseGroups(&raw, d)

	return d, nil
}

func parseFields(raw *rawFix, d *FixTagLookup) {
	for _, f := range raw.Fields {
		d.tagToName[f.Tag] = f.Name
		d.fieldTypes[f.Tag] = f.Type

		enumMap := make(map[string]string, len(f.Values)+len(f.ValuesWrapper))
		for _, v := range f.Values {
			enumMap[v.Enum] = v.Description
		}
		for _, v := range f.ValuesWrapper {
			enumMap[v.Enum] = v.Description
		}

		if len(enumMap) > 0 {
			d.enumMap[f.Tag] = enumMap
		}
	}
}

func parseMessages(raw *rawFix, d *FixTagLookup) {
	const msgTypeTag = 35

	for _, msg := range raw.Messages {
		fieldOrder, required := extractMessageFields(msg, d)

		d.Messages[msg.MsgType] = MessageDef{
			Name:       msg.Name,
			MsgType:    msg.MsgType,
			FieldOrder: fieldOrder,
			Required:   required,
		}

		addMsgTypeEnumDescription(msgTypeTag, msg.MsgType, msg.Name, d)
	}
}

func extractMessageFields(msg struct {
	XMLName xml.Name `xml:"message"`
	Name    string   `xml:"name,attr"`
	MsgType string   `xml:"msgtype,attr"`
	Fields  []struct {
		Name     string `xml:"name,attr"`
		Required string `xml:"required,attr"`
	} `xml:"field"`
}, d *FixTagLookup) ([]int, []int) {
	var fieldOrder []int
	var required []int

	for _, f := range msg.Fields {
		if tag := resolveTagByName(f.Name, d.tagToName); tag != -1 {
			fieldOrder = append(fieldOrder, tag)
			if f.Required == "Y" {
				required = append(required, tag)
			}
		}
	}

	return fieldOrder, required
}

func resolveTagByName(name string, tagToName map[int]string) int {
	for tag, n := range tagToName {
		if n == name {
			return tag
		}
	}
	return -1
}

func addMsgTypeEnumDescription(msgTypeTag int, msgType string, name string, d *FixTagLookup) {
	if _, ok := d.enumMap[msgTypeTag]; !ok {
		d.enumMap[msgTypeTag] = make(map[string]string)
	}
	d.enumMap[msgTypeTag][msgType] = name
}

func parseGroups(raw *rawFix, d *FixTagLookup) {
	for _, group := range raw.Groups {
		d.groupCounts[group.NumInGroup] = true

		for _, tag := range group.Tags {
			d.groupOwners[tag] = group.NumInGroup
		}

		d.groupDefs[group.NumInGroup] = GroupDef{
			NumInGroupTag: group.NumInGroup,
			FieldOrder:    group.Tags,
		}
	}
}

// getTagValue pulls the value for a given FIX tag out of the message.
func getTagValue(msg, tag string) (string, bool) {
	const soh = "\x01"

	for _, f := range strings.Split(msg, soh) {
		if f == "" {
			continue
		}

		kv := strings.SplitN(f, "=", 2)
		if len(kv) == 2 && kv[0] == tag {
			return kv[1], true
		}
	}
	return "", false
}

// detectSchemaKey returns our internal dictionary key for a FIX message.
func detectSchemaKey(msg string) string {
	begin, ok := getTagValue(msg, "8")
	if !ok {
		return "FIX44" // default if BeginString is missing
	}

	if begin == "FIXT.1.1" {
		appl, _ := getTagValue(msg, "1128")

		switch appl {
		case "0":
			return "FIX27"
		case "1":
			return "FIX30"
		case "2":
			return "FIX40"
		case "3":
			return "FIX41"
		case "4":
			return "FIX42"
		case "5":
			return "FIX43"
		case "6":
			return "FIX44"
		case "7":
			return "FIX50"
		case "8":
			return "FIX50SP1"
		case "9":
			return "FIX50SP2"
		default:
			return "FIX50"
		}
	}

	// Classic BeginString – e.g. FIX.4.2 → FIX42
	return strings.ReplaceAll(begin, ".", "")
}

// mergeLookups grafts tags/enums from src into dst without overwriting.
func mergeLookups(dst, src *FixTagLookup) {
	if dst == nil || src == nil {
		return
	}

	for tag, name := range src.tagToName {
		if _, exists := dst.tagToName[tag]; !exists {
			dst.tagToName[tag] = name
		}
	}

	for tag, enumSrc := range src.enumMap {
		if _, ok := dst.enumMap[tag]; !ok {
			dst.enumMap[tag] = make(map[string]string, len(enumSrc))
		}

		for v, desc := range enumSrc {
			if _, ok := dst.enumMap[tag][v]; !ok {
				dst.enumMap[tag][v] = desc
			}
		}
	}
}

var (
	dicts   = make(map[string]*FixTagLookup) // schema-key → lookup
	dictMux sync.RWMutex                     // guards the map
)

// schema key → embedded-XML ID used by fix.ChooseEmbeddedXML
var schemaToXMLID = map[string]string{
	"FIX27":    "40", // ApplVerID 0 (FIX 2.7) – closest superset
	"FIX30":    "40", // ApplVerID 1 (FIX 3.0)
	"FIX40":    "40",
	"FIX41":    "41",
	"FIX42":    "42",
	"FIX43":    "43",
	"FIX44":    "44",
	"FIX50":    "50",
	"FIX50SP1": "50SP1",
	"FIX50SP2": "50SP2",
	"FIXT11":   "T11",
}

func getDictionary(key string) *FixTagLookup {
	// Fast path: read lock
	dictMux.RLock()
	if d, ok := dicts[key]; ok {
		dictMux.RUnlock()
		return d
	}
	dictMux.RUnlock()

	// Lookup XML ID
	xmlID, ok := schemaToXMLID[key]
	if !ok {
		return nil
	}

	// Parse dictionary without holding lock
	xmlBytes := chooseEmbeddedXML(xmlID)
	parsed, err := parseDictionary(xmlBytes)
	if err != nil {
		return nil
	}

	// Write to cache under lock
	dictMux.Lock()
	dicts[key] = parsed
	dictMux.Unlock()

	// Merge FIXT11 session tags if needed
	if key == "FIX50" || key == "FIX50SP1" || key == "FIX50SP2" {
		if t11 := getDictionary("FIXT11"); t11 != nil {
			mergeLookups(parsed, t11)
		}
	}

	return parsed
}

/* ---------- PUBLIC API ---------- */

func LoadDictionary(msg string) *FixTagLookup {
	key := detectSchemaKey(msg)
	if d := getDictionary(key); d != nil {
		return d
	}

	return getDictionary("FIX44") // safe fallback; never nil after first call
}

func (d *FixTagLookup) GetFieldName(tag int) string {
	if n, ok := d.tagToName[tag]; ok {
		return n
	}

	return strconv.Itoa(tag)
}

func (d *FixTagLookup) GetEnumDescription(tag int, val string) string {
	if m, ok := d.enumMap[tag]; ok {
		if desc, ok2 := m[val]; ok2 {
			return desc
		}
	}

	return ""
}

func (d *FixTagLookup) GetFieldType(tag int) string {
	return d.fieldTypes[tag]
}

func (d *FixTagLookup) IsGroupCountField(tag int) bool {
	return d.groupCounts[tag]
}

func (d *FixTagLookup) GetGroupOwner(tag int) int {
	return d.groupOwners[tag]
}
