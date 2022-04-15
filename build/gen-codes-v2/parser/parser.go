package parser

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

type EncodingType string

const (
	EncodingUnknown  EncodingType = ""
	EncodingHex      EncodingType = "hex"
	EncodingInteger  EncodingType = "int"
	EncodingText     EncodingType = "text"     // e.g. "SOME_OTHER_CONST_NAME"
	EncodingEquation EncodingType = "equation" // e.g. "(CONST_NAME + 1)"
)

const (
	stateNeutral = iota
	stateReadingMultilineComment
	stateReadingDefineWithMultilineComment
)

var regexHex = regexp.MustCompile(`^0x[0-9a-fA-F]+$`)
var regexInteger = regexp.MustCompile(`^[0-9]+$`)
var regexText = regexp.MustCompile(`^\w+$`)
var regexEquation = regexp.MustCompile(`^\((\w+)\s*(\+)\s*(\d)+\)$`)

func detectEncodingType(value string) (EncodingType, int) {
	var convertedValue int64
	var err error

	switch {
	case regexHex.MatchString(value):
		convertedValue, err = strconv.ParseInt(value[2:], 16, 64)
		if err != nil {
			panic(err)
		}
		return EncodingHex, int(convertedValue)
	case regexInteger.MatchString(value):
		convertedValue, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			panic(err)
		}
		return EncodingInteger, int(convertedValue)
	case regexEquation.MatchString(value):
		return EncodingEquation, 0 // can't resolve value that points on different constant here
	case regexText.MatchString(value):
		return EncodingText, 0 // can't resolve value that points on different constant here
	default:
		return EncodingUnknown, 0
	}
}

var typeNames = map[string]string{
	"EV_":         "EvType",
	"INPUT_PROP_": "EvProp",
}

func getType(in string) string {
	for k, v := range typeNames {
		if strings.HasPrefix(in, k) {
			return v
		}
	}
	return "EvCode"
}

type separator string // one line

type comment string // may be bigger than one line

type constant struct {
	name         string
	value        string
	decodedValue int
	comment      string // oneliner
	encodingType EncodingType
}

func (c *constant) resolveEquation(group *Group) (EncodingType, int, error) {
	out := regexEquation.FindStringSubmatch(c.value)
	rawA, sign, rawB := out[1], out[2], out[3]
	if sign != "+" {
		return EncodingUnknown, 0, errors.New("e1")
	}
	i, err := strconv.ParseInt(rawB, 10, 64)
	if err != nil {
		return EncodingUnknown, 0, errors.New("e2")
	}

	valName, valDecoded := rawA, int(i)

	var found, ok bool
	var v2 constant
	for _, e := range group.elements {
		v2, ok = e.(constant)
		if !ok {
			continue
		}
		if v2.name == valName {
			found = true
			break
		}
	}
	if !found {
		return EncodingUnknown, 0, errors.New("value not found")
	}

	return v2.encodingType, v2.decodedValue + valDecoded, nil
}

func (c *constant) resolveText(group *Group) (EncodingType, int, error) {
	var found, ok bool
	var v2 constant
	for _, e := range group.elements {
		v2, ok = e.(constant)
		if !ok {
			continue
		}
		if v2.name == c.value {
			found = true
			break
		}
	}
	if !found {
		return EncodingUnknown, 0, errors.New("value not found")
	}

	return v2.encodingType, v2.decodedValue, nil
}

// Group contains list of elements (separator, comment or constant)
type Group struct {
	comment  string
	elements []interface{}
}

func NewGroup() Group {
	return Group{
		comment:  "",
		elements: make([]interface{}, 0),
	}
}

var SelectedPrefixesGroups = map[string][][]string{
	// "Group" detection relies on the correct order of these set of prefixes
	// this order should reflect exact order of groups in related source files
	"input.h": {
		{"ID_"},
		{"BUS_"},
		{"MT_TOOL_"},
		{"FF_"},
	},
	"input-event-codes.h": {
		{"INPUT_PROP_"},
		{"EV_"},
		{"SYN_"},
		{"KEY_", "BTN_"}, // these have different prefixes but belongs to the same group type
		{"REL_"},
		{"ABS_"},
		{"SW_"},
		{"MSC_"},
		{"LED_"},
		{"REP_"},
		{"SND_"},
	},
}

type CodeProcessor struct {
	prefixesGroups [][]string

	elements    []interface{}
	lastComment string
	state       int
}

func NewCodeProcessor(prefixesGroups [][]string) CodeProcessor {
	return CodeProcessor{
		prefixesGroups: prefixesGroups,

		elements:    make([]interface{}, 0),
		lastComment: "",
		state:       stateNeutral,
	}
}

var (
	reDefineWithComment      = regexp.MustCompile(`^#define[ \t]+(?P<name>[A-Z0-9_]+)[ \t]+(?P<value>[0-9a-fA-Zx()_+ ]+)[ \t]+/\*(?P<comment>.*)\*/[ \t]*$`)
	reDefineWithCommentStart = regexp.MustCompile(`^#define[ \t]+(?P<name>[A-Z0-9_]+)[ \t]+(?P<value>[0-9a-fA-Zx()_+ ]+)[ \t]+/\*(?P<comment>.*)[ \t]*$`)
	reDefine                 = regexp.MustCompile(`^#define[ \t]+(?P<name>[A-Z0-9_]+)[ \t]+(?P<value>[0-9a-fA-Zx()_+ ]+)`)
	reOnelineComment         = regexp.MustCompile(`^[ \t]*/\*(?P<text>.*)\*/[ \t]*$`)
	reMultilineCommentStart  = regexp.MustCompile(`^[ \t]*/\*(?P<text>.*)[ \t]*$`)
	reMultilineCommentEnd    = regexp.MustCompile(`^[ \t](?:\*)?(?P<text>.*)\*/[ \t]*$`)
	reMultilineCommentMid    = regexp.MustCompile(`^[ \t]*(?:\*)?(?P<text>.*)[ \t]*$`)
)

func hasPrefixInSlice(v string, s []string) bool {
	for _, vs := range s {
		if strings.HasPrefix(v, vs) {
			return true
		}
	}
	return false
}

func (p *CodeProcessor) ProcessFile(file io.Reader) ([]Group, error) {
	var groups = make([]Group, 0)
	var group = NewGroup()

	var variable constant
	var suffixCounter int
	var lastComment string
	var sepAfterComment bool
	var groupCommentAdded bool
	var seeking = true
	var currentPrefixes = p.prefixesGroups[suffixCounter]

	scanner := bufio.NewScanner(file)

scanning:
	for scanner.Scan() {
		s := scanner.Text()

		switch p.state {
		case stateNeutral:
			if match := reDefine.FindStringSubmatch(s); match != nil {
				if !hasPrefixInSlice(match[1], currentPrefixes) { // exiting type group
					if seeking {
						continue // seeking through #define elements and preserving last comment in the meantime
					}
					groups = append(groups, group)
					suffixCounter++
					if suffixCounter == len(p.prefixesGroups) {
						break scanning
					}
					group = NewGroup()
					groupCommentAdded = false
					currentPrefixes = p.prefixesGroups[suffixCounter]
				}
				seeking = false

				if lastComment != "" && !groupCommentAdded {
					group.comment = lastComment
					groupCommentAdded = true
					lastComment = ""
				}

				if lastComment != "" {
					group.elements = append(group.elements, comment(lastComment))
					lastComment = ""
					if sepAfterComment {
						group.elements = append(group.elements, separator(""))
					}
				}
			}

			if match := reDefineWithComment.FindStringSubmatch(s); match != nil {
				variable.name, variable.value, variable.comment = match[1], match[2], strings.Trim(match[3], " \t")
				variable.value = strings.Trim(variable.value, " \t")
				variable.encodingType, variable.decodedValue = detectEncodingType(variable.value)
				group.elements = append(group.elements, variable)
				continue
			}

			if match := reDefineWithCommentStart.FindStringSubmatch(s); match != nil {
				variable.name, variable.value, variable.comment = match[1], match[2], strings.Trim(match[3], " \t")
				variable.value = strings.Trim(variable.value, " \t")
				variable.encodingType, variable.decodedValue = detectEncodingType(variable.value)
				p.state = stateReadingDefineWithMultilineComment
				continue
			}

			if match := reDefine.FindStringSubmatch(s); match != nil {
				variable.name, variable.value, variable.comment = match[1], match[2], ""
				variable.value = strings.Trim(variable.value, " \t")
				variable.encodingType, variable.decodedValue = detectEncodingType(variable.value)
				group.elements = append(group.elements, variable)
				continue
			}

			if match := reOnelineComment.FindStringSubmatch(s); match != nil {
				lastComment = strings.Trim(match[1], " \t")
				continue
			}

			if match := reMultilineCommentStart.FindStringSubmatch(s); match != nil {
				lastComment = ""
				if len(match[1]) > 0 {
					lastComment += strings.Trim(match[1], " \t")
				}
				p.state = stateReadingMultilineComment
				continue
			}

			if s == "" {
				sepAfterComment = true
				group.elements = append(group.elements, separator(""))
				continue
			} else {
				sepAfterComment = false
			}

		case stateReadingDefineWithMultilineComment:
			if match := reMultilineCommentEnd.FindStringSubmatch(s); match != nil {
				if len(match[1]) > 0 {
					variable.comment += strings.Trim(match[1], " \t") + "\n"
				}
				group.elements = append(group.elements, variable)
				p.state = stateNeutral
				continue
			}

			if match := reMultilineCommentMid.FindStringSubmatch(s); match != nil {
				if len(match[1]) > 0 {
					variable.comment += strings.Trim(match[1], " \t")
				}
				continue
			}

			return groups, errors.New("processing should not reach this point: #1")

		case stateReadingMultilineComment:
			if match := reMultilineCommentEnd.FindStringSubmatch(s); match != nil {
				if len(match[1]) > 0 {
					lastComment += " " + strings.Trim(match[1], " \t")
				}
				lastComment += match[1]
				p.state = stateNeutral
				continue
			}

			if match := reMultilineCommentMid.FindStringSubmatch(s); match != nil {
				if len(match[1]) > 0 {
					lastComment += strings.Trim(match[1], " \t") + "\n"
				}
				continue
			}

			return groups, errors.New("processing should not reach this point: #2")
		}
	}
	groups = append(groups, group)
	return groups, nil
}

// stripSeparators removes separators from the begging and the end
func stripSeparators(elements []interface{}) []interface{} {
	var start, stop int

	for i, e := range elements {
		_, ok := e.(separator)
		if !ok {
			start = i
			break
		}
	}

	for i, e := range elements {
		_, ok := e.(separator)
		if !ok {
			stop = i
		}
	}

	return elements[start : stop+1]
}

type decodedValue struct {
	value        int
	encodingType EncodingType
}

func removeRepeatedSeparators(groups []Group) []Group {
	var newGroups = make([]Group, 0)

	for _, group := range groups {
		newGroup := Group{
			comment:  group.comment,
			elements: make([]interface{}, 0),
		}

		var sepPreviously bool
		for _, element := range group.elements {
			switch element.(type) {
			case separator:
				if !sepPreviously {
					newGroup.elements = append(newGroup.elements, separator(""))
					sepPreviously = true
				}
			default:
				sepPreviously = false
				newGroup.elements = append(newGroup.elements, element)
			}

		}

		newGroups = append(newGroups, newGroup)
	}

	return newGroups
}

func generateSections(rawGroups []Group, disableComments bool) (codes, typeToString, typeFromString string) {
	groups := removeRepeatedSeparators(rawGroups)

	var decodedValues = make(map[constant]decodedValue)

	for _, group := range groups {
		for _, element := range group.elements {
			switch v := element.(type) {
			case constant:
				switch v.encodingType {
				case EncodingEquation:
					encType, valDecoded, err := v.resolveEquation(&group)
					if err != nil {
						panic(fmt.Errorf("failed to decode equation: %w", err))
					}
					decodedValues[v] = decodedValue{valDecoded, encType}
				case EncodingText:
					encType, valDecoded, err := v.resolveText(&group)
					if err != nil {
						panic(fmt.Errorf("failed to decode text: %w", err))
					}
					decodedValues[v] = decodedValue{valDecoded, encType}
				case EncodingInteger:
					i, err := strconv.ParseInt(v.value, 10, 64)
					if err != nil {
						panic(fmt.Errorf("integer conversion failed: %w", err))
					}
					decodedValues[v] = decodedValue{int(i), EncodingInteger}
				case EncodingHex:
					i, err := strconv.ParseInt(v.value[2:], 16, 64)
					if err != nil {
						panic(fmt.Errorf("hex conversion failed: %w", err))
					}
					decodedValues[v] = decodedValue{int(i), EncodingHex}
				}
			}
		}
	}

	for i, group := range groups {
		if !disableComments && group.comment != "" {
			lines := strings.Split(group.comment, "\n")
			for i := 0; i < len(lines); i++ {
				cl := lines[i]
				if cl != "" {
					codes += "// " + cl + "\n"
				}
			}
		}

		codes += "const (\n"
		for _, element := range stripSeparators(group.elements) {
			switch v := element.(type) {
			case separator: // space between blocks
				codes += "\n"
			case comment: // single comment entry
				if disableComments {
					continue
				}
				lines := strings.Split(string(v), "\n")
				for i := 0; i < len(lines); i++ {
					cl := lines[i]
					if cl != "" {
						codes += "\t// " + cl + "\n"
					}
				}
			case constant:
				if !disableComments && v.comment != "" {
					codes += fmt.Sprintf("\t%s = %s // %s\n", v.name, v.value, v.comment)
				} else {
					codes += fmt.Sprintf("\t%s = %s\n", v.name, v.value)
				}
			default:
				panic("")
			}
		}
		codes += ")\n"

		if i < len(groups) {
			codes += "\n"
		}
	}

	// generating additional mappings
	for i, group := range groups {
		var firstConstant constant

		for _, element := range group.elements {
			c, ok := element.(constant)
			if !ok {
				continue
			}
			firstConstant = c
			break
		}

		parts := strings.Split(firstConstant.name, "_")
		name := parts[0]

		var valueRegistered = make(map[int]string)

		typeToString += fmt.Sprintf("var %sToString = map[%s]string{\n", name, getType(firstConstant.name))
		for _, element := range stripSeparators(group.elements) {
			switch v := element.(type) {
			case separator:
				typeToString += "\n"
			case constant:
				decoded, ok := decodedValues[v]
				if !ok {
					panic("decoded value not found")
				}
				if name, ok := valueRegistered[decoded.value]; ok {
					typeToString += fmt.Sprintf("\t// %s: \"%s\", // (%s)\n", v.name, v.name, name)
					continue
				}

				valueRegistered[decoded.value] = v.name
				switch decoded.encodingType {
				case EncodingHex, EncodingInteger:
					typeToString += fmt.Sprintf("\t%s: \"%s\",\n", v.name, v.name)
				default:
					panic("unexpected encoding type")
				}
			}
		}

		typeFromString += fmt.Sprintf("var %sFromString = map[string]%s{\n", name, getType(firstConstant.name))
		for _, element := range stripSeparators(group.elements) {
			switch v := element.(type) {
			case separator:
				typeFromString += "\n"
			case constant:
				typeFromString += fmt.Sprintf("\t\"%s\": %s,\n", v.name, v.name)
			}
		}

		typeToString += "}\n"
		typeFromString += "}\n"

		if i < len(groups) {
			typeToString += "\n"
			typeFromString += "\n"
		}
	}
	return
}

func GenerateFile(rawGroups []Group, disableComments bool, selectedTag, inputHURL, eventCodesURL string) string {
	var file string
	codes, typeToString, typeFromString := generateSections(rawGroups, disableComments)

	file += "// Code generated by build/gen-codes-v2. DO NOT EDIT.\n"
	file += fmt.Sprintf("// version tag: \"%s\", generated from files:\n", selectedTag)
	file += fmt.Sprintf("// - %s\n", inputHURL)
	file += fmt.Sprintf("// - %s\n\n", eventCodesURL)
	file += "package evdev\n\n"
	file += codes

	file += "\n// Type to String\n\n"
	file += typeToString

	file += "\n// Type from String\n\n"
	file += typeFromString

	return file
}
