package config

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/unstable"
)

type bytePatch struct {
	start int
	end   int
	repl  []byte
}

func applyBytePatches(src []byte, patches []bytePatch) []byte {
	if len(patches) == 0 {
		return src
	}
	sort.Slice(patches, func(i, j int) bool {
		if patches[i].start == patches[j].start {
			return patches[i].end > patches[j].end
		}
		return patches[i].start > patches[j].start
	})

	out := append([]byte(nil), src...)
	for _, p := range patches {
		if p.start < 0 || p.start > len(out) || p.end < p.start || p.end > len(out) {
			continue
		}
		out = append(out[:p.start], append(p.repl, out[p.end:]...)...)
	}
	return out
}

func tomlEncodeValue(v interface{}) ([]byte, error) {
	encoded, err := toml.Marshal(map[string]interface{}{"_": v})
	if err != nil {
		return nil, err
	}
	eq := bytes.IndexByte(encoded, '=')
	if eq < 0 {
		return nil, fmt.Errorf("unexpected toml encoding")
	}
	return bytes.TrimSpace(encoded[eq+1:]), nil
}

func detectNewline(data []byte) string {
	if bytes.Contains(data, []byte("\r\n")) {
		return "\r\n"
	}
	return "\n"
}

func parseKeyParts(it unstable.Iterator) []string {
	var out []string
	for it.Next() {
		n := it.Node()
		if n == nil || !n.Valid() {
			continue
		}
		out = append(out, string(n.Data))
	}
	return out
}

func dot(parts []string) string {
	return strings.Join(parts, ".")
}

func findLineStart(data []byte, offset int) int {
	if offset < 0 {
		return 0
	}
	if offset > len(data) {
		offset = len(data)
	}
	for offset > 0 && data[offset-1] != '\n' {
		offset--
	}
	return offset
}

func findLineEnd(data []byte, offset int) int {
	if offset < 0 {
		offset = 0
	}
	if offset > len(data) {
		offset = len(data)
	}
	for offset < len(data) && data[offset] != '\n' {
		offset++
	}
	return offset
}

func findValueSpanForKeyAt(data []byte, keyOffset int) (valueStart, valueEnd int, ok bool) {
	exprStart := findLineStart(data, keyOffset)
	exprEnd := findLineEnd(data, keyOffset)
	line := data[exprStart:exprEnd]

	eq := bytes.IndexByte(line, '=')
	if eq < 0 {
		return 0, 0, false
	}

	valueStart = exprStart + eq + 1
	for valueStart < exprEnd {
		b := data[valueStart]
		if b != ' ' && b != '\t' {
			break
		}
		valueStart++
	}
	if valueStart >= exprEnd {
		return 0, 0, false
	}

	switch data[valueStart] {
	case '"':
		// basic string (no multiline in our managed keys)
		i := valueStart + 1
		for i < exprEnd {
			switch data[i] {
			case '\\':
				i += 2
				continue
			case '"':
				i++
				return valueStart, i, true
			default:
				i++
			}
		}
		return 0, 0, false
	case '\'':
		// literal string
		i := valueStart + 1
		for i < exprEnd {
			if data[i] == '\'' {
				i++
				return valueStart, i, true
			}
			i++
		}
		return 0, 0, false
	default:
		i := valueStart
		for i < exprEnd {
			if data[i] == '#' {
				break
			}
			i++
		}
		valueEnd = i
		for valueEnd > valueStart {
			b := data[valueEnd-1]
			if b != ' ' && b != '\t' && b != '\r' {
				break
			}
			valueEnd--
		}
		if valueEnd <= valueStart {
			return 0, 0, false
		}
		return valueStart, valueEnd, true
	}
}

func findBeforeFirstTableOffset(data []byte) int {
	i := 0
	for i < len(data) {
		lineStart := i
		lineEnd := bytes.IndexByte(data[i:], '\n')
		if lineEnd < 0 {
			lineEnd = len(data) - i
		}
		i += lineEnd
		line := data[lineStart:i]
		if bytes.HasSuffix(line, []byte("\r")) {
			line = bytes.TrimSuffix(line, []byte("\r"))
		}

		trim := strings.TrimSpace(string(line))
		if strings.HasPrefix(trim, "[") {
			return lineStart
		}

		if i < len(data) && data[i] == '\n' {
			i++
		}
	}
	return len(data)
}

func findTrailingWhitespaceStart(data []byte) int {
	i := len(data)
	for i > 0 {
		b := data[i-1]
		if b != ' ' && b != '\t' && b != '\r' && b != '\n' {
			break
		}
		i--
	}
	return i
}

func findTableInsertOffset(data []byte, tableStart, tableEnd int) int {
	if tableStart < 0 {
		tableStart = 0
	}
	if tableEnd > len(data) {
		tableEnd = len(data)
	}
	if tableStart > tableEnd {
		return tableEnd
	}

	seg := data[tableStart:tableEnd]
	trimEnd := len(seg)
	for trimEnd > 0 {
		b := seg[trimEnd-1]
		if b != ' ' && b != '\t' && b != '\r' && b != '\n' {
			break
		}
		trimEnd--
	}

	if trimEnd == 0 {
		return tableEnd
	}

	nl := bytes.IndexByte(seg[trimEnd:], '\n')
	if nl < 0 {
		return tableEnd
	}
	return tableStart + trimEnd + nl + 1
}

// buildCodexManagedKeyValues extracts the managed key/value pairs from the CCS config TOML.
// Keys are represented as dot paths (e.g. "model_provider", "model_providers.custom.base_url").
func buildCodexManagedKeyValues(ccsConfig map[string]interface{}) map[string]interface{} {
	managed := make(map[string]interface{})

	for _, k := range []string{
		"model_provider",
		"model",
		"model_reasoning_effort",
		"disable_response_storage",
	} {
		if v, ok := ccsConfig[k]; ok {
			managed[k] = v
		}
	}

	if providers, ok := ccsConfig["model_providers"].(map[string]interface{}); ok {
		for name, raw := range providers {
			cfg, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			for k, v := range cfg {
				if k == "env_key" {
					continue
				}
				managed["model_providers."+name+"."+k] = v
			}
		}
	}

	return managed
}

func stableProviderKeys(m map[string]interface{}) []string {
	if m == nil {
		return nil
	}
	preferred := []string{"name", "base_url", "wire_api", "requires_openai_auth"}
	seen := make(map[string]bool, len(m))
	var out []string
	for _, k := range preferred {
		if _, ok := m[k]; ok {
			out = append(out, k)
			seen[k] = true
		}
	}
	var rest []string
	for k := range m {
		if seen[k] {
			continue
		}
		rest = append(rest, k)
	}
	sort.Strings(rest)
	out = append(out, rest...)
	return out
}

func buildProviderTableBlock(provider string, kv map[string]interface{}, newline string) ([]byte, error) {
	var b strings.Builder
	b.WriteString("[model_providers.")
	b.WriteString(provider)
	b.WriteString("]")
	b.WriteString(newline)

	keys := stableProviderKeys(kv)
	for _, k := range keys {
		valBytes, err := tomlEncodeValue(kv[k])
		if err != nil {
			return nil, err
		}
		b.WriteString(k)
		b.WriteString(" = ")
		b.Write(valBytes)
		b.WriteString(newline)
	}

	return []byte(b.String()), nil
}

// patchCodexTOMLPreserveLayout updates only managed key/value ranges in an existing TOML document,
// and inserts missing managed entries without re-serializing the whole file.
func patchCodexTOMLPreserveLayout(existing []byte, ccsConfig map[string]interface{}) ([]byte, error) {
	managed := buildCodexManagedKeyValues(ccsConfig)
	if len(managed) == 0 {
		return existing, nil
	}

	newline := detectNewline(existing)

	type kvSpan struct {
		valueStart int
		valueEnd   int
	}
	kvSpans := make(map[string]kvSpan)

	type tableHeader struct {
		path  string
		start int
	}
	var tableHeaders []tableHeader

	var parser unstable.Parser
	parser.KeepComments = true
	parser.Reset(existing)

	var currentTable []string
	for parser.NextExpression() {
		expr := parser.Expression()
		if expr == nil || !expr.Valid() {
			continue
		}

		switch expr.Kind {
		case unstable.Table, unstable.ArrayTable:
			currentTable = parseKeyParts(expr.Key())
			keyIt := expr.Key()
			if !keyIt.Next() {
				continue
			}
			keyStart := int(keyIt.Node().Raw.Offset)
			tableHeaders = append(tableHeaders, tableHeader{path: dot(currentTable), start: findLineStart(existing, keyStart)})
		case unstable.KeyValue:
			keyParts := parseKeyParts(expr.Key())
			full := append(append([]string(nil), currentTable...), keyParts...)
			key := dot(full)
			if _, ok := managed[key]; !ok {
				continue
			}
			keyIt := expr.Key()
			if !keyIt.Next() {
				continue
			}
			keyStart := int(keyIt.Node().Raw.Offset)
			vs, ve, ok := findValueSpanForKeyAt(existing, keyStart)
			if !ok {
				continue
			}
			kvSpans[key] = kvSpan{valueStart: vs, valueEnd: ve}
		}
	}
	if err := parser.Error(); err != nil {
		return nil, err
	}

	// Table ranges: [start, end)
	tableRanges := make(map[string][2]int)
	for i, h := range tableHeaders {
		end := len(existing)
		if i+1 < len(tableHeaders) {
			end = tableHeaders[i+1].start
		}
		tableRanges[h.path] = [2]int{h.start, end}
	}

	var patches []bytePatch
	patchedKeys := make(map[string]bool, len(managed))

	// 1) Patch existing managed values in place.
	for key, r := range kvSpans {
		valBytes, err := tomlEncodeValue(managed[key])
		if err != nil {
			return nil, err
		}
		patches = append(patches, bytePatch{start: r.valueStart, end: r.valueEnd, repl: valBytes})
		patchedKeys[key] = true
	}

	// 2) Insert missing top-level managed keys (must be before any table header).
	topInsertOffset := findBeforeFirstTableOffset(existing)
	if topInsertOffset == len(existing) {
		topInsertOffset = findTrailingWhitespaceStart(existing)
	}
	var topMissing []string
	for _, k := range []string{"model_provider", "model", "model_reasoning_effort", "disable_response_storage"} {
		if _, ok := managed[k]; !ok {
			continue
		}
		if patchedKeys[k] {
			continue
		}
		topMissing = append(topMissing, k)
	}
	if len(topMissing) > 0 {
		var b strings.Builder
		for _, k := range topMissing {
			valBytes, err := tomlEncodeValue(managed[k])
			if err != nil {
				return nil, err
			}
			b.WriteString(k)
			b.WriteString(" = ")
			b.Write(valBytes)
			b.WriteString(newline)
		}
		patches = append(patches, bytePatch{start: topInsertOffset, end: topInsertOffset, repl: []byte(b.String())})
		for _, k := range topMissing {
			patchedKeys[k] = true
		}
	}

	// 3) Ensure managed provider tables exist and contain managed keys.
	// Group managed provider values by table (model_providers.<name>).
	providerTables := make(map[string]map[string]interface{})
	for k, v := range managed {
		if !strings.HasPrefix(k, "model_providers.") {
			continue
		}
		rest := strings.TrimPrefix(k, "model_providers.")
		parts := strings.SplitN(rest, ".", 2)
		if len(parts) != 2 {
			continue
		}
		name := parts[0]
		field := parts[1]
		if providerTables[name] == nil {
			providerTables[name] = make(map[string]interface{})
		}
		providerTables[name][field] = v
	}

	trailing := findTrailingWhitespaceStart(existing)
	for name, kv := range providerTables {
		tablePath := "model_providers." + name
		if rng, ok := tableRanges[tablePath]; ok {
			// Insert missing keys into existing table without touching its other bytes.
			insertAt := findTableInsertOffset(existing, rng[0], rng[1])
			var missingKeys []string
			for _, field := range stableProviderKeys(kv) {
				fullKey := "model_providers." + name + "." + field
				if patchedKeys[fullKey] {
					continue
				}
				// If the field doesn't exist, add it.
				if _, ok := kvSpans[fullKey]; ok {
					continue
				}
				missingKeys = append(missingKeys, field)
			}
			if len(missingKeys) == 0 {
				continue
			}

			var b strings.Builder
			for _, field := range missingKeys {
				valBytes, err := tomlEncodeValue(kv[field])
				if err != nil {
					return nil, err
				}
				b.WriteString(field)
				b.WriteString(" = ")
				b.Write(valBytes)
				b.WriteString(newline)
				patchedKeys["model_providers."+name+"."+field] = true
			}
			patches = append(patches, bytePatch{start: insertAt, end: insertAt, repl: []byte(b.String())})
			continue
		}

		// Missing table: append a full managed table block near EOF (before trailing whitespace).
		block, err := buildProviderTableBlock(name, kv, newline)
		if err != nil {
			return nil, err
		}
		prefix := []byte{}
		if trailing > 0 && !bytes.HasSuffix(existing[:trailing], []byte("\n")) {
			prefix = []byte(newline)
		}
		prefix = append(prefix, block...)
		patches = append(patches, bytePatch{start: trailing, end: trailing, repl: prefix})
		for field := range kv {
			patchedKeys["model_providers."+name+"."+field] = true
		}
	}

	return applyBytePatches(existing, patches), nil
}
