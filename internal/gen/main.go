// Command gen regenerates upstream.go from Braintree's
// credit-card-type (https://github.com/braintree/credit-card-type, MIT).
//
// Usage:
//
//	go generate ./...
//	go run ./internal/gen -version 10.4.0
//
// The generated file is the package's whole BIN table, and it holds upstream's
// data exactly: no brand, range or rule is added, dropped or adjusted here.
// Projects whose BIN data differs layer their own ranges on top with NewTable.
//
// Three upstream files are read. card-types.ts carries the BIN patterns,
// lengths and security codes. index.ts carries ORIGINAL_TEST_ORDER, which
// decides which brand wins when two patterns are equally specific -- Verve
// before Maestro, for instance -- so the generated slice must preserve it.
// LICENSE is upstream's MIT license: it is copied to LICENSE-credit-card-type
// and its copyright line goes into the generated file's header, so the notice
// MIT requires always matches the data it covers.
package main

import (
	"flag"
	"fmt"
	"go/format"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// httpClient bounds each sync request so a hung connection cannot stall
// go generate indefinitely.
var httpClient = &http.Client{Timeout: 30 * time.Second}

const rawURL = "https://raw.githubusercontent.com/braintree/credit-card-type/v%s/%s"

// defaultVersion is the release "go generate" syncs from. Keep it equal to the
// UpstreamVersion of the committed upstream.go -- a test checks -- or a plain
// "go generate" would quietly rebuild an older table.
const defaultVersion = "10.3.1"

func main() {
	version := flag.String("version", defaultVersion, "credit-card-type release to sync from")
	out := flag.String("o", "upstream.go", "file to write")
	licenseOut := flag.String("license", "LICENSE-credit-card-type", "file to copy upstream's license to")
	flag.Parse()

	types, err := fetch(*version, "src/lib/card-types.ts")
	if err != nil {
		log.Fatalf("fetch card-types.ts: %v", err)
	}

	index, err := fetch(*version, "src/index.ts")
	if err != nil {
		log.Fatalf("fetch index.ts: %v", err)
	}

	license, err := fetch(*version, "LICENSE")
	if err != nil {
		log.Fatalf("fetch LICENSE: %v", err)
	}

	copyrights, err := parseCopyright(license)
	if err != nil {
		log.Fatalf("parse LICENSE: %v", err)
	}

	cards, err := parseCards(types)
	if err != nil {
		log.Fatalf("parse card-types.ts: %v", err)
	}

	order, err := parseTestOrder(index)
	if err != nil {
		log.Fatalf("parse index.ts: %v", err)
	}

	ordered, err := sortByTestOrder(cards, order)
	if err != nil {
		log.Fatal(err)
	}

	src, err := render(*version, copyrights, *licenseOut, ordered)
	if err != nil {
		log.Fatalf("render: %v", err)
	}

	if err := os.WriteFile(*out, src, 0o644); err != nil {
		log.Fatalf("write %s: %v", *out, err)
	}

	if err := os.WriteFile(*licenseOut, []byte(license), 0o644); err != nil {
		log.Fatalf("write %s: %v", *licenseOut, err)
	}

	fmt.Printf("wrote %s: %d brands from credit-card-type v%s\n", *out, len(ordered), *version)
}

func fetch(version, path string) (string, error) {
	url := fmt.Sprintf(rawURL, version, path)

	resp, err := httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: %s", url, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

type pattern struct{ min, max string }

func (p pattern) String() string {
	if p.max == "" {
		return fmt.Sprintf("Prefix(%q)", p.min)
	}

	return fmt.Sprintf("Range(%q, %q)", p.min, p.max)
}

type card struct {
	typ      string
	niceType string
	patterns []pattern
	lengths  []int
	codeName string
	codeSize int
}

var (
	blockRe    = regexp.MustCompile(`(?s)\{(.*?)\}\s*as BuiltInCreditCardType`)
	typeRe     = regexp.MustCompile(`\btype:\s*"([^"]+)"`)
	niceTypeRe = regexp.MustCompile(`\bniceType:\s*"([^"]+)"`)
	lengthsRe  = regexp.MustCompile(`(?s)\blengths:\s*\[(.*?)\]`)
	codeNameRe = regexp.MustCompile(`\bname:\s*"([^"]+)"`)
	codeSizeRe = regexp.MustCompile(`\bsize:\s*(\d+)`)
)

func parseCards(src string) ([]card, error) {
	blocks := blockRe.FindAllStringSubmatch(src, -1)
	if len(blocks) == 0 {
		return nil, fmt.Errorf("no card definitions found; upstream file layout may have changed")
	}

	cards := make([]card, 0, len(blocks))

	for _, b := range blocks {
		body := b[1]

		var (
			c   card
			err error
		)

		if m := typeRe.FindStringSubmatch(body); m != nil {
			c.typ = m[1]
		} else {
			return nil, fmt.Errorf("card block without a type field: %.60s", body)
		}

		if m := niceTypeRe.FindStringSubmatch(body); m != nil {
			c.niceType = m[1]
		} else {
			return nil, fmt.Errorf("%s: no niceType field", c.typ)
		}

		c.patterns, err = extractPatterns(body)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", c.typ, err)
		}

		if m := lengthsRe.FindStringSubmatch(body); m != nil {
			c.lengths, err = parseInts(m[1])
			if err != nil {
				return nil, fmt.Errorf("%s lengths: %w", c.typ, err)
			}
		} else {
			return nil, fmt.Errorf("%s: no lengths field", c.typ)
		}

		if len(c.lengths) == 0 {
			return nil, fmt.Errorf("%s: empty lengths field", c.typ)
		}

		for _, length := range c.lengths {
			if length <= 0 {
				return nil, fmt.Errorf("%s: invalid card-number length %d", c.typ, length)
			}
		}

		if m := codeNameRe.FindStringSubmatch(body); m != nil {
			c.codeName = m[1]
		} else {
			return nil, fmt.Errorf("%s: no security-code name field", c.typ)
		}

		if m := codeSizeRe.FindStringSubmatch(body); m != nil {
			c.codeSize, err = strconv.Atoi(m[1])
			if err != nil || c.codeSize <= 0 {
				return nil, fmt.Errorf("%s: invalid security-code size %q", c.typ, m[1])
			}
		} else {
			return nil, fmt.Errorf("%s: no security-code size field", c.typ)
		}

		cards = append(cards, c)
	}

	return cards, nil
}

// extractPatterns pulls the contents of the `patterns: [...]` array. It counts
// brackets rather than using a regexp because the array nests ranges and spans
// many lines.
func extractPatterns(body string) ([]pattern, error) {
	start := strings.Index(body, "patterns:")
	if start < 0 {
		return nil, fmt.Errorf("no patterns field")
	}

	open := strings.Index(body[start:], "[")
	if open < 0 {
		return nil, fmt.Errorf("patterns field is not an array")
	}
	open += start

	depth, end := 0, -1

	for i := open; i < len(body); i++ {
		switch body[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				end = i
			}
		}

		if end >= 0 {
			break
		}
	}

	if end < 0 {
		return nil, fmt.Errorf("unterminated patterns array")
	}

	return parsePatternList(body[open+1 : end])
}

func parsePatternList(s string) ([]pattern, error) {
	var out []pattern

	for i := 0; i < len(s); {
		switch {
		case isSpace(s[i]) || s[i] == ',':
			i++
		case s[i] == '[':
			end := strings.IndexByte(s[i:], ']')
			if end < 0 {
				return nil, fmt.Errorf("unterminated range")
			}

			bounds, err := parseDigitList(s[i+1 : i+end])
			if err != nil {
				return nil, err
			}

			if len(bounds) != 2 {
				return nil, fmt.Errorf("range with %d bounds, want 2", len(bounds))
			}

			if len(bounds[0]) != len(bounds[1]) {
				return nil, fmt.Errorf("range %s-%s has mismatched widths", bounds[0], bounds[1])
			}

			// Equal-width digit strings order the same as the numbers they
			// spell. A backwards range could never match anything.
			if bounds[0] > bounds[1] {
				return nil, fmt.Errorf("range %s-%s runs backwards", bounds[0], bounds[1])
			}

			out = append(out, pattern{bounds[0], bounds[1]})
			i += end + 1
		default:
			j := i
			for j < len(s) && !isSpace(s[j]) && s[j] != ',' {
				j++
			}

			lit := s[i:j]
			if !isDigits(lit) {
				return nil, fmt.Errorf("non-numeric pattern %q", lit)
			}

			out = append(out, pattern{min: lit})
			i = j
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("empty patterns array")
	}

	return out, nil
}

func parseDigitList(s string) ([]string, error) {
	var out []string

	for _, f := range strings.Split(s, ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}

		if !isDigits(f) {
			return nil, fmt.Errorf("non-numeric value %q", f)
		}

		out = append(out, f)
	}

	return out, nil
}

func parseInts(s string) ([]int, error) {
	fields, err := parseDigitList(s)
	if err != nil {
		return nil, err
	}

	out := make([]int, 0, len(fields))

	for _, f := range fields {
		n, err := strconv.Atoi(f)
		if err != nil {
			return nil, err
		}

		out = append(out, n)
	}

	return out, nil
}

// parseCopyright returns every copyright line of upstream's MIT license, and
// fails if the license is no longer MIT: the package's own license and the
// notice in the generated file both assume it is.
func parseCopyright(license string) ([]string, error) {
	if !strings.Contains(license, "Permission is hereby granted, free of charge") {
		return nil, fmt.Errorf("upstream LICENSE is no longer the MIT license; review it before syncing")
	}

	var lines []string

	for _, line := range strings.Split(license, "\n") {
		if line = strings.TrimSpace(line); strings.HasPrefix(line, "Copyright") {
			lines = append(lines, line)
		}
	}

	if len(lines) == 0 {
		return nil, fmt.Errorf("no copyright line in upstream LICENSE")
	}

	return lines, nil
}

// parseTestOrder resolves ORIGINAL_TEST_ORDER from index.ts through the
// cardNames map, yielding upstream type ids in tie-break order.
func parseTestOrder(src string) ([]string, error) {
	names := map[string]string{}

	nameBlock := regexp.MustCompile(`(?s)const cardNames[^{]*\{(.*?)\n\};`).FindStringSubmatch(src)
	if nameBlock == nil {
		return nil, fmt.Errorf("cardNames map not found")
	}

	for _, m := range regexp.MustCompile(`(\w+):\s*"([^"]+)"`).FindAllStringSubmatch(nameBlock[1], -1) {
		names[m[1]] = m[2]
	}

	orderBlock := regexp.MustCompile(`(?s)const ORIGINAL_TEST_ORDER\s*=\s*\[(.*?)\n\];`).FindStringSubmatch(src)
	if orderBlock == nil {
		return nil, fmt.Errorf("ORIGINAL_TEST_ORDER not found")
	}

	var out []string

	for _, m := range regexp.MustCompile(`cardNames\.(\w+)`).FindAllStringSubmatch(orderBlock[1], -1) {
		typ, ok := names[m[1]]
		if !ok {
			return nil, fmt.Errorf("ORIGINAL_TEST_ORDER references unknown cardNames.%s", m[1])
		}

		out = append(out, typ)
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("ORIGINAL_TEST_ORDER is empty")
	}

	return out, nil
}

func sortByTestOrder(cards []card, order []string) ([]card, error) {
	byType := make(map[string]card, len(cards))
	for _, c := range cards {
		if _, duplicate := byType[c.typ]; duplicate {
			return nil, fmt.Errorf("card-types.ts defines %q more than once", c.typ)
		}

		byType[c.typ] = c
	}

	out := make([]card, 0, len(cards))

	for _, typ := range order {
		c, ok := byType[typ]
		if !ok {
			return nil, fmt.Errorf("ORIGINAL_TEST_ORDER lists %q but card-types.ts does not define it", typ)
		}

		out = append(out, c)
		delete(byType, typ)
	}

	if len(byType) > 0 {
		missing := make([]string, 0, len(byType))
		for typ := range byType {
			missing = append(missing, typ)
		}

		return nil, fmt.Errorf("card-types.ts defines %v but ORIGINAL_TEST_ORDER omits them", missing)
	}

	return out, nil
}

func render(version string, copyrights []string, licenseFile string, cards []card) ([]byte, error) {
	var b strings.Builder

	fmt.Fprintf(&b, `// Code generated by "go run ./internal/gen"; DO NOT EDIT.
//
// Source: https://github.com/braintree/credit-card-type v%s
// %s
// MIT licensed; see %s.
//
// Entry order follows upstream's ORIGINAL_TEST_ORDER, which decides which brand
// wins when two ranges are equally specific.

package creditcard

// UpstreamVersion is the credit-card-type release the package table was
// generated from. Bump it by regenerating, not by editing this file.
const UpstreamVersion = %q

// upstream is the package table, the one Default returns.
var upstream = []entry{
`, version, strings.Join(copyrights, "\n// "), licenseFile, version)

	for _, c := range cards {
		fmt.Fprintf(&b, "\t{\n\t\tBrand: %q,\n\t\tName:  %q,\n", c.typ, c.niceType)

		fmt.Fprintf(&b, "\t\tRanges: []BINRange{\n")
		for _, p := range c.patterns {
			fmt.Fprintf(&b, "\t\t\t%s,\n", p)
		}
		fmt.Fprintf(&b, "\t\t},\n")

		fmt.Fprintf(&b, "\t\tLengths: []int{")
		for i, l := range c.lengths {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "%d", l)
		}
		fmt.Fprintf(&b, "},\n")

		fmt.Fprintf(&b, "\t\tCode:    SecurityCode{Name: %q, Size: %d},\n\t},\n", c.codeName, c.codeSize)
	}

	b.WriteString("}\n")

	return format.Source([]byte(b.String()))
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}

	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}

	return true
}
