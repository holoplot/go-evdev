package parser

import (
	"bufio"
	"errors"
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

var (
	regexHex      = regexp.MustCompile(`^0x[0-9a-fA-F]+$`)
	regexInteger  = regexp.MustCompile(`^[0-9]+$`)
	regexText     = regexp.MustCompile(`^\w+$`)
	regexEquation = regexp.MustCompile(`^\((\w+)\s*(\+)\s*(\d)+\)$`)
)

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
