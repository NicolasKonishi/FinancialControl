package statement

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

var (
	ofxTrnRe  = regexp.MustCompile(`(?is)<STMTTRN>(.*?)</STMTTRN>`)
	ofxTagRe  = regexp.MustCompile(`(?i)<([A-Z0-9.]+)>([^<\r\n]*)`)
	ofxDateRe = regexp.MustCompile(`^(\d{8})`)
)

func looksLikeOFX(data []byte) bool {
	sample := strings.ToUpper(decodeText(data))
	if len(sample) > 1200 {
		sample = sample[:1200]
	}
	return strings.Contains(sample, "OFXHEADER") ||
		strings.Contains(sample, "<OFX>") ||
		strings.Contains(sample, "<STMTTRN>")
}

func parseOFX(data []byte, year, month int) (models.ParsedStatement, error) {
	text := decodeText(data)
	statementType := "account"
	upper := strings.ToUpper(text)
	if strings.Contains(upper, "<CREDITCARDMSGSRSV1>") || strings.Contains(upper, "<CCSTMTRS>") {
		statementType = "credit_card"
	}
	blocks := ofxTrnRe.FindAllStringSubmatch(text, -1)
	if len(blocks) == 0 {
		return models.ParsedStatement{}, ErrNoText
	}
	items := make([]models.ParsedStatementItem, 0, len(blocks))
	for _, block := range blocks {
		tags := ofxTags(block[1])
		amount, ok := parseSignedNumber(firstTag(tags, "TRNAMT"))
		if !ok || amount == 0 {
			continue
		}
		parsedDate, ok := parseOFXDate(firstTag(tags, "DTPOSTED"), year, month)
		if !ok {
			continue
		}
		desc := strings.TrimSpace(firstTag(tags, "MEMO"))
		if desc == "" {
			desc = strings.TrimSpace(firstTag(tags, "NAME"))
		}
		if desc == "" {
			continue
		}
		kind := classifyKind(desc, amount)
		items = append(items, models.ParsedStatementItem{
			Date:          parsedDate.Format("2006-01-02"),
			Description:   cleanMerchant(desc),
			Amount:        round2(abs(amount)),
			Kind:          kind,
			SuggestedIcon: suggestIconFor(desc, kind),
		})
	}
	periodStart := ofxDocumentDate(text, "DTSTART")
	periodEnd := ofxDocumentDate(text, "DTEND")
	balance := ofxDocumentAmount(text, "BALAMT")
	if len(items) == 0 {
		return models.ParsedStatement{
			Issuer:        detectIssuer(text),
			StatementType: statementType,
			PeriodStart:   periodStart,
			PeriodEnd:     periodEnd,
			Balance:       balance,
		}, nil
	}
	if periodStart == nil {
		periodStart = strPtr(items[0].Date)
	}
	if periodEnd == nil {
		periodEnd = strPtr(items[len(items)-1].Date)
	}
	return models.ParsedStatement{
		Issuer:        detectIssuer(text),
		StatementType: statementType,
		PeriodStart:   periodStart,
		PeriodEnd:     periodEnd,
		Balance:       balance,
		Items:         items,
	}, nil
}

func ofxDocumentDate(text, tag string) *string {
	re := regexp.MustCompile(`(?i)<` + regexp.QuoteMeta(tag) + `>([^<\r\n]*)`)
	match := re.FindStringSubmatch(text)
	if len(match) < 2 {
		return nil
	}
	parsed, ok := parseOFXDate(match[1], time.Now().Year(), int(time.Now().Month()))
	if !ok {
		return nil
	}
	formatted := parsed.Format("2006-01-02")
	return &formatted
}

func ofxDocumentAmount(text, tag string) *float64 {
	re := regexp.MustCompile(`(?i)<` + regexp.QuoteMeta(tag) + `>([^<\r\n]*)`)
	match := re.FindStringSubmatch(text)
	if len(match) < 2 {
		return nil
	}
	value, ok := parseSignedNumber(match[1])
	if !ok {
		return nil
	}
	value = round2(abs(value))
	return &value
}

func ofxTags(block string) map[string]string {
	out := make(map[string]string)
	for _, match := range ofxTagRe.FindAllStringSubmatch(block, -1) {
		key := strings.ToUpper(strings.TrimSpace(match[1]))
		if _, exists := out[key]; exists {
			continue
		}
		out[key] = strings.TrimSpace(match[2])
	}
	return out
}

func firstTag(tags map[string]string, name string) string {
	return tags[strings.ToUpper(name)]
}

func parseOFXDate(raw string, year, month int) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if i := strings.IndexByte(raw, '['); i >= 0 {
		raw = raw[:i]
	}
	if match := ofxDateRe.FindStringSubmatch(raw); match != nil {
		parsed, err := time.Parse("20060102", match[1])
		if err == nil {
			return parsed, true
		}
	}
	parsed, _, ok := extractDate(raw, year, month, nil, nil)
	return parsed, ok
}

func parseSignedNumber(raw string) (float64, bool) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "\u00a0", " "))
	if raw == "" {
		return 0, false
	}
	if amount, _, _, ok := extractAmount(raw); ok {
		if strings.HasPrefix(strings.TrimLeft(raw, " "), "-") && amount > 0 {
			amount = -amount
		}
		return amount, true
	}
	neg := strings.HasPrefix(raw, "-")
	s := strings.Trim(raw, "+- ")
	s = strings.ReplaceAll(s, ",", "")
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	if neg {
		value = -value
	}
	return value, true
}
