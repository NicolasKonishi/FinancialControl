package statement

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

var (
	// ErrEncrypted means the PDF needs a password.
	ErrEncrypted = errors.New("encrypted")
	// ErrNoText means the file had no extractable statement text.
	ErrNoText = errors.New("no_text")
)

var months = map[string]int{
	"jan": 1, "janeiro": 1, "fev": 2, "feb": 2, "fevereiro": 2,
	"mar": 3, "marco": 3, "abr": 4, "apr": 4, "abril": 4,
	"mai": 5, "may": 5, "maio": 5, "jun": 6, "junho": 6,
	"jul": 7, "julho": 7, "ago": 8, "aug": 8, "agosto": 8,
	"set": 9, "sep": 9, "sept": 9, "setembro": 9,
	"out": 10, "oct": 10, "outubro": 10, "nov": 11, "novembro": 11,
	"dez": 12, "dec": 12, "dezembro": 12,
}

var (
	paymentRe     = regexp.MustCompile(`(?i)\b(pagamento\s+recebido|pagamento\s+efetuado|pagamento\s+on[\s\-]?line|pagamento\s+da\s+fatura|pagto\.?\s+efetuado|pagto\.?\s+fatura|credit\s+payment|invoice\s+payment|pagamento\s+fatura)\b`)
	refundRe      = regexp.MustCompile(`(?i)\b(estorno|cancelamento|devolucao|devolução|refund)\b`)
	skipDescRe    = regexp.MustCompile(`(?i)(^total\b|^saldo\b|^limite\b|^vencimento\b|^fatura\s+atual|pagamento\s+m[ií]nimo|cr[eé]dito\s+dispon[ií]vel|lan[cç]amentos?\s+futuros|compras\s+parceladas|^transa[cç][oõ]es?\b|^nu\b|^nubank\b|^itau\b|^itaú\b|^extrato\b|^periodo\b|^per[ií]odo\b|cnpj|cpf\s*:|p[aá]gina\s+\d|www\.|http)`)
	brAmountRe    = regexp.MustCompile(`(?i)(-?\s*R\$\s*)?(-?\d{1,3}(?:\.\d{3})*,\d{2}|-?\d+,\d{2})`)
	usAmountRe    = regexp.MustCompile(`(?i)(-?\s*R\$\s*)?(-?\d{1,3}(?:,\d{3})*\.\d{2}|-?\d+\.\d{2})`)
	namedDateRe   = regexp.MustCompile(`(?i)\b(\d{1,2})\s+([A-Za-zÀ-ÿ]{3,9})\b(?:\s+(\d{4}))?`)
	numericDateRe = regexp.MustCompile(`\b(\d{1,2})[/-](\d{1,2})(?:[/-](\d{2,4}))?\b`)
	isoDateRe     = regexp.MustCompile(`\b(\d{4})-(\d{2})-(\d{2})\b`)
	faturaMonthRe = regexp.MustCompile(`(?i)fatura\s+de\s+([A-Za-zÀ-ÿ]+)\s+(\d{4})`)
	periodRe      = regexp.MustCompile(`(?i)(?:transa[cç][oõ]es?|per[ií]odo|de)\s+(\d{1,2})\s*([A-Za-zÀ-ÿ]{3,9})\s*(?:a|at[eé]|-|–)\s*(\d{1,2})\s*([A-Za-zÀ-ÿ]{3,9})(?:\s+(\d{4}))?`)
	dueRe         = regexp.MustCompile(`(?i)vencimento\s+(\d{1,2})\s*([A-Za-zÀ-ÿ]{3,9})\s+(\d{4})`)
	flatLineRe    = regexp.MustCompile(`(?i)(\d{1,2}\s+[A-Za-zÀ-ÿ]{3,9}(?:\s+\d{4})?|\d{1,2}[/-]\d{1,2}(?:[/-]\d{2,4})?)\s+(.{3,80}?)\s+(?:R\$\s*)?(-?\d{1,3}(?:\.\d{3})*,\d{2}|-?\d+,\d{2})`)
)

var iconKeywords = []struct {
	icon     string
	keywords []string
}{
	{icon: "subscriptions", keywords: []string{"netflix", "spotify", "disney", "prime video", "amazon prime", "youtube", "apple.com/bill", "paramount", "hbo", "globoplay", "openai", "chatgpt", "cursor", "icloud", "google one", "microsoft 365", "xbox game pass"}},
	{icon: "food", keywords: []string{"ifood", "rappi", "uber eats", "ubereats", "restaurante", "mcdonald", "burger king", "pizza", "outback", "habib", "subway", "delivery", "madero", "china in box", "giraffas"}},
	{icon: "market", keywords: []string{"carrefour", "extra", "assai", "assaí", "pao de acucar", "pão de açúcar", "supermercado", "hipermercado", "atacadao", "atacadão", "sams club", "savegnago", "hortifruti", "feira"}},
	{icon: "cafe", keywords: []string{"starbucks", "cafeteria", "padaria", "bakery", "coffee"}},
	{icon: "transport", keywords: []string{"uber", "99app", "99 pop", "99pay", "cabify", "metro", "metrô", "onibus", "ônibus", "bilhete unico", "bilhete único", "estacionamento", "parking", "sem parar", "conectcar", "veloe"}},
	{icon: "car", keywords: []string{"shell", "ipiranga", "petrobras", "auto posto", "posto ", "combustivel", "combustível"}},
	{icon: "health", keywords: []string{"farmacia", "farmácia", "drogaria", "droga raia", "pague menos", "panvel", "hospital", "laboratorio", "laboratório", "consulta", "unimed"}},
	{icon: "pets", keywords: []string{"petlove", "petz", "cobasi", "pet shop"}},
	{icon: "shopping", keywords: []string{"amazon", "mercadolivre", "mercado livre", "shopee", "magazine luiza", "magalu", "americanas", "casas bahia", "shein", "aliexpress"}},
	{icon: "clothing", keywords: []string{"renner", "c&a", "zara", "nike", "adidas", "hering", "riachuelo"}},
	{icon: "home", keywords: []string{"ikea", "leroy", "telhanorte", "aluguel", "condominio", "condomínio", "mobly"}},
	{icon: "utilities", keywords: []string{"enel", "cpfl", "sabesp", "comgas", "vivo fibra", "claro net", "internet", "cemig"}},
	{icon: "phone", keywords: []string{"vivo", "claro", "tim ", "oi celular"}},
	{icon: "travel", keywords: []string{"booking", "airbnb", "decolar", "latam", "gol linhas", "azul linhas", "hotel", "passagem"}},
	{icon: "leisure", keywords: []string{"cinema", "ingresso", "steam", "playstation", "show ", "bar "}},
	{icon: "education", keywords: []string{"udemy", "alura", "coursera", "escola", "faculdade"}},
	{icon: "gift", keywords: []string{"presente", "floricultura"}},
}

// ParseFile extracts statement lines from a PDF, CSV, or OFX upload.
func ParseFile(data []byte, year, month int) (models.ParsedStatement, error) {
	if len(data) == 0 {
		return models.ParsedStatement{}, ErrNoText
	}
	if year < 1 {
		now := time.Now()
		year = now.Year()
		if month < 1 {
			month = int(now.Month())
		}
	}
	if looksLikeOFX(data) {
		return parseOFX(data, year, month)
	}
	if looksLikeCSV(data) {
		text := decodeText(data)
		items := parseCSV(text, year, month)
		if len(items) > 0 {
			return models.ParsedStatement{
				Issuer:        detectIssuer(text),
				StatementType: "account",
				PeriodStart:   strPtr(items[0].Date),
				PeriodEnd:     strPtr(items[len(items)-1].Date),
				Items:         items,
			}, nil
		}
	}
	text, err := extractPDFText(data)
	if err != nil {
		if bytes.HasPrefix(bytes.TrimSpace(data), []byte("%PDF")) {
			return models.ParsedStatement{}, err
		}
		text = decodeText(data)
	}
	return ParseText(text, year, month)
}

// ParseText turns extracted statement text into dated purchases.
func ParseText(text string, year, month int) (models.ParsedStatement, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return models.ParsedStatement{}, ErrNoText
	}
	closingYear, closingMonth := closingFromText(text, year, month)
	periodStart, periodEnd := periodFromText(text, closingYear, closingMonth)

	var items []models.ParsedStatementItem
	if isAccountStatement(text) {
		items = parseAccountStatement(text, year, month)
	}
	if len(items) == 0 {
		seen := map[string]bool{}
		items = make([]models.ParsedStatementItem, 0)
		for _, line := range candidateLines(text) {
			item, ok := parseLine(line, closingYear, closingMonth, periodStart, periodEnd)
			if !ok {
				continue
			}
			key := item.Date + "|" + NormalizeDesc(item.Description) + "|" + strconv.Itoa(int(item.Amount*100+0.5))
			if seen[key] {
				continue
			}
			seen[key] = true
			items = append(items, item)
		}
		items = reclassifyAccountStatement(items)
	}
	statementType := "credit_card"
	if isAccountStatement(text) {
		statementType = "account"
	}
	if len(items) == 0 {
		return models.ParsedStatement{Issuer: detectIssuer(text), StatementType: statementType}, nil
	}
	out := models.ParsedStatement{
		Issuer:        detectIssuer(text),
		StatementType: statementType,
		Items:         items,
	}
	if periodStart != nil {
		value := periodStart.Format("2006-01-02")
		out.PeriodStart = &value
	} else {
		out.PeriodStart = &items[0].Date
	}
	if periodEnd != nil {
		value := periodEnd.Format("2006-01-02")
		out.PeriodEnd = &value
	} else {
		out.PeriodEnd = &items[len(items)-1].Date
	}
	return out, nil
}

func SuggestIcon(description string) string {
	hay := NormalizeDesc(description)
	for _, group := range iconKeywords {
		for _, keyword := range group.keywords {
			if strings.Contains(hay, NormalizeDesc(keyword)) {
				return group.icon
			}
		}
	}
	return "other"
}

func candidateLines(text string) []string {
	compact := strings.ReplaceAll(text, "\u00a0", " ")
	space := regexp.MustCompile(`[ \t]+`)
	compact = space.ReplaceAllString(compact, " ")
	seen := map[string]bool{}
	var lines []string
	add := func(line string) {
		line = strings.Trim(line, " -•\t")
		if len(line) < 8 || seen[line] {
			return
		}
		seen[line] = true
		lines = append(lines, line)
	}
	for _, raw := range strings.Split(compact, "\n") {
		add(raw)
	}
	flat := regexp.MustCompile(`\s+`).ReplaceAllString(compact, " ")
	for _, match := range flatLineRe.FindAllStringSubmatch(flat, -1) {
		add(strings.TrimSpace(strings.Join(match[1:], " ")))
	}
	return lines
}

func parseLine(line string, closingYear, closingMonth int, periodStart, periodEnd *time.Time) (models.ParsedStatementItem, bool) {
	amount, start, end, ok := extractAmount(line)
	if !ok || amount == 0 || abs(amount) > 1_000_000 {
		return models.ParsedStatementItem{}, false
	}
	before := strings.Trim(line[:start], " -–·|")
	after := strings.TrimSpace(line[end:])
	if after != "" {
		if matched, _ := regexp.MatchString(`(?i)^(D|C|CR|DB|\*)$`, after); !matched {
			return models.ParsedStatementItem{}, false
		}
	}
	parsedDate, dateEnd, ok := extractDate(before, closingYear, closingMonth, periodStart, periodEnd)
	if !ok {
		return models.ParsedStatementItem{}, false
	}
	description := strings.Trim(before[dateEnd:], " -–·|:")
	description = regexp.MustCompile(`\s+`).ReplaceAllString(description, " ")
	description = strings.Trim(description, "* ")
	if len(description) < 2 || skipDescRe.FindString(description) != "" {
		return models.ParsedStatementItem{}, false
	}
	if matched, _ := regexp.MatchString(`^[\d./\-]+$`, description); matched {
		return models.ParsedStatementItem{}, false
	}

	kind := classifyKind(description, amount)
	return models.ParsedStatementItem{
		Date:          parsedDate.Format("2006-01-02"),
		Description:   cleanMerchant(description),
		Amount:        round2(abs(amount)),
		Kind:          kind,
		SuggestedIcon: suggestIconFor(description, kind),
	}, true
}

func classifyKind(description string, amount float64) string {
	hay := foldAccents(strings.ToLower(description))
	if refundRe.FindString(description) != "" {
		return "refund"
	}
	if paymentRe.FindString(description) != "" {
		return "payment"
	}
	if isInternalMove(hay) {
		return "transfer"
	}
	if isIncomeMemo(hay) {
		return "income"
	}
	if amount < 0 {
		return "expense"
	}
	return "expense"
}

func isInternalMove(hay string) bool {
	return strings.Contains(hay, "aplicacao rdb") ||
		strings.Contains(hay, "resgate rdb") ||
		strings.Contains(hay, "aplicacao automatica") ||
		strings.Contains(hay, "resgate automatico") ||
		strings.Contains(hay, "transferencia entre contas")
}

func isIncomeMemo(hay string) bool {
	return strings.Contains(hay, "recebida") ||
		strings.Contains(hay, "recebido") ||
		strings.Contains(hay, "rendimento") ||
		strings.Contains(hay, "salario") ||
		strings.Contains(hay, "deposito")
}

func suggestIconFor(description, kind string) string {
	if kind == "income" {
		icon := SuggestIcon(description)
		if icon == "other" {
			return "salary"
		}
		return icon
	}
	return SuggestIcon(description)
}

func extractAmount(line string) (float64, int, int, bool) {
	matches := brAmountRe.FindAllStringIndex(line, -1)
	us := false
	if len(matches) == 0 {
		matches = usAmountRe.FindAllStringIndex(line, -1)
		us = true
	}
	if len(matches) == 0 {
		return 0, 0, 0, false
	}
	span := matches[len(matches)-1]
	raw := line[span[0]:span[1]]
	var value float64
	var ok bool
	if us {
		value, ok = parseUSAmount(raw)
	} else {
		value, ok = parseBRAmount(raw)
	}
	if !ok {
		return 0, 0, 0, false
	}
	prefix := strings.TrimRight(line[:span[0]], " ")
	if strings.HasSuffix(prefix, "-") && !strings.HasSuffix(prefix, " -") {
		value = -abs(value)
	}
	return value, span[0], span[1], true
}

func parseBRAmount(raw string) (float64, bool) {
	s := strings.ReplaceAll(strings.ReplaceAll(raw, "R$", ""), " ", "")
	s = strings.ReplaceAll(s, "\u00a0", "")
	neg := strings.HasPrefix(s, "-") || strings.HasSuffix(s, "-")
	s = strings.Trim(s, "-")
	if matched, _ := regexp.MatchString(`^\d{1,3}(\.\d{3})*,\d{2}$|^\d+,\d{2}$`, s); !matched {
		return 0, false
	}
	s = strings.ReplaceAll(strings.ReplaceAll(s, ".", ""), ",", ".")
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	if neg {
		value = -value
	}
	return value, true
}

func parseUSAmount(raw string) (float64, bool) {
	s := strings.ReplaceAll(strings.ReplaceAll(raw, "R$", ""), " ", "")
	neg := strings.HasPrefix(s, "-") || strings.HasSuffix(s, "-")
	s = strings.Trim(s, "-")
	if matched, _ := regexp.MatchString(`^\d{1,3}(,\d{3})*\.\d{2}$|^\d+\.\d{2}$`, s); !matched {
		return 0, false
	}
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

func extractDate(text string, closingYear, closingMonth int, periodStart, periodEnd *time.Time) (time.Time, int, bool) {
	if match := isoDateRe.FindStringSubmatchIndex(text); match != nil {
		year, _ := strconv.Atoi(text[match[2]:match[3]])
		month, _ := strconv.Atoi(text[match[4]:match[5]])
		day, _ := strconv.Atoi(text[match[6]:match[7]])
		parsed, err := time.Parse("2006-01-02", fmt.Sprintf("%04d-%02d-%02d", year, month, day))
		if err == nil {
			return parsed, match[1], true
		}
	}
	if match := numericDateRe.FindStringSubmatchIndex(text); match != nil {
		day, _ := strconv.Atoi(text[match[2]:match[3]])
		month, _ := strconv.Atoi(text[match[4]:match[5]])
		year := closingYear
		if match[6] != -1 {
			year, _ = strconv.Atoi(text[match[6]:match[7]])
			if year < 100 {
				year += 2000
			}
		} else {
			year = coerceYear(0, month, closingYear, closingMonth)
		}
		parsed, err := time.Parse("2006-01-02", fmt.Sprintf("%04d-%02d-%02d", year, month, day))
		if err == nil {
			return parsed, match[1], true
		}
	}
	if match := namedDateRe.FindStringSubmatchIndex(text); match != nil {
		day, _ := strconv.Atoi(text[match[2]:match[3]])
		month := monthNum(text[match[4]:match[5]])
		if month == 0 {
			return time.Time{}, 0, false
		}
		year := closingYear
		if match[6] != -1 {
			year, _ = strconv.Atoi(text[match[6]:match[7]])
		} else {
			year = coerceYear(0, month, closingYear, closingMonth)
		}
		parsed, err := time.Parse("2006-01-02", fmt.Sprintf("%04d-%02d-%02d", year, month, day))
		if err == nil {
			if periodStart != nil && periodEnd != nil {
				if parsed.Before(*periodStart) || parsed.After(*periodEnd) {
					if shifted, ok := shiftIntoPeriod(parsed, *periodStart, *periodEnd); ok {
						parsed = shifted
					}
				}
			}
			return parsed, match[1], true
		}
	}
	return time.Time{}, 0, false
}

func coerceYear(raw, month, closingYear, closingMonth int) int {
	if raw > 0 {
		if raw < 100 {
			return raw + 2000
		}
		return raw
	}
	if month > closingMonth {
		return closingYear - 1
	}
	return closingYear
}

func shiftIntoPeriod(parsed, start, end time.Time) (time.Time, bool) {
	for _, delta := range []int{0, -1, 1} {
		candidate := parsed.AddDate(delta, 0, 0)
		if !candidate.Before(start) && !candidate.After(end) {
			return candidate, true
		}
	}
	return time.Time{}, false
}

func closingFromText(text string, fallbackYear, fallbackMonth int) (int, int) {
	if match := faturaMonthRe.FindStringSubmatch(text); match != nil {
		if month := monthNum(match[1]); month > 0 {
			year, _ := strconv.Atoi(match[2])
			return year, month
		}
	}
	if match := dueRe.FindStringSubmatch(text); match != nil {
		if month := monthNum(match[2]); month > 0 {
			year, _ := strconv.Atoi(match[3])
			return year, month
		}
	}
	return fallbackYear, fallbackMonth
}

func periodFromText(text string, closingYear, closingMonth int) (*time.Time, *time.Time) {
	_ = closingMonth
	match := periodRe.FindStringSubmatch(text)
	if match == nil {
		return nil, nil
	}
	startMonth := monthNum(match[2])
	endMonth := monthNum(match[4])
	if startMonth == 0 || endMonth == 0 {
		return nil, nil
	}
	endYear := closingYear
	if match[5] != "" {
		endYear, _ = strconv.Atoi(match[5])
	}
	startYear := endYear
	if startMonth > endMonth {
		startYear = endYear - 1
	}
	startDay, _ := strconv.Atoi(match[1])
	endDay, _ := strconv.Atoi(match[3])
	start, err1 := time.Parse("2006-01-02", fmt.Sprintf("%04d-%02d-%02d", startYear, startMonth, startDay))
	end, err2 := time.Parse("2006-01-02", fmt.Sprintf("%04d-%02d-%02d", endYear, endMonth, endDay))
	if err1 != nil || err2 != nil {
		return nil, nil
	}
	return &start, &end
}

func monthNum(raw string) int {
	key := strings.ToLower(strings.TrimSpace(raw))
	repl := strings.NewReplacer("ç", "c", "ã", "a", "á", "a", "é", "e", "ê", "e", "í", "i", "ó", "o", "ô", "o", "ú", "u")
	key = repl.Replace(key)
	if len(key) >= 3 {
		if month, ok := months[key[:3]]; ok {
			return month
		}
	}
	return months[key]
}

func detectIssuer(text string) string {
	head := NormalizeDesc(text)
	if len(head) > 80 {
		head = NormalizeDesc(text[:80])
	} else {
		head = NormalizeDesc(text)
	}
	if head == "nu" || strings.HasPrefix(head, "nu ") {
		return "nubank"
	}
	hay := foldAccents(NormalizeDesc(text))
	if len(hay) > 4000 {
		hay = hay[:4000]
	}
	checks := []struct {
		issuer  string
		needles []string
	}{
		{"nubank", []string{"nubank", "nu pagamentos"}},
		{"itau", []string{"banco itau", "itau", "itau unibanco"}},
		{"inter", []string{"banco inter", "inter pag"}},
		{"c6", []string{"c6 bank", "banco c6"}},
		{"santander", []string{"santander"}},
		{"bradesco", []string{"bradesco"}},
		{"bb", []string{"banco do brasil"}},
		{"picpay", []string{"picpay"}},
		{"mercado-pago", []string{"mercado pago", "mercadopago"}},
	}
	for _, check := range checks {
		for _, needle := range check.needles {
			if strings.Contains(hay, needle) {
				return check.issuer
			}
		}
	}
	return "unknown"
}

func cleanMerchant(description string) string {
	cleaned := regexp.MustCompile(`\s+`).ReplaceAllString(description, " ")
	cleaned = strings.Trim(cleaned, " -*")
	cleaned = strings.TrimSpace(memoPrefixRe.ReplaceAllString(cleaned, ""))
	if cut := cutBankDetails(cleaned); cut != "" {
		cleaned = cut
	}
	place := regexp.MustCompile(`(?i)\b(sao paulo|são paulo|brasil|brazil|br)\b`)
	cleaned = strings.TrimSpace(place.ReplaceAllString(cleaned, " "))
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")
	cleaned = strings.Trim(cleaned, " -*")
	if i := strings.Index(cleaned, "*"); i >= 0 {
		left := strings.TrimSpace(cleaned[:i])
		right := strings.TrimSpace(cleaned[i+1:])
		if NormalizeDesc(left) != "" && strings.Contains(NormalizeDesc(right), NormalizeDesc(left)) {
			cleaned = right
		} else if NormalizeDesc(right) != "" && strings.Contains(NormalizeDesc(left), NormalizeDesc(right)) {
			cleaned = left
		}
	}
	if cleaned == "" {
		cleaned = strings.TrimSpace(description)
	}
	if len(cleaned) > 80 {
		cleaned = strings.TrimSpace(cleaned[:80])
	}
	return cleaned
}

var memoPrefixRe = regexp.MustCompile(`(?i)^(compra no d[eé]bito|compra no cr[eé]dito|transfer[eê]ncia enviada pelo pix|transfer[eê]ncia recebida pelo pix|transfer[eê]ncia recebida)\s*-?\s*`)

func cutBankDetails(description string) string {
	parts := strings.Split(description, " - ")
	if len(parts) < 2 {
		return description
	}
	keep := parts[0]
	for i := 1; i < len(parts); i++ {
		rest := foldAccents(strings.ToLower(strings.Join(parts[i:], " - ")))
		if strings.Contains(rest, "agencia") ||
			strings.Contains(rest, "nu pagamentos") ||
			strings.Contains(rest, "itau unibanco") ||
			strings.Contains(rest, "cnpj") ||
			strings.Contains(rest, "••") ||
			regexp.MustCompile(`\d{2}\.\d{3}\.\d{3}/\d{4}`).FindString(rest) != "" {
			return strings.TrimSpace(keep)
		}
		keep = strings.TrimSpace(keep + " - " + parts[i])
	}
	return strings.TrimSpace(keep)
}

func reclassifyAccountStatement(items []models.ParsedStatementItem) []models.ParsedStatementItem {
	unlabeled := 0
	labeled := 0
	for _, item := range items {
		if paymentRe.FindString(item.Description) != "" {
			labeled++
			continue
		}
		if item.Kind == "payment" {
			unlabeled++
		}
	}
	if len(items) >= 3 && unlabeled*10 >= len(items)*7 && labeled <= 2 {
		for i := range items {
			if items[i].Kind == "payment" && paymentRe.FindString(items[i].Description) == "" {
				items[i].Kind = "expense"
			}
		}
	}
	return items
}

func looksLikeCSV(data []byte) bool {
	if bytes.HasPrefix(bytes.TrimSpace(data), []byte("%PDF")) || looksLikeOFX(data) {
		return false
	}
	sample := decodeText(data)
	if len(sample) > 4000 {
		sample = sample[:4000]
	}
	if !strings.Contains(sample, ",") && !strings.Contains(sample, ";") {
		return false
	}
	header := sample
	if i := strings.IndexByte(sample, '\n'); i >= 0 {
		header = sample[:i]
	}
	header = strings.ToLower(header)
	for _, token := range []string{"data", "date", "descr", "title", "valor", "amount"} {
		if strings.Contains(header, token) {
			return true
		}
	}
	return false
}

func decodeText(data []byte) string {
	return string(bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}))
}

func parseCSV(text string, year, month int) []models.ParsedStatementItem {
	delimiter := ','
	if strings.Count(text, ";") > strings.Count(text, ",") {
		delimiter = ';'
	}
	reader := csv.NewReader(strings.NewReader(text))
	reader.Comma = delimiter
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil || len(rows) < 2 {
		return nil
	}
	header := rows[0]
	dateIdx, descIdx, amountIdx := csvColumns(header)
	if dateIdx < 0 || descIdx < 0 || amountIdx < 0 {
		return nil
	}
	items := make([]models.ParsedStatementItem, 0)
	for _, row := range rows[1:] {
		if len(row) <= dateIdx || len(row) <= descIdx || len(row) <= amountIdx {
			continue
		}
		parsedDate, _, ok := extractDate(strings.TrimSpace(row[dateIdx]), year, month, nil, nil)
		if !ok {
			continue
		}
		desc := strings.TrimSpace(row[descIdx])
		amount, ok := parseSignedNumber(strings.TrimSpace(row[amountIdx]))
		if !ok || desc == "" {
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
	return reclassifyAccountStatement(items)
}

func csvColumns(header []string) (dateIdx, descIdx, amountIdx int) {
	dateIdx, descIdx, amountIdx = -1, -1, -1
	for i, raw := range header {
		name := strings.ToLower(strings.TrimSpace(raw))
		switch {
		case dateIdx < 0 && (name == "data" || name == "date" || name == "dt" || strings.Contains(name, "data")):
			dateIdx = i
		case descIdx < 0 && (strings.Contains(name, "descr") || name == "title" || strings.Contains(name, "histor") || name == "memo"):
			descIdx = i
		case amountIdx < 0 && (name == "valor" || name == "amount" || name == "value" || name == "vlr"):
			amountIdx = i
		}
	}
	return dateIdx, descIdx, amountIdx
}

func strPtr(value string) *string {
	return &value
}

func foldAccents(value string) string {
	repl := strings.NewReplacer(
		"á", "a", "à", "a", "ã", "a", "â", "a",
		"é", "e", "ê", "e", "í", "i",
		"ó", "o", "ô", "o", "õ", "o", "ú", "u", "ü", "u", "ç", "c",
	)
	return repl.Replace(value)
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
