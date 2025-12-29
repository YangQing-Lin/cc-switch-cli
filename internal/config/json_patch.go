package config

import (
	"bytes"
	"fmt"
	"sort"
)

type jsonUpdate struct {
	key    string
	value  []byte // JSON-encoded value (object/string/number/literal). Ignored when del=true.
	del    bool
	insert bool // allow inserting when missing (for set ops)
}

type jsonMember struct {
	key        string
	keyStart   int
	valueStart int
	valueEnd   int
}

type jsonBytePatch struct {
	start int
	end   int
	repl  []byte
}

func jsonSkipWS(b []byte, i int) int {
	for i < len(b) {
		switch b[i] {
		case ' ', '\t', '\r', '\n':
			i++
		default:
			return i
		}
	}
	return i
}

func jsonScanString(b []byte, i int) (start, end int, s []byte, ok bool) {
	if i >= len(b) || b[i] != '"' {
		return 0, 0, nil, false
	}
	start = i
	i++
	for i < len(b) {
		switch b[i] {
		case '\\':
			i += 2
		case '"':
			end = i + 1
			return start, end, b[start:end], true
		default:
			i++
		}
	}
	return 0, 0, nil, false
}

func jsonScanValueEnd(b []byte, i int) (end int, ok bool) {
	i = jsonSkipWS(b, i)
	if i >= len(b) {
		return 0, false
	}
	switch b[i] {
	case '"':
		_, end, _, ok = jsonScanString(b, i)
		return end, ok
	case '{':
		return jsonScanBalanced(b, i, '{', '}')
	case '[':
		return jsonScanBalanced(b, i, '[', ']')
	case 't':
		if bytes.HasPrefix(b[i:], []byte("true")) {
			return i + 4, true
		}
	case 'f':
		if bytes.HasPrefix(b[i:], []byte("false")) {
			return i + 5, true
		}
	case 'n':
		if bytes.HasPrefix(b[i:], []byte("null")) {
			return i + 4, true
		}
	default:
		// number
		if (b[i] >= '0' && b[i] <= '9') || b[i] == '-' {
			j := i + 1
			for j < len(b) {
				c := b[j]
				switch {
				case (c >= '0' && c <= '9') || c == '.' || c == 'e' || c == 'E' || c == '+' || c == '-':
					j++
				default:
					return j, true
				}
			}
			return j, true
		}
	}
	return 0, false
}

func jsonScanBalanced(b []byte, i int, open, close byte) (end int, ok bool) {
	if i >= len(b) || b[i] != open {
		return 0, false
	}
	depth := 0
	inString := false
	escape := false
	for i < len(b) {
		c := b[i]
		if inString {
			if escape {
				escape = false
				i++
				continue
			}
			if c == '\\' {
				escape = true
				i++
				continue
			}
			if c == '"' {
				inString = false
			}
			i++
			continue
		}

		switch c {
		case '"':
			inString = true
			i++
		case open:
			depth++
			i++
		case close:
			depth--
			i++
			if depth == 0 {
				return i, true
			}
		default:
			i++
		}
	}
	return 0, false
}

func jsonDetectIndent(b []byte) (newline []byte, indent []byte) {
	nlIdx := bytes.IndexByte(b, '\n')
	if nlIdx < 0 {
		return nil, nil
	}
	newline = []byte{'\n'}
	i := nlIdx + 1
	start := i
	for i < len(b) && (b[i] == ' ' || b[i] == '\t') {
		i++
	}
	if i < len(b) && b[i] == '"' {
		indent = b[start:i]
	}
	return newline, indent
}

func jsonParseTopLevelObjectMembers(b []byte) (objStart, objEnd int, members []jsonMember, ok bool) {
	i := jsonSkipWS(b, 0)
	if i >= len(b) || b[i] != '{' {
		return 0, 0, nil, false
	}
	objStart = i
	end, ok := jsonScanValueEnd(b, objStart)
	if !ok {
		return 0, 0, nil, false
	}
	objEnd = end

	i = objStart + 1
	for {
		i = jsonSkipWS(b, i)
		if i >= objEnd {
			return 0, 0, nil, false
		}
		if b[i] == '}' {
			break
		}
		if b[i] == ',' {
			i++
			continue
		}

		keyStart, keyEnd, rawKey, ok := jsonScanString(b, i)
		if !ok {
			return 0, 0, nil, false
		}
		i = jsonSkipWS(b, keyEnd)
		if i >= objEnd || b[i] != ':' {
			return 0, 0, nil, false
		}
		i++
		i = jsonSkipWS(b, i)
		valueStart := i
		valueEnd, ok := jsonScanValueEnd(b, valueStart)
		if !ok {
			return 0, 0, nil, false
		}
		i = valueEnd

		members = append(members, jsonMember{
			key:        string(bytes.Trim(rawKey, `"`)),
			keyStart:   keyStart,
			valueStart: valueStart,
			valueEnd:   valueEnd,
		})
	}

	return objStart, objEnd, members, true
}

func jsonFindTrailingWSBeforeClose(b []byte, objEnd int) int {
	// objEnd points to index after matching '}' of the top-level object.
	closeIdx := objEnd - 1
	i := closeIdx
	for i > 0 {
		c := b[i-1]
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			break
		}
		i--
	}
	return i
}

func jsonPatchTopLevelObject(b []byte, updates []jsonUpdate) ([]byte, error) {
	objStart, objEnd, members, ok := jsonParseTopLevelObjectMembers(b)
	if !ok {
		return nil, fmt.Errorf("invalid top-level json object")
	}

	memberByKey := make(map[string]jsonMember, len(members))
	for _, m := range members {
		memberByKey[m.key] = m
	}

	var patches []jsonBytePatch

	// replace/delete existing keys
	for _, u := range updates {
		m, exists := memberByKey[u.key]
		if !exists {
			continue
		}
		if !u.del {
			patches = append(patches, jsonBytePatch{start: m.valueStart, end: m.valueEnd, repl: u.value})
			continue
		}

		// delete: remove member + (prefer) trailing comma; otherwise remove preceding comma.
		delStart := m.keyStart
		delEnd := m.valueEnd

		after := jsonSkipWS(b, delEnd)
		if after < objEnd && b[after] == ',' {
			after++
			after = jsonSkipWS(b, after)
			delEnd = after
		} else {
			before := delStart
			for before > objStart+1 {
				c := b[before-1]
				if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
					before--
					continue
				}
				if c == ',' {
					delStart = before - 1
				}
				break
			}
		}

		patches = append(patches, jsonBytePatch{start: delStart, end: delEnd, repl: nil})
	}

	// Apply replacements/deletions from end to start.
	if len(patches) > 0 {
		sort.Slice(patches, func(i, j int) bool {
			if patches[i].start == patches[j].start {
				return patches[i].end > patches[j].end
			}
			return patches[i].start > patches[j].start
		})
		out := append([]byte(nil), b...)
		for _, p := range patches {
			if p.start < 0 || p.start > len(out) || p.end < p.start || p.end > len(out) {
				continue
			}
			out = append(out[:p.start], append(p.repl, out[p.end:]...)...)
		}
		b = out
	}

	// Re-parse after structural edits for insertion offsets.
	objStart, objEnd, members, ok = jsonParseTopLevelObjectMembers(b)
	if !ok {
		return nil, fmt.Errorf("invalid json after patch")
	}
	memberByKey = make(map[string]jsonMember, len(members))
	for _, m := range members {
		memberByKey[m.key] = m
	}

	newline, indent := jsonDetectIndent(b)
	insertPos := jsonFindTrailingWSBeforeClose(b, objEnd)

	// insert missing keys
	var inserts []byte
	hasMembers := len(members) > 0
	for _, u := range updates {
		if u.del || !u.insert {
			continue
		}
		if _, exists := memberByKey[u.key]; exists {
			continue
		}
		if hasMembers || len(inserts) > 0 {
			inserts = append(inserts, ',')
		}
		if newline != nil {
			inserts = append(inserts, newline...)
			inserts = append(inserts, indent...)
		} else if hasMembers || len(inserts) > 0 {
			inserts = append(inserts, ' ')
		}
		inserts = append(inserts, '"')
		inserts = append(inserts, []byte(u.key)...)
		inserts = append(inserts, '"', ':')
		if newline != nil {
			inserts = append(inserts, ' ')
		}
		inserts = append(inserts, u.value...)
		hasMembers = true
	}

	if len(inserts) > 0 {
		out := append([]byte(nil), b...)
		out = append(out[:insertPos], append(inserts, out[insertPos:]...)...)
		b = out
	}

	return b, nil
}
