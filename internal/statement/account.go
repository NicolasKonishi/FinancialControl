package statement

import (
	"regexp"
	"strings"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

var (
	accountDayRe  = regexp.MustCompile(`(?i)^(\d{1,2})\s+([A-Za-zÀ-ÿ]{3,9})(?:\s+de)?\s+(\d{4})\b`)
	accountSkipRe = regexp.MustCompile(`(?i)(total de entradas|total de saídas|total de saidas|^saldo\b|rendimento l[ií]quido|movimenta[cç][oõ]es$|valores em r\$|tem alguma d[uú]vida|extrato gerado|ouvidoria|nubank\.com|n[aã]o nos responsabilizamos|asseguramos a autenticidade|nu financeira|nu pagamentos s\.a|cnpj:|cpf ag|p[aá]gina\s+\d|\d+\s+de\s+\d+$)`)
	amountOnlyRe  = regexp.MustCompile(`(?i)^[+\-]?\s*(?:R\$\s*)?\d{1,3}(?:\.\d{3})*,\d{2}$`)
	descAmountRe  = regexp.MustCompile(`(?i)^(.+?)\s+([+\-]?\s*(?:R\$\s*)?\d{1,3}(?:\.\d{3})*,\d{2})$`)
	accountTxnRe  = regexp.MustCompile(`(?i)(compra no d[eé]bito|compra no cr[eé]dito|transfer[eê]ncia recebida pelo pix|transfer[eê]ncia enviada pelo pix|transfer[eê]ncia recebida|transfer[eê]ncia enviada|aplica[cç][aã]o rdb|resgate rdb|pagamento de boleto)`)
	bankDetailRe  = regexp.MustCompile(`(?i)(ag[eê]ncia:|\d{2}\.\d{3}\.\d{3}/\d{4}|nu pagamentos|itau unibanco|itaú unibanco)`)
)

func isAccountStatement(text string) bool {
	hay := foldAccents(strings.ToLower(text))
	return strings.Contains(hay, "compra no debito") ||
		(strings.Contains(hay, "movimentacoes") && strings.Contains(hay, "total de entradas"))
}

func parseAccountStatement(text string, year, month int) []models.ParsedStatementItem {
	closingYear, closingMonth := closingFromText(text, year, month)
	var (
		currentDate time.Time
		pending     []string
		items       []models.ParsedStatementItem
	)
	flushAmount := func(amount float64, extraDesc string) {
		parts := append([]string{}, pending...)
		if extraDesc != "" {
			parts = append(parts, extraDesc)
		}
		pending = nil
		desc := strings.TrimSpace(strings.Join(parts, " "))
		desc = regexp.MustCompile(`\s+`).ReplaceAllString(desc, " ")
		if desc == "" || currentDate.IsZero() || amount == 0 {
			return
		}
		if accountSkipRe.FindString(desc) != "" {
			return
		}
		if accountTxnRe.FindString(desc) == "" && bankDetailRe.FindString(desc) != "" {
			return
		}
		if amount > 0 && isLikelyOutflow(desc) {
			amount = -amount
		}
		kind := classifyKind(desc, amount)
		items = append(items, models.ParsedStatementItem{
			Date:          currentDate.Format("2006-01-02"),
			Description:   cleanMerchant(desc),
			Amount:        round2(abs(amount)),
			Kind:          kind,
			SuggestedIcon: suggestIconFor(desc, kind),
		})
	}

	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(strings.ReplaceAll(raw, "\u00a0", " "))
		line = regexp.MustCompile(`[ \t]+`).ReplaceAllString(line, " ")
		if line == "" {
			continue
		}
		if match := accountDayRe.FindStringSubmatch(line); match != nil {
			if parsed, _, ok := extractDate(strings.TrimSpace(match[0]), closingYear, closingMonth, nil, nil); ok {
				currentDate = parsed
			}
			pending = nil
			rest := strings.TrimSpace(line[len(match[0]):])
			if rest == "" || accountSkipRe.FindString(rest) != "" {
				continue
			}
			line = rest
		}
		if accountSkipRe.FindString(line) != "" {
			pending = nil
			continue
		}
		collapsed := strings.ReplaceAll(line, " ", "")
		if amountOnlyRe.MatchString(collapsed) || amountOnlyRe.MatchString(line) {
			if amount, _, _, ok := extractAmount(line); ok {
				if strings.Contains(line, "-") {
					amount = -abs(amount)
				}
				flushAmount(amount, "")
				continue
			}
		}
		if match := descAmountRe.FindStringSubmatch(line); match != nil {
			desc := strings.TrimSpace(match[1])
			if accountDayRe.MatchString(desc) {
				continue
			}
			if amount, _, _, ok := extractAmount(match[2]); ok {
				if strings.Contains(match[2], "-") {
					amount = -abs(amount)
				}
				flushAmount(amount, desc)
				continue
			}
		}
		if len(pending) == 0 && bankDetailRe.FindString(line) != "" && accountTxnRe.FindString(line) == "" {
			continue
		}
		pending = append(pending, line)
	}
	return items
}

func isLikelyOutflow(desc string) bool {
	hay := foldAccents(strings.ToLower(desc))
	return strings.Contains(hay, "compra no debito") ||
		strings.Contains(hay, "compra no credito") ||
		strings.Contains(hay, "enviada") ||
		strings.Contains(hay, "enviado") ||
		strings.Contains(hay, "aplicacao")
}
