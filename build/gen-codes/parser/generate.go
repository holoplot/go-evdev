package parser

import (
	"fmt"
	"strconv"
	"strings"
)

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

func generateSections(rawGroups []Group, disableComments bool) (codes, typeToString, typeFromString, typeNames string) {
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

				switch decoded.encodingType {
				case EncodingHex, EncodingInteger:
					typeToString += fmt.Sprintf("\t%s: \"%s\",\n", v.name, v.name)
					valueRegistered[decoded.value] = v.name
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

		valueRegistered = make(map[int]string)
		duplicates := make(map[int][]string)
		for _, e := range group.elements {
			re, ok := e.(constant)
			if !ok {
				continue
			}
			decoded, ok := decodedValues[re]
			if !ok {
				panic("decoded value not found")
			}

			duplicates[decoded.value] = append(duplicates[decoded.value], re.name)
		}

		typeNames += fmt.Sprintf("var %sNames = map[%s]string{\n", name, getType(firstConstant.name))
		for _, element := range stripSeparators(group.elements) {
			switch v := element.(type) {
			case separator:
				typeNames += "\n"
			case constant:
				decoded, ok := decodedValues[v]
				if !ok {
					panic("decoded value not found")
				}

				if _, ok := valueRegistered[decoded.value]; ok {
					continue
				}

				var names string
				for i, name := range duplicates[decoded.value] {
					names += name
					if i != len(duplicates[decoded.value])-1 {
						names += "/"
					}
				}

				switch decoded.encodingType {
				case EncodingHex, EncodingInteger:
					typeNames += fmt.Sprintf("\t%s: \"%s\",\n", v.name, names)
					valueRegistered[decoded.value] = v.name
				default:
					panic("unexpected encoding type")
				}
			}
		}

		typeToString += "}\n"
		typeFromString += "}\n"
		typeNames += "}\n"

		if i < len(groups) {
			typeToString += "\n"
			typeFromString += "\n"
			typeNames += "\n"
		}
	}
	return
}

func GenerateFile(rawGroups []Group, disableComments bool, selectedTag, inputHURL, eventCodesURL string) string {
	var file string
	codes, typeToString, typeFromString, typeNames := generateSections(rawGroups, disableComments)

	file += "// Code generated by build/gen-codes. DO NOT EDIT.\n"
	file += fmt.Sprintf("// version tag: \"%s\", generated from files:\n", selectedTag)
	file += fmt.Sprintf("// - %s\n", inputHURL)
	file += fmt.Sprintf("// - %s\n\n", eventCodesURL)
	file += "package evdev\n\n"
	file += codes

	file += "\n//\n// Type to String\n//\n\n"
	file += typeToString

	file += "\n//\n// Type from String\n//\n\n"
	file += typeFromString

	file += "\n//\n// Type Names, useful for information/debug use only.\n"
	file += "// When one code has two or more string representations, all available aliases are provided seperated by a slash.\n"
	file += "// Example: KEY_COFFEE: \"KEY_COFFEE/KEY_SCREENLOCK\"\n//\n\n"
	file += typeNames

	return file
}
