package statement

import (
	"bytes"
	"compress/zlib"
	"io"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	pdfStreamRe   = regexp.MustCompile(`(?s)stream\r?\n(.*?)\r?\nendstream`)
	pdfTjRe       = regexp.MustCompile(`\(((?:\\.|[^\\)])*)\)\s*Tj`)
	pdfTJRe       = regexp.MustCompile(`\[(.*?)\]\s*TJ`)
	pdfTJLitRe    = regexp.MustCompile(`\(((?:\\.|[^\\)])*)\)`)
	pdfObjHeadRe  = regexp.MustCompile(`(\d+)\s+0\s+obj`)
	pdfFontNameRe = regexp.MustCompile(`/([A-Za-z0-9]+)\s+(\d+)\s+0\s+R`)
	pdfBfRangeRe  = regexp.MustCompile(`(?is)beginbfrange(.*?)endbfrange`)
	pdfBfCharRe   = regexp.MustCompile(`(?is)beginbfchar(.*?)endbfchar`)
	pdfHexTokenRe = regexp.MustCompile(`<([0-9A-Fa-f]+)>`)
)

type pdfObject struct {
	num    int
	dict   string
	stream []byte
}

type pdfFont struct {
	identity bool
	cmap     map[uint16]rune
}

type pdfSpan struct {
	x, y float64
	text string
}

func extractPDFText(data []byte) (string, error) {
	trimmed := bytes.TrimSpace(data)
	if !bytes.HasPrefix(trimmed, []byte("%PDF")) {
		return "", ErrNoText
	}
	if bytes.Contains(data, []byte("/Encrypt")) {
		return "", ErrEncrypted
	}

	objects := parsePDFObjects(data)
	if text := extractPDFLayout(objects); strings.TrimSpace(text) != "" {
		return text, nil
	}

	var b strings.Builder
	matches := pdfStreamRe.FindAllSubmatch(data, -1)
	if len(matches) == 0 {
		writePDFOperators(&b, data)
	} else {
		for _, match := range matches {
			writePDFOperators(&b, decodePDFStream(match[1]))
		}
	}
	text := strings.TrimSpace(b.String())
	if text == "" {
		return "", ErrNoText
	}
	return text, nil
}

func parsePDFObjects(data []byte) map[int]pdfObject {
	out := make(map[int]pdfObject)
	for _, loc := range pdfObjHeadRe.FindAllSubmatchIndex(data, -1) {
		num, _ := strconv.Atoi(string(data[loc[2]:loc[3]]))
		i := loc[1]
		for i < len(data) && data[i] != '<' && !bytes.HasPrefix(data[i:], []byte("endobj")) {
			i++
		}
		if i+1 >= len(data) || data[i] != '<' {
			continue
		}
		dict, next, ok := readPDFDict(data, i)
		if !ok {
			continue
		}
		obj := pdfObject{num: num, dict: string(dict)}
		rest := bytes.TrimSpace(data[next:])
		if bytes.HasPrefix(rest, []byte("stream")) {
			payload, _ := readPDFStreamPayload(data, next)
			obj.stream = payload
		}
		out[num] = obj
	}
	return out
}

func readPDFDict(data []byte, start int) ([]byte, int, bool) {
	if start+1 >= len(data) || data[start] != '<' || data[start+1] != '<' {
		return nil, start, false
	}
	depth := 0
	inString := false
	escape := false
	inHex := false
	for i := start; i < len(data); i++ {
		c := data[i]
		if inString {
			if escape {
				escape = false
				continue
			}
			if c == '\\' {
				escape = true
				continue
			}
			if c == ')' {
				inString = false
			}
			continue
		}
		if inHex {
			if c == '>' {
				inHex = false
			}
			continue
		}
		if c == '(' {
			inString = true
			continue
		}
		if c == '<' {
			if i+1 < len(data) && data[i+1] == '<' {
				depth++
				i++
				continue
			}
			inHex = true
			continue
		}
		if c == '>' {
			if i+1 < len(data) && data[i+1] == '>' {
				depth--
				i++
				if depth == 0 {
					return data[start : i+1], i + 1, true
				}
			}
		}
	}
	return nil, start, false
}

func readPDFStreamPayload(data []byte, dictEnd int) ([]byte, bool) {
	i := dictEnd
	for i < len(data) && (data[i] == ' ' || data[i] == '\n' || data[i] == '\r' || data[i] == '\t') {
		i++
	}
	if !bytes.HasPrefix(data[i:], []byte("stream")) {
		return nil, false
	}
	i += len("stream")
	if i < len(data) && data[i] == '\r' {
		i++
	}
	if i < len(data) && data[i] == '\n' {
		i++
	}
	end := bytes.Index(data[i:], []byte("endstream"))
	if end < 0 {
		return nil, false
	}
	payload := data[i : i+end]
	payload = bytes.TrimRight(payload, "\r\n")
	return payload, true
}

func (o pdfObject) decodedStream() []byte {
	if len(o.stream) == 0 {
		return nil
	}
	if strings.Contains(o.dict, "/FlateDecode") {
		return decodePDFStream(o.stream)
	}
	return o.stream
}

func extractPDFLayout(objects map[int]pdfObject) string {
	fontsByObj := loadPDFFonts(objects)
	if len(fontsByObj) == 0 && len(objects) == 0 {
		return ""
	}
	var pages []pdfObject
	for _, obj := range objects {
		if isPDFPageDict(obj.dict) {
			pages = append(pages, obj)
		}
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].num < pages[j].num })
	if len(pages) == 0 {
		return ""
	}

	var lines []string
	for _, page := range pages {
		fonts := fontsForResources(objects, fontsByObj, page.dict)
		content := contentForPage(objects, page.dict)
		if len(content) == 0 {
			continue
		}
		spans := walkPDFContent(content, fonts)
		lines = append(lines, layoutPDFSpans(spans)...)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func loadPDFFonts(objects map[int]pdfObject) map[int]pdfFont {
	out := make(map[int]pdfFont)
	for num, obj := range objects {
		if !strings.Contains(obj.dict, "/Type /Font") && !strings.Contains(obj.dict, "/Type/Font") {
			continue
		}
		font := pdfFont{cmap: map[uint16]rune{}}
		if strings.Contains(obj.dict, "/Identity-H") || strings.Contains(obj.dict, "/Identity") {
			font.identity = true
		}
		if ref, ok := dictRef(obj.dict, "ToUnicode"); ok {
			if uni, exists := objects[ref]; exists {
				font.cmap = parseCMap(uni.decodedStream())
			}
		}
		out[num] = font
	}
	return out
}

func fontsForResources(objects map[int]pdfObject, fontsByObj map[int]pdfFont, dict string) map[string]pdfFont {
	out := make(map[string]pdfFont)
	if ref, ok := dictRef(dict, "Resources"); ok {
		if obj, exists := objects[ref]; exists {
			dict = obj.dict
		}
	} else if inner, ok := dictEntryDict(dict, "Resources"); ok {
		dict = inner
	}
	fontDict := ""
	if ref, ok := dictRef(dict, "Font"); ok {
		if obj, exists := objects[ref]; exists {
			fontDict = obj.dict
		}
	} else if inner, ok := dictEntryDict(dict, "Font"); ok {
		fontDict = inner
	}
	if fontDict == "" {
		return out
	}
	for _, match := range pdfFontNameRe.FindAllStringSubmatch(fontDict, -1) {
		objNum, _ := strconv.Atoi(match[2])
		if font, ok := fontsByObj[objNum]; ok {
			out[match[1]] = font
		} else {
			out[match[1]] = pdfFont{}
		}
	}
	return out
}

func contentForPage(objects map[int]pdfObject, dict string) []byte {
	if ref, ok := dictRef(dict, "Contents"); ok {
		if obj, exists := objects[ref]; exists {
			return obj.decodedStream()
		}
	}
	if refs := dictRefArray(dict, "Contents"); len(refs) > 0 {
		var b bytes.Buffer
		for _, ref := range refs {
			if obj, exists := objects[ref]; exists {
				b.Write(obj.decodedStream())
				b.WriteByte('\n')
			}
		}
		return b.Bytes()
	}
	return nil
}

func isPDFPageDict(dict string) bool {
	hasPages := strings.Contains(dict, "/Type /Pages") || strings.Contains(dict, "/Type/Pages")
	hasPage := strings.Contains(dict, "/Type /Page") || strings.Contains(dict, "/Type/Page")
	return hasPage && !hasPages
}

func dictRef(dict, key string) (int, bool) {
	re := regexp.MustCompile(`/` + regexp.QuoteMeta(key) + `\s+(\d+)\s+0\s+R`)
	match := re.FindStringSubmatch(dict)
	if match == nil {
		return 0, false
	}
	n, _ := strconv.Atoi(match[1])
	return n, true
}

func dictRefArray(dict, key string) []int {
	re := regexp.MustCompile(`/` + regexp.QuoteMeta(key) + `\s*\[(.*?)\]`)
	match := re.FindStringSubmatch(dict)
	if match == nil {
		return nil
	}
	var out []int
	for _, part := range regexp.MustCompile(`(\d+)\s+0\s+R`).FindAllStringSubmatch(match[1], -1) {
		n, _ := strconv.Atoi(part[1])
		out = append(out, n)
	}
	return out
}

func dictEntryDict(dict, key string) (string, bool) {
	re := regexp.MustCompile(`/` + regexp.QuoteMeta(key) + `\s+<<`)
	loc := re.FindStringIndex(dict)
	if loc == nil {
		return "", false
	}
	start := loc[1] - 2
	raw, _, ok := readPDFDict([]byte(dict), start)
	if !ok {
		return "", false
	}
	return string(raw), true
}

func parseCMap(data []byte) map[uint16]rune {
	out := make(map[uint16]rune)
	text := string(data)
	for _, block := range pdfBfRangeRe.FindAllStringSubmatch(text, -1) {
		hexes := pdfHexTokenRe.FindAllStringSubmatch(block[1], -1)
		for i := 0; i+2 < len(hexes); i += 3 {
			start := parseHex16(hexes[i][1])
			end := parseHex16(hexes[i+1][1])
			dst := parseHexRune(hexes[i+2][1])
			if end < start {
				continue
			}
			for cid := start; cid <= end; cid++ {
				out[cid] = dst + rune(cid-start)
			}
		}
	}
	for _, block := range pdfBfCharRe.FindAllStringSubmatch(text, -1) {
		hexes := pdfHexTokenRe.FindAllStringSubmatch(block[1], -1)
		for i := 0; i+1 < len(hexes); i += 2 {
			out[parseHex16(hexes[i][1])] = parseHexRune(hexes[i+1][1])
		}
	}
	return out
}

func parseHex16(raw string) uint16 {
	v, _ := strconv.ParseUint(raw, 16, 16)
	return uint16(v)
}

func parseHexRune(raw string) rune {
	if len(raw) > 8 {
		raw = raw[len(raw)-8:]
	}
	v, _ := strconv.ParseUint(raw, 16, 32)
	return rune(v)
}

func walkPDFContent(content []byte, fonts map[string]pdfFont) []pdfSpan {
	s := string(content)
	i := 0
	var (
		stack []pdfToken
		x, y  float64
		font  pdfFont
		spans []pdfSpan
	)
	flush := func(raw []byte) {
		text := decodePDFFontBytes(raw, font)
		text = strings.ReplaceAll(text, "\u0000", "")
		if strings.TrimSpace(text) == "" {
			return
		}
		spans = append(spans, pdfSpan{x: x, y: y, text: text})
	}

	for i < len(s) {
		tok, next, ok := readPDFToken(s, i)
		if !ok {
			break
		}
		i = next
		if tok.kind != "op" {
			stack = append(stack, tok)
			continue
		}
		switch tok.raw {
		case "BT":
			x, y = 0, 0
			stack = nil
		case "ET":
			stack = nil
		case "Tf":
			if name, ok := lastName(stack); ok {
				if found, exists := fonts[name]; exists {
					font = found
				} else if strings.HasPrefix(name, "/") {
					font = fonts[strings.TrimPrefix(name, "/")]
				} else {
					font = fonts[name]
				}
			}
			stack = nil
		case "Tm":
			if len(stack) >= 6 {
				x = stack[len(stack)-2].num
				y = stack[len(stack)-1].num
			}
			stack = nil
		case "Td", "TD":
			if len(stack) >= 2 {
				x += stack[len(stack)-2].num
				y += stack[len(stack)-1].num
			}
			stack = nil
		case "T*":
			y -= 12
			stack = nil
		case "Tj":
			if len(stack) >= 1 && (stack[len(stack)-1].kind == "str" || stack[len(stack)-1].kind == "hex") {
				flush(stack[len(stack)-1].bytes)
			}
			stack = nil
		case "'":
			y -= 12
			if len(stack) >= 1 && (stack[len(stack)-1].kind == "str" || stack[len(stack)-1].kind == "hex") {
				flush(stack[len(stack)-1].bytes)
			}
			stack = nil
		case "TJ":
			if len(stack) >= 1 && stack[len(stack)-1].kind == "array" {
				var b []byte
				for _, item := range stack[len(stack)-1].array {
					if item.kind == "str" || item.kind == "hex" {
						b = append(b, item.bytes...)
					}
				}
				flush(b)
			}
			stack = nil
		default:
			stack = nil
		}
	}
	return spans
}

type pdfToken struct {
	kind  string
	raw   string
	num   float64
	bytes []byte
	array []pdfToken
}

func readPDFToken(s string, i int) (pdfToken, int, bool) {
	for i < len(s) {
		c := s[i]
		if c == ' ' || c == '\n' || c == '\r' || c == '\t' || c == '\x00' {
			i++
			continue
		}
		if c == '%' {
			for i < len(s) && s[i] != '\n' && s[i] != '\r' {
				i++
			}
			continue
		}
		break
	}
	if i >= len(s) {
		return pdfToken{}, i, false
	}
	switch s[i] {
	case '[':
		arr, next := readPDFArray(s, i+1)
		return pdfToken{kind: "array", array: arr}, next, true
	case ']':
		return pdfToken{kind: "op", raw: "]"}, i + 1, true
	case '(':
		raw, next := readPDFLiteral(s, i+1)
		return pdfToken{kind: "str", bytes: unescapePDFLiteral(raw)}, next, true
	case '<':
		if i+1 < len(s) && s[i+1] == '<' {
			_, next, ok := readPDFDict([]byte(s), i)
			if !ok {
				return pdfToken{kind: "op", raw: "<<"}, i + 2, true
			}
			return pdfToken{kind: "dict"}, next, true
		}
		end := strings.IndexByte(s[i:], '>')
		if end < 0 {
			return pdfToken{}, len(s), false
		}
		hex := strings.ReplaceAll(s[i+1:i+end], " ", "")
		hex = strings.ReplaceAll(hex, "\n", "")
		hex = strings.ReplaceAll(hex, "\r", "")
		return pdfToken{kind: "hex", bytes: parsePDFHex(hex)}, i + end + 1, true
	case '/':
		j := i + 1
		for j < len(s) && isPDFNameChar(s[j]) {
			j++
		}
		return pdfToken{kind: "name", raw: s[i+1 : j]}, j, true
	}
	if s[i] == '+' || s[i] == '-' || s[i] == '.' || (s[i] >= '0' && s[i] <= '9') {
		j := i
		if s[j] == '+' || s[j] == '-' {
			j++
		}
		for j < len(s) && ((s[j] >= '0' && s[j] <= '9') || s[j] == '.') {
			j++
		}
		if j > i && (j < len(s) && !isPDFOpChar(s[j]) || j == len(s) || s[j] == ' ' || s[j] == '\n' || s[j] == '\r' || s[j] == '\t' || s[j] == '[' || s[j] == ']' || s[j] == '/' || s[j] == '(' || s[j] == '<' || s[j] == '%') {
			n, err := strconv.ParseFloat(s[i:j], 64)
			if err == nil {
				return pdfToken{kind: "num", num: n, raw: s[i:j]}, j, true
			}
		}
	}
	j := i
	for j < len(s) && isPDFOpChar(s[j]) {
		j++
	}
	if j == i {
		return pdfToken{kind: "op", raw: s[i : i+1]}, i + 1, true
	}
	return pdfToken{kind: "op", raw: s[i:j]}, j, true
}

func readPDFArray(s string, i int) ([]pdfToken, int) {
	var out []pdfToken
	for i < len(s) {
		if s[i] == ']' {
			return out, i + 1
		}
		tok, next, ok := readPDFToken(s, i)
		if !ok {
			return out, i
		}
		if tok.kind == "op" && tok.raw == "]" {
			return out, next
		}
		out = append(out, tok)
		i = next
	}
	return out, i
}

func readPDFLiteral(s string, i int) (string, int) {
	depth := 1
	start := i
	escape := false
	for i < len(s) {
		c := s[i]
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
		if c == '(' {
			depth++
		} else if c == ')' {
			depth--
			if depth == 0 {
				return s[start:i], i + 1
			}
		}
		i++
	}
	return s[start:], len(s)
}

func unescapePDFLiteral(s string) []byte {
	var out []byte
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			out = append(out, s[i])
			continue
		}
		if i+1 >= len(s) {
			break
		}
		i++
		switch s[i] {
		case 'n':
			out = append(out, '\n')
		case 'r':
			out = append(out, '\r')
		case 't':
			out = append(out, '\t')
		case 'b':
			out = append(out, '\b')
		case 'f':
			out = append(out, '\f')
		case '(', ')', '\\':
			out = append(out, s[i])
		case '\n':
		case '\r':
			if i+1 < len(s) && s[i+1] == '\n' {
				i++
			}
		default:
			if s[i] >= '0' && s[i] <= '7' {
				val := int(s[i] - '0')
				n := 1
				for n < 3 && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '7' {
					i++
					n++
					val = val*8 + int(s[i]-'0')
				}
				out = append(out, byte(val))
			} else {
				out = append(out, s[i])
			}
		}
	}
	return out
}

func parsePDFHex(raw string) []byte {
	if len(raw)%2 == 1 {
		raw += "0"
	}
	out := make([]byte, 0, len(raw)/2)
	for i := 0; i+1 < len(raw); i += 2 {
		v, err := strconv.ParseUint(raw[i:i+2], 16, 8)
		if err != nil {
			continue
		}
		out = append(out, byte(v))
	}
	return out
}

func decodePDFFontBytes(raw []byte, font pdfFont) string {
	if font.identity || len(font.cmap) > 0 {
		return decodeIdentityCIDs(raw, font.cmap)
	}
	s := string(raw)
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "")
	}
	return s
}

func decodeIdentityCIDs(data []byte, cmap map[uint16]rune) string {
	if len(data) == 0 {
		return ""
	}
	if len(data)%2 == 1 {
		data = append([]byte{0}, data...)
	}
	var b strings.Builder
	b.Grow(len(data) / 2)
	for i := 0; i+1 < len(data); i += 2 {
		cid := uint16(data[i])<<8 | uint16(data[i+1])
		if r, ok := cmap[cid]; ok {
			b.WriteRune(r)
			continue
		}
		if cid >= 32 && cid < 127 {
			b.WriteRune(rune(cid))
		}
	}
	return b.String()
}

func layoutPDFSpans(spans []pdfSpan) []string {
	if len(spans) == 0 {
		return nil
	}
	const yTol = 2.8
	sort.SliceStable(spans, func(i, j int) bool {
		if math.Abs(spans[i].y-spans[j].y) > yTol {
			return spans[i].y > spans[j].y
		}
		return spans[i].x < spans[j].x
	})
	var (
		lines []string
		cur   []string
		curY  float64
	)
	for _, span := range spans {
		text := strings.TrimSpace(span.text)
		if text == "" {
			continue
		}
		if len(cur) == 0 {
			cur = []string{text}
			curY = span.y
			continue
		}
		if math.Abs(span.y-curY) > yTol {
			lines = append(lines, strings.Join(cur, " "))
			cur = []string{text}
			curY = span.y
			continue
		}
		cur = append(cur, text)
	}
	if len(cur) > 0 {
		lines = append(lines, strings.Join(cur, " "))
	}
	return lines
}

func lastName(stack []pdfToken) (string, bool) {
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i].kind == "name" {
			return stack[i].raw, true
		}
	}
	return "", false
}

func isPDFNameChar(c byte) bool {
	return c > ' ' && c != '/' && c != '(' && c != ')' && c != '<' && c != '>' && c != '[' && c != ']' && c != '{' && c != '}' && c != '%'
}

func isPDFOpChar(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '*' || c == '\'' || c == '"'
}

func decodePDFStream(payload []byte) []byte {
	reader, err := zlib.NewReader(bytes.NewReader(payload))
	if err != nil {
		return payload
	}
	defer reader.Close()
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return payload
	}
	return decoded
}

func writePDFOperators(b *strings.Builder, content []byte) {
	for _, match := range pdfTjRe.FindAllSubmatch(content, -1) {
		writePDFString(b, match[1])
		b.WriteByte('\n')
	}
	for _, match := range pdfTJRe.FindAllSubmatch(content, -1) {
		for _, lit := range pdfTJLitRe.FindAllSubmatch(match[1], -1) {
			writePDFString(b, lit[1])
		}
		b.WriteByte('\n')
	}
}

func writePDFString(b *strings.Builder, raw []byte) {
	s := string(unescapePDFLiteral(string(raw)))
	s = strings.ReplaceAll(s, "\u0000", "")
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "")
	}
	b.WriteString(s)
	if !strings.HasSuffix(s, " ") {
		b.WriteByte(' ')
	}
}
