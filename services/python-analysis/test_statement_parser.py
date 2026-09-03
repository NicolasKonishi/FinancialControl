import unittest

from statement_parser import parse_statement_bytes, parse_statement_text, suggest_icon


NUBANK_TEXT = """
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
"""


ITAU_TEXT = """
Banco Itaú
Extrato cartão
10/07/2026 SUPERMERCADO EXTRA 156,78
11/07/2026 UBER TRIP 18,50
15/07/2026 IFOOD 42,90
20/07/2026 Pagamento fatura -1.200,00
"""


class StatementParserTest(unittest.TestCase):
    def test_nubank_extracts_purchases_and_skips_payment(self):
        parsed = parse_statement_text(NUBANK_TEXT, 2026, 8)
        self.assertEqual(parsed.issuer, "nubank")
        kinds = {item.description: item.kind for item in parsed.items}
        self.assertEqual(kinds.get("Pagamento recebido em 08 JUL 2026"), "payment")
        self.assertEqual(kinds.get("ESTORNO UBER"), "refund")
        expenses = [item for item in parsed.items if item.kind == "expense"]
        self.assertEqual(len(expenses), 5)
        by_desc = {item.description: item for item in expenses}
        self.assertEqual(by_desc["IFOOD"].amount, 42.90)
        self.assertEqual(by_desc["IFOOD"].date, "2026-07-10")
        self.assertEqual(by_desc["IFOOD"].suggested_icon, "food")
        self.assertEqual(by_desc["UBER TRIP"].suggested_icon, "transport")
        self.assertEqual(by_desc["SUPERMERCADO EXTRA"].suggested_icon, "market")
        self.assertEqual(by_desc["NETFLIX.COM"].date, "2026-08-01")
        self.assertEqual(by_desc["NETFLIX.COM"].suggested_icon, "subscriptions")

    def test_year_wraps_across_december(self):
        text = """
        Fatura de janeiro 2027
        20 DEZ IFOOD R$ 30,00
        03 JAN UBER R$ 12,00
        """
        parsed = parse_statement_text(text, 2027, 1)
        dates = {item.description: item.date for item in parsed.items if item.kind == "expense"}
        self.assertEqual(dates["IFOOD"], "2026-12-20")
        self.assertEqual(dates["UBER"], "2027-01-03")

    def test_itau_numeric_dates(self):
        parsed = parse_statement_text(ITAU_TEXT, 2026, 7)
        self.assertEqual(parsed.issuer, "itau")
        expenses = [item for item in parsed.items if item.kind == "expense"]
        self.assertEqual(len(expenses), 3)
        payments = [item for item in parsed.items if item.kind == "payment"]
        self.assertEqual(len(payments), 1)
        self.assertEqual(payments[0].amount, 1200)

    def test_suggest_icon_keywords(self):
        self.assertEqual(suggest_icon("IFOOD *IFOOD"), "food")
        self.assertEqual(suggest_icon("Posto Shell Centro"), "car")
        self.assertEqual(suggest_icon("Farmácia Droga Raia"), "health")
        self.assertEqual(suggest_icon("Loja desconhecida"), "other")

    def test_nubank_checking_ofx(self):
        ofx = b"""OFXHEADER:100
<OFX>
<STMTTRN>
<TRNTYPE>CREDIT</TRNTYPE>
<DTPOSTED>20260901000000[-3:BRT]</DTPOSTED>
<TRNAMT>2000.00</TRNAMT>
<MEMO>Transferencia Recebida - EMPRESA EXEMPLO LTDA</MEMO>
</STMTTRN>
<STMTTRN>
<TRNTYPE>DEBIT</TRNTYPE>
<DTPOSTED>20260901000000[-3:BRT]</DTPOSTED>
<TRNAMT>-2000.00</TRNAMT>
<MEMO>Aplicacao RDB</MEMO>
</STMTTRN>
<STMTTRN>
<TRNTYPE>DEBIT</TRNTYPE>
<DTPOSTED>20260901000000[-3:BRT]</DTPOSTED>
<TRNAMT>-43.46</TRNAMT>
<MEMO>Compra no debito - SAO ROQUE</MEMO>
</STMTTRN>
</OFX>
"""
        parsed = parse_statement_bytes(ofx, 2026, 9)
        kinds = [(item.kind, item.amount) for item in parsed.items]
        self.assertEqual(kinds, [("income", 2000.0), ("transfer", 2000.0), ("expense", 43.46)])


if __name__ == "__main__":
    unittest.main()
