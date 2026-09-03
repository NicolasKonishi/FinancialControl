package statement_test

import (
	"os"
	"strings"
	"testing"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
	"github.com/NicolasKonishi/FinancialControl/internal/statement"
)

const nubankText = `
nu
Fatura de agosto 2026
Vencimento 17 AGO 2026
Transações de 08 JUL a 07 AGO

08 JUL Pagamento recebido em 08 JUL 2026 - R$ 1.234,56
10 JUL IFOOD *IFOOD SAO PAULO BR R$ 42,90
11 JUL UBER *UBER TRIP R$ 18,50
15 JUL SUPERMERCADO EXTRA 156,78
20 JUL AMAZON.COM.BR 2/4 R$ 89,90
01 AGO NETFLIX.COM R$ 55,90
02 AGO SPOTIFY 21,90
05 AGO ESTORNO UBER *UBER 18,50
`

func TestParseTextNubank(t *testing.T) {
	parsed, err := statement.ParseText(nubankText, 2026, 8)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Issuer != "nubank" {
		t.Fatalf("issuer = %q, want nubank", parsed.Issuer)
	}
	expenses := map[string]float64{}
	var payments, refunds int
	for _, item := range parsed.Items {
		switch item.Kind {
		case "payment":
			payments++
		case "refund":
			refunds++
		default:
			expenses[item.Description] = item.Amount
		}
	}
	if payments != 1 || refunds != 1 {
		t.Fatalf("payments=%d refunds=%d items=%d", payments, refunds, len(parsed.Items))
	}
	if expenses["IFOOD"] != 42.9 {
		t.Fatalf("ifood = %v items=%v", expenses["IFOOD"], expenses)
	}
	if expenses["UBER TRIP"] != 18.5 {
		t.Fatalf("uber = %v", expenses["UBER TRIP"])
	}
	if expenses["NETFLIX.COM"] != 55.9 {
		t.Fatalf("netflix = %v", expenses["NETFLIX.COM"])
	}
	if statement.SuggestIcon("IFOOD") != "food" {
		t.Fatalf("icon ifood = %s", statement.SuggestIcon("IFOOD"))
	}
}

func TestParseCSV(t *testing.T) {
	csv := "Data;Descrição;Valor\n10/07/2026;IFOOD;42,90\n11/07/2026;UBER TRIP;18,50\n"
	parsed, err := statement.ParseFile([]byte(csv), 2026, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Items) != 2 {
		t.Fatalf("items = %d, want 2 (%+v)", len(parsed.Items), parsed.Items)
	}
}

const nubankCheckingCSV = `Data,Valor,Identificador,Descrição
01/09/2026,2000.00,id-1,Transferência Recebida - EMPRESA EXEMPLO LTDA - 12.345.678/0001-90 - NU PAGAMENTOS - IP (0260) Agência: 1 Conta: 11111111-1
01/09/2026,-2000.00,id-2,Aplicação RDB
01/09/2026,-43.46,id-3,Compra no débito - SAO ROQUE
02/09/2026,-60.00,id-4,Transferência enviada pelo Pix - Pessoa Exemplo - •••.000.000-•• - NU PAGAMENTOS - IP (0260) Agência: 1 Conta: 22222222-2
02/09/2026,-62.63,id-5,Compra no débito - SUPERMERCADO IMPACTO T
02/09/2026,100.00,id-6,Transferência recebida pelo Pix - COMERCIO EXEMPLO LTDA - 00.000.000/0001-00 - ITAÚ UNIBANCO S.A. (0341) Agência: 1 Conta: 33333-3
`

const nubankCheckingOFX = `OFXHEADER:100
DATA:OFXSGML
VERSION:102
SECURITY:NONE
ENCODING:UTF-8
CHARSET:NONE
COMPRESSION:NONE
OLDFILEUID:NONE
NEWFILEUID:NONE
<OFX>
<BANKMSGSRSV1>
<STMTTRNRS>
<STMTRS>
<BANKACCTFROM>
<ACCTTYPE>CHECKING</ACCTTYPE>
</BANKACCTFROM>
<BANKTRANLIST>
<STMTTRN>
<TRNTYPE>CREDIT</TRNTYPE>
<DTPOSTED>20260901000000[-3:BRT]</DTPOSTED>
<TRNAMT>2000.00</TRNAMT>
<FITID>id-1</FITID>
<MEMO>Transferência Recebida - EMPRESA EXEMPLO LTDA - 12.345.678/0001-90 - NU PAGAMENTOS - IP (0260) Agência: 1 Conta: 11111111-1</MEMO>
</STMTTRN>
<STMTTRN>
<TRNTYPE>DEBIT</TRNTYPE>
<DTPOSTED>20260901000000[-3:BRT]</DTPOSTED>
<TRNAMT>-2000.00</TRNAMT>
<FITID>id-2</FITID>
<MEMO>Aplicação RDB</MEMO>
</STMTTRN>
<STMTTRN>
<TRNTYPE>DEBIT</TRNTYPE>
<DTPOSTED>20260901000000[-3:BRT]</DTPOSTED>
<TRNAMT>-43.46</TRNAMT>
<FITID>id-3</FITID>
<MEMO>Compra no débito - SAO ROQUE</MEMO>
</STMTTRN>
<STMTTRN>
<TRNTYPE>DEBIT</TRNTYPE>
<DTPOSTED>20260902000000[-3:BRT]</DTPOSTED>
<TRNAMT>-60.00</TRNAMT>
<FITID>id-4</FITID>
<MEMO>Transferência enviada pelo Pix - Pessoa Exemplo - •••.000.000-•• - NU PAGAMENTOS - IP (0260) Agência: 1 Conta: 22222222-2</MEMO>
</STMTTRN>
<STMTTRN>
<TRNTYPE>DEBIT</TRNTYPE>
<DTPOSTED>20260902000000[-3:BRT]</DTPOSTED>
<TRNAMT>-62.63</TRNAMT>
<FITID>id-5</FITID>
<MEMO>Compra no débito - SUPERMERCADO IMPACTO T</MEMO>
</STMTTRN>
<STMTTRN>
<TRNTYPE>CREDIT</TRNTYPE>
<DTPOSTED>20260902000000[-3:BRT]</DTPOSTED>
<TRNAMT>100.00</TRNAMT>
<FITID>id-6</FITID>
<MEMO>Transferência recebida pelo Pix - COMERCIO EXEMPLO LTDA - 00.000.000/0001-00 - ITAÚ UNIBANCO S.A. (0341) Agência: 1 Conta: 33333-3</MEMO>
</STMTTRN>
</BANKTRANLIST>
</STMTRS>
</STMTTRNRS>
</BANKMSGSRSV1>
</OFX>
`

const nubankCheckingText = `
Nu Pagamentos S.A.
Movimentações
01 SET 2026 Total de entradas + 2.000,00
Transferência Recebida EMPRESA EXEMPLO LTDA - 2.000,00
12.345.678/0001-90 - NU PAGAMENTOS - IP (0260)
Agência: 1 Conta: 11111111-1
Total de saídas - 2.043,46
Aplicação RDB 2.000,00
Compra no débito SAO ROQUE 43,46
02 SET 2026 Total de entradas + 100,00
Transferência recebida pelo Pix COMERCIO EXEMPLO LTDA - 100,00
00.000.000/0001-00 - ITAÚ UNIBANCO S.A. (0341) Agência: 1 Conta: 33333-3
Total de saídas - 122,63
Transferência enviada pelo Pix Pessoa Exemplo - •••.000.000-•• - 60,00
NU PAGAMENTOS - IP (0260) Agência: 1 Conta: 22222222-2
Compra no débito SUPERMERCADO IMPACTO T 62,63
`

func TestParseNubankCheckingCSVAndOFX(t *testing.T) {
	for _, file := range []struct {
		name string
		data []byte
	}{
		{"csv", []byte(nubankCheckingCSV)},
		{"ofx", []byte(nubankCheckingOFX)},
	} {
		parsed, err := statement.ParseFile(file.data, 2026, 9)
		if err != nil {
			t.Fatalf("%s: %v", file.name, err)
		}
		assertNubankChecking(t, file.name, parsed)
	}
}

func TestParseNubankCheckingPDFText(t *testing.T) {
	parsed, err := statement.ParseText(nubankCheckingText, 2026, 9)
	if err != nil {
		t.Fatal(err)
	}
	assertNubankChecking(t, "pdf-text", parsed)
}

func TestParseLiveNubankCheckingFiles(t *testing.T) {
	base := "/home/nicolas/Downloads/NU_976689446_01SET2026_02SET2026"
	for _, ext := range []string{".csv", ".ofx", ".pdf"} {
		data, err := os.ReadFile(base + ext)
		if err != nil {
			t.Skipf("live Nubank files not present: %v", err)
		}
		parsed, err := statement.ParseFile(data, 2026, 9)
		if err != nil {
			t.Fatalf("%s: %v", ext, err)
		}
		assertNubankChecking(t, "live"+ext, parsed)
	}
}

func assertNubankChecking(t *testing.T, label string, parsed models.ParsedStatement) {
	t.Helper()
	type row struct {
		date   string
		kind   string
		amount float64
	}
	want := []row{
		{"2026-09-01", "income", 2000},
		{"2026-09-01", "transfer", 2000},
		{"2026-09-01", "expense", 43.46},
		{"2026-09-02", "expense", 60},
		{"2026-09-02", "expense", 62.63},
		{"2026-09-02", "income", 100},
	}
	got := make([]row, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		got = append(got, row{item.Date, item.Kind, item.Amount})
	}
	if len(got) != len(want) {
		t.Fatalf("%s items = %d, want %d (%+v)", label, len(got), len(want), parsed.Items)
	}
	used := make([]bool, len(got))
	for _, w := range want {
		found := false
		for i, g := range got {
			if used[i] {
				continue
			}
			if g == w {
				used[i] = true
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s missing %+v in %+v", label, w, parsed.Items)
		}
	}
	if parsed.Issuer != "nubank" {
		t.Fatalf("%s issuer = %q, want nubank", label, parsed.Issuer)
	}
}

func TestParseNubankCreditCardOFXMetadata(t *testing.T) {
	ofx := `OFXHEADER:100
<OFX><CREDITCARDMSGSRSV1><CCSTMTRS>
<BANKTRANLIST><DTSTART>20260814000000[-3:BRT]</DTSTART><DTEND>20260914000000[-3:BRT]</DTEND>
<STMTTRN><TRNTYPE>DEBIT</TRNTYPE><DTPOSTED>20260903000000[-3:BRT]</DTPOSTED><TRNAMT>-34.90</TRNAMT><MEMO>Amazon Prime Canais</MEMO></STMTTRN>
<STMTTRN><TRNTYPE>CREDIT</TRNTYPE><DTPOSTED>20260902000000[-3:BRT]</DTPOSTED><TRNAMT>10.00</TRNAMT><MEMO>Estorno de compra (Amazon)</MEMO></STMTTRN>
<STMTTRN><TRNTYPE>CREDIT</TRNTYPE><DTPOSTED>20260814000000[-3:BRT]</DTPOSTED><TRNAMT>1186.16</TRNAMT><MEMO>Pagamento recebido</MEMO></STMTTRN>
</BANKTRANLIST><LEDGERBAL><BALAMT>-2673.93</BALAMT></LEDGERBAL>
</CCSTMTRS></CREDITCARDMSGSRSV1></OFX>`
	parsed, err := statement.ParseFile([]byte(ofx), 2026, 9)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.StatementType != "credit_card" {
		t.Fatalf("statement_type = %q", parsed.StatementType)
	}
	if parsed.PeriodStart == nil || *parsed.PeriodStart != "2026-08-14" || parsed.PeriodEnd == nil || *parsed.PeriodEnd != "2026-09-14" {
		t.Fatalf("period = %v..%v", parsed.PeriodStart, parsed.PeriodEnd)
	}
	if parsed.Balance == nil || *parsed.Balance != 2673.93 {
		t.Fatalf("balance = %v", parsed.Balance)
	}
	if len(parsed.Items) != 3 || parsed.Items[1].Kind != "refund" || parsed.Items[2].Kind != "payment" {
		t.Fatalf("items = %+v", parsed.Items)
	}
}

func TestParseSimplePDF(t *testing.T) {
	pdf := simplePDF([]string{
		"Fatura de agosto 2026",
		"10 JUL IFOOD *IFOOD R$ 42,90",
		"11 JUL UBER *UBER TRIP R$ 18,50",
	})
	parsed, err := statement.ParseFile(pdf, 2026, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Items) != 2 {
		t.Fatalf("pdf items = %d (%+v)", len(parsed.Items), parsed.Items)
	}
}

func simplePDF(lines []string) []byte {
	var ops strings.Builder
	y := 720
	for _, line := range lines {
		escaped := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`).Replace(line)
		ops.WriteString("BT /F1 12 Tf 50 ")
		ops.WriteString(itoa(y))
		ops.WriteString(" Td (")
		ops.WriteString(escaped)
		ops.WriteString(") Tj ET\n")
		y -= 18
	}
	stream := ops.String()
	objects := []string{
		"1 0 obj<< /Type /Catalog /Pages 2 0 R >>endobj\n",
		"2 0 obj<< /Type /Pages /Kids [3 0 R] /Count 1 >>endobj\n",
		"3 0 obj<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>endobj\n",
		"4 0 obj<< /Length " + itoa(len(stream)) + " >>stream\n" + stream + "endstream\nendobj\n",
		"5 0 obj<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>endobj\n",
	}
	var body strings.Builder
	body.WriteString("%PDF-1.1\n")
	offsets := make([]int, len(objects))
	for i, obj := range objects {
		offsets[i] = body.Len()
		body.WriteString(obj)
	}
	xref := body.Len()
	body.WriteString("xref\n0 ")
	body.WriteString(itoa(len(objects) + 1))
	body.WriteString("\n0000000000 65535 f \n")
	for _, off := range offsets {
		body.WriteString(pad10(off) + " 00000 n \n")
	}
	body.WriteString("trailer<< /Size ")
	body.WriteString(itoa(len(objects) + 1))
	body.WriteString(" /Root 1 0 R >>\nstartxref\n")
	body.WriteString(itoa(xref))
	body.WriteString("\n%%EOF\n")
	return []byte(body.String())
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits [16]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}

func pad10(n int) string {
	s := itoa(n)
	return strings.Repeat("0", 10-len(s)) + s
}
