package query

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// FetchBGPSummary fetches and parses APNIC thyme's data-summary file, a
// colon-separated key/value listing of BGP routing table analysis metrics.
func FetchBGPSummary(ctx context.Context, c *transport.Client) (*models.BGPSummary, error) {
	url := transport.BuildThymeURL(c.ThymeBaseURL(), c.ThymeSource(), "data-summary")
	body, err := c.FetchText(ctx, url)
	if err != nil {
		return nil, err
	}
	return parseBGPSummary(body), nil
}

// FetchBGPRawTable fetches and parses APNIC thyme's data-raw-table file, which
// lists every BGP route as a "prefix\tASN" line.
func FetchBGPRawTable(ctx context.Context, c *transport.Client) (*models.BGPRawTable, error) {
	url := transport.BuildThymeURL(c.ThymeBaseURL(), c.ThymeSource(), "data-raw-table")
	body, err := c.FetchTextStr(ctx, url)
	if err != nil {
		return nil, err
	}
	t, err := parseBGPRawTable(body)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// FetchBGPASNMap fetches the raw BGP table and aggregates it by origin ASN,
// returning a map from each origin ASN to the prefixes it announces. This is a
// client-side derivation from data-raw-table; thyme does not publish a separate
// per-ASN file.
func FetchBGPASNMap(ctx context.Context, c *transport.Client) (*models.BGPASNMap, error) {
	raw, err := FetchBGPRawTable(ctx, c)
	if err != nil {
		return nil, err
	}
	m := &models.BGPASNMap{ASNs: make(map[string][]string, len(raw.Routes))}
	for _, r := range raw.Routes {
		m.ASNs[r.ASN] = append(m.ASNs[r.ASN], r.Prefix)
	}
	return m, nil
}

// FetchBGPBadPrefixes fetches and parses thyme's data-badpfx-nos file, which
// lists prefixes longer than /24 and their origin AS (potential route leaks).
// source is "current" (default), "au", or "hk"; an empty string uses the
// client's default source.
func FetchBGPBadPrefixes(ctx context.Context, c *transport.Client, source string) (*models.BGPBadPrefixes, error) {
	url := transport.BuildThymeURL(c.ThymeBaseURL(), transport.SourceOrDefault(source, c.ThymeSource()), "data-badpfx-nos")
	body, err := c.FetchText(ctx, url)
	if err != nil {
		return nil, err
	}
	return parseBGPBadPrefixes(body), nil
}

// FetchBGPPerPrefixLength fetches and parses thyme's data-pfx-nos file, which
// counts announced prefixes per prefix length.
func FetchBGPPerPrefixLength(ctx context.Context, c *transport.Client, source string) (*models.BGPPerPrefixLength, error) {
	url := transport.BuildThymeURL(c.ThymeBaseURL(), transport.SourceOrDefault(source, c.ThymeSource()), "data-pfx-nos")
	body, err := c.FetchText(ctx, url)
	if err != nil {
		return nil, err
	}
	return parseBGPPerPrefixLength(body), nil
}

// FetchBGPUsedAutnums fetches and parses thyme's data-used-autnums file, which
// lists every in-use ASN with its registered name and country.
func FetchBGPUsedAutnums(ctx context.Context, c *transport.Client, source string) (*models.BGPUsedAutnums, error) {
	url := transport.BuildThymeURL(c.ThymeBaseURL(), transport.SourceOrDefault(source, c.ThymeSource()), "data-used-autnums")
	body, err := c.FetchTextStr(ctx, url)
	if err != nil {
		return nil, err
	}
	return parseBGPUsedAutnums(body), nil
}

// FetchBGPSparPrefixes fetches and parses thyme's data-spar file, which lists
// prefixes from the Special Purpose Address Registry (RFC 6890) and their
// origin AS.
func FetchBGPSparPrefixes(ctx context.Context, c *transport.Client, source string) (*models.BGPSparPrefixes, error) {
	url := transport.BuildThymeURL(c.ThymeBaseURL(), transport.SourceOrDefault(source, c.ThymeSource()), "data-spar")
	body, err := c.FetchText(ctx, url)
	if err != nil {
		return nil, err
	}
	return parseBGPSparPrefixes(body), nil
}

// FetchBGPSinglePfx fetches and parses thyme's data-singlepfx file, which
// tallies how many ASNs announce fewer than 20 prefixes, grouped by RIR.
func FetchBGPSinglePfx(ctx context.Context, c *transport.Client, source string) (*models.BGPSinglePfx, error) {
	url := transport.BuildThymeURL(c.ThymeBaseURL(), transport.SourceOrDefault(source, c.ThymeSource()), "data-singlepfx")
	body, err := c.FetchText(ctx, url)
	if err != nil {
		return nil, err
	}
	return parseBGPSinglePfx(body), nil
}

// parseBGPSummary parses the thyme data-summary file. Lines without a colon are
// skipped (including the "Analysis Summary" title and the dash separator). The
// key is the trimmed text before the first colon; the value is the trimmed text
// after it. Indented sub-metrics (which also use "key: value" form) are
// captured as their own entries.
func parseBGPSummary(data string) *models.BGPSummary {
	s := &models.BGPSummary{Entries: make([]models.BGPKeyValue, 0, 64)}
	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "-") {
			continue
		}
		colon := strings.Index(line, ":")
		if colon < 0 {
			continue
		}
		key := strings.TrimSpace(line[:colon])
		val := strings.TrimSpace(line[colon+1:])
		if key == "" {
			continue
		}
		s.Entries = append(s.Entries, models.BGPKeyValue{Key: key, Value: val})
	}
	return s
}

// parseBGPRawTable parses the thyme data-raw-table file. Each non-empty line is
// a "prefix\tASN" pair. Lines that do not split into exactly two fields are
// skipped defensively.
func parseBGPRawTable(data string) (*models.BGPRawTable, error) {
	t := &models.BGPRawTable{Routes: make([]models.BGPRoute, 0, 1000000)}
	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024) // large default for big files
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		// thyme separates prefix and ASN with a tab.
		fields := strings.Split(line, "\t")
		if len(fields) != 2 {
			// Fall back to any-whitespace split for robustness.
			fields = strings.Fields(line)
			if len(fields) != 2 {
				continue
			}
		}
		t.Routes = append(t.Routes, models.BGPRoute{Prefix: fields[0], ASN: fields[1]})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("BGP raw table scan failed: %w", err)
	}
	return t, nil
}

// parseBGPBadPrefixes parses thyme's data-badpfx-nos file. After a header
// (title + dash separator + column header), each non-empty line is
// "OriginAS<TAB>Address". Lines without two whitespace fields are skipped.
func parseBGPBadPrefixes(data string) *models.BGPBadPrefixes {
	r := &models.BGPBadPrefixes{Prefixes: make([]models.BGPBadPrefix, 0, 10000)}
	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "Prefixes longer") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		// Skip the column header row.
		if strings.EqualFold(fields[0], "Origin") || strings.EqualFold(fields[1], "Address") {
			continue
		}
		r.Prefixes = append(r.Prefixes, models.BGPBadPrefix{OriginAS: fields[0], Address: fields[1]})
	}
	return r
}

// parseBGPPerPrefixLength parses thyme's data-pfx-nos file. The file lays out
// "/N:count" tokens in a multi-column grid (several per line). Each token is
// split on ":" into length (the N in /N) and count. Tokens that fail to parse
// are skipped.
func parseBGPPerPrefixLength(data string) *models.BGPPerPrefixLength {
	r := &models.BGPPerPrefixLength{Counts: make([]models.BGPPrefixLengthCount, 0, 128)}
	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "Number of prefixes") {
			continue
		}
		for _, tok := range strings.Fields(line) {
			if !strings.HasPrefix(tok, "/") {
				continue
			}
			colon := strings.Index(tok, ":")
			if colon < 0 {
				continue
			}
			lengthStr := tok[1:colon] // strip leading "/"
			countStr := tok[colon+1:]
			length, err := strconv.Atoi(lengthStr)
			if err != nil {
				continue
			}
			count, err := strconv.Atoi(countStr)
			if err != nil {
				continue
			}
			r.Counts = append(r.Counts, models.BGPPrefixLengthCount{Length: length, Count: count, Raw: tok})
		}
	}
	return r
}

// parseBGPUsedAutnums parses thyme's data-used-autnums file. Each line is
// "<ASN> <Name> - <Description>, <CC>", e.g. "1 LVLT-1 - Level 3 Parent, LLC, US".
// The ASN is the first whitespace field; the country code is the text after the
// final comma; the FullName is everything between them.
func parseBGPUsedAutnums(data string) *models.BGPUsedAutnums {
	r := &models.BGPUsedAutnums{Autnums: make([]models.BGPUsedAutnum, 0, 80000)}
	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		asn := fields[0]
		// Country code is the token after the last comma.
		commaIdx := strings.LastIndex(line, ",")
		if commaIdx < 0 {
			continue
		}
		country := strings.TrimSpace(line[commaIdx+1:])
		// FullName is the text between the ASN and the comma (exclusive).
		// Guard against a comma that appears at or before the ASN (e.g. "1, Foo"
		// or ", foo"), where len(asn) > commaIdx would slice out of range.
		if len(asn) > commaIdx {
			continue
		}
		rest := strings.TrimSpace(line[len(asn):commaIdx])
		// Name is the first whitespace field of rest.
		nameFields := strings.Fields(rest)
		name := ""
		if len(nameFields) > 0 {
			name = nameFields[0]
		}
		r.Autnums = append(r.Autnums, models.BGPUsedAutnum{
			ASN:      asn,
			Name:     name,
			Country:  country,
			FullName: rest,
		})
	}
	return r
}

// parseBGPSparPrefixes parses thyme's data-spar file. After a header, each line
// is "<Prefix><TAB>OriginAS<TAB>Description". The description may contain
// spaces, so the line is split into at most 3 fields by tab (falling back to
// any-whitespace when the tab split yields only 2 fields).
func parseBGPSparPrefixes(data string) *models.BGPSparPrefixes {
	r := &models.BGPSparPrefixes{Prefixes: make([]models.BGPSparPrefix, 0, 64)}
	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "Prefixes from") {
			continue
		}
		// Tab-split first; if it yields 2 fields, the description is empty.
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			fields = strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
		}
		// Skip column header.
		if strings.EqualFold(fields[0], "Prefix") {
			continue
		}
		prefix := strings.TrimSpace(fields[0])
		originAS := strings.TrimSpace(fields[1])
		desc := ""
		if len(fields) >= 3 {
			desc = strings.TrimSpace(strings.Join(fields[2:], " "))
		}
		r.Prefixes = append(r.Prefixes, models.BGPSparPrefix{Prefix: prefix, OriginAS: originAS, Description: desc})
	}
	return r
}

// parseBGPSinglePfx parses thyme's data-singlepfx file. After a header, each
// line is "<PrefixCount><TAB><ASNCount><TAB><RIR>", e.g. "1 27539 Global".
// Non-numeric prefix/ASN counts are skipped.
func parseBGPSinglePfx(data string) *models.BGPSinglePfx {
	r := &models.BGPSinglePfx{Counts: make([]models.BGPSinglePfxCount, 0, 32)}
	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "Number of ASNs") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		// Skip column header.
		if strings.EqualFold(fields[0], "No.") {
			continue
		}
		prefixCount, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		asnCount, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		rir := strings.Join(fields[2:], " ")
		r.Counts = append(r.Counts, models.BGPSinglePfxCount{PrefixCount: prefixCount, ASNCount: asnCount, RIR: rir})
	}
	return r
}
