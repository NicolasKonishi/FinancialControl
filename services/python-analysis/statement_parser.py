from __future__ import annotations

import csv
import io
import re
from dataclasses import dataclass
from datetime import date
from typing import Iterable

MONTHS: dict[str, int] = {
    "jan": 1,
    "janeiro": 1,
    "fev": 2,
    "feb": 2,
    "fevereiro": 2,
    "mar": 3,
    "marco": 3,
    "março": 3,
    "abr": 4,
    "apr": 4,
    "abril": 4,
    "mai": 5,
    "may": 5,
    "maio": 5,
    "jun": 6,
    "junho": 6,
    "jul": 7,
    "julho": 7,
    "ago": 8,
    "aug": 8,
    "agosto": 8,
    "set": 9,
    "sep": 9,
    "sept": 9,
    "setembro": 9,
    "out": 10,
    "oct": 10,
    "outubro": 10,
    "nov": 11,
    "novembro": 11,
    "dez": 12,
    "dec": 12,
    "dezembro": 12,
}

PAYMENT_RE = re.compile(
    r"\b("
    r"pagamento\s+recebido|pagamento\s+efetuado|pagamento\s+on[\s\-]?line|"
    r"pagamento\s+da\s+fatura|pagto\.?\s+efetuado|pagto\.?\s+fatura|"
    r"credit\s+payment|invoice\s+payment|pagamento\s+fatura"
    r")\b",
    re.I,
)
REFUND_RE = re.compile(r"\b(estorno|cancelamento|devolucao|devolução|refund)\b", re.I)
SKIP_DESC_RE = re.compile(
    r"("
    r"^total\b|^saldo\b|^limite\b|^vencimento\b|^fatura\s+atual|"
    r"pagamento\s+m[ií]nimo|cr[eé]dito\s+dispon[ií]vel|"
    r"lan[cç]amentos?\s+futuros|compras\s+parceladas|^transa[cç][oõ]es?\b|"
    r"^nu\b|^nubank\b|^itau\b|^ita[uú]\b|^extrato\b|^periodo\b|^per[ií]odo\b|"
    r"cnpj|cpf\s*:|p[aá]gina\s+\d|www\.|http"
    r")",
    re.I,
)

BR_AMOUNT_RE = re.compile(
    r"(?<![\d.,])(-?\s*R\$\s*)?(-?\d{1,3}(?:\.\d{3})*,\d{2}|-?\d+,\d{2})(?![\d,])"
)
US_AMOUNT_RE = re.compile(r"(?<![\d.,])(-?\s*R\$\s*)?(-?\d{1,3}(?:,\d{3})*\.\d{2}|-?\d+\.\d{2})(?![\d.])")

NAMED_DATE_RE = re.compile(
    r"\b(\d{1,2})\s+([A-Za-zÀ-ÿ]{3,9})\b(?:\s+(\d{4}))?",
    re.I,
)
NUMERIC_DATE_RE = re.compile(r"\b(\d{1,2})[/-](\d{1,2})(?:[/-](\d{2,4}))?\b")
ISO_DATE_RE = re.compile(r"\b(\d{4})-(\d{2})-(\d{2})\b")

FATURA_MONTH_RE = re.compile(
    r"fatura\s+de\s+([A-Za-zÀ-ÿ]+)\s+(\d{4})",
    re.I,
)
PERIOD_RE = re.compile(
    r"(?:transa[cç][oõ]es?|per[ií]odo|de)\s+"
    r"(\d{1,2})\s*([A-Za-zÀ-ÿ]{3,9})\s*(?:a|at[eé]|-|–)\s*"
    r"(\d{1,2})\s*([A-Za-zÀ-ÿ]{3,9})(?:\s+(\d{4}))?",
    re.I,
)
DUE_RE = re.compile(
    r"vencimento\s+(\d{1,2})\s*([A-Za-zÀ-ÿ]{3,9})\s+(\d{4})",
    re.I,
)

ICON_KEYWORDS: list[tuple[str, tuple[str, ...]]] = [
    (
        "subscriptions",
        (
            "netflix",
            "spotify",
            "disney",
            "prime video",
            "amazon prime",
            "youtube",
            "apple.com/bill",
            "paramount",
            "hbo",
            "globoplay",
            "openai",
            "chatgpt",
            "cursor",
            "icloud",
            "google one",
            "microsoft 365",
            "xbox game pass",
        ),
    ),
    (
        "food",
        (
            "ifood",
            "rappi",
            "uber eats",
            "ubereats",
            "restaurante",
            "mcdonald",
            "burger king",
            "bk ",
            "pizza",
            "outback",
            "habib",
            "subway",
            "delivery",
            "raial",
            "madero",
            "china in box",
            "giraffas",
        ),
    ),
    (
        "market",
        (
            "carrefour",
            "extra",
            "assai",
            "assaí",
            "pao de acucar",
            "pão de açúcar",
            "supermercado",
            "hipermercado",
            "atacadao",
            "atacadão",
            "sams club",
            "savegnago",
            "obah",
            "hortifruti",
            "feira",
            "mercado livre gas",
        ),
    ),
    (
        "cafe",
        ("starbucks", "cafeteria", "padaria", "bakery", "cafeteira", "coffee"),
    ),
    (
        "transport",
        (
            "uber",
            "99app",
            "99 pop",
            "99pay",
            "cabify",
            "metro",
            "metrô",
            "onibus",
            "ônibus",
            "bilhete unico",
            "bilhete único",
            "estacionamento",
            "parking",
            "sem parar",
            "conectcar",
            "veloe",
        ),
    ),
    (
        "car",
        (
            "shell",
            "ipiranga",
            "petrobras",
            "auto posto",
            "posto ",
            "combustivel",
            "combustível",
            "br manteiga",
            "ale combustiveis",
        ),
    ),
    (
        "health",
        (
            "farmacia",
            "farmácia",
            "drogaria",
            "droga raia",
            "pague menos",
            "panvel",
            "hospital",
            "laboratorio",
            "laboratório",
            "consulta",
            "unimed",
            "sulamerica",
        ),
    ),
    (
        "pets",
        ("petlove", "petz", "cobasi", "pet shop"),
    ),
    (
        "shopping",
        (
            "amazon",
            "mercadolivre",
            "mercado livre",
            "shopee",
            "magazine luiza",
            "magalu",
            "americanas",
            "casas bahia",
            "shein",
            "aliexpress",
        ),
    ),
    (
        "clothing",
        ("renner", "c&a", "cea ", "zara", "nike", "adidas", "hering", "riachuelo"),
    ),
    (
        "home",
        ("ikea", "leroy", "telhanorte", "aluguel", "condominio", "condomínio", "mobly"),
    ),
    (
        "utilities",
        (
            "enel",
            "cpfl",
            "sabesp",
            "comgas",
            "vivo fibra",
            "claro net",
            "internet",
            "light ",
            "cemig",
        ),
    ),
    (
        "phone",
        ("vivo", "claro", " tim", "tim ", "oi celular"),
    ),
    (
        "travel",
        (
            "booking",
            "airbnb",
            "decolar",
            "latam",
            "gol linhas",
            "azul linhas",
            "hotel",
            "passagem",
        ),
    ),
    (
        "leisure",
        ("cinema", "ingresso", "steam", "playstation", "show ", "bar ", "balada"),
    ),
    (
        "education",
        ("udemy", "alura", "coursera", "escola", "faculdade", "livro"),
    ),
    (
        "gift",
        ("presente", "floricultura"),
    ),
]


class StatementError(Exception):
    def __init__(self, code: str, message: str):
        super().__init__(message)
        self.code = code
        self.message = message


@dataclass
class ParsedItem:
    date: str
    description: str
    amount: float
    kind: str
    suggested_icon: str


@dataclass
class ParsedStatement:
    issuer: str
    period_start: str | None
    period_end: str | None
    items: list[ParsedItem]


def parse_statement_bytes(
    data: bytes,
    reference_year: int | None = None,
    reference_month: int | None = None,
) -> ParsedStatement:
    if not data:
        raise StatementError("no_text", "empty file")

    today = date.today()
    year = reference_year or today.year
    month = reference_month or today.month

    if _looks_like_ofx(data):
        text = _decode_text(data)
        items = _parse_ofx(text, year, month)
        if items:
            return ParsedStatement(
                issuer=_detect_issuer(text),
                period_start=items[0].date,
                period_end=items[-1].date,
                items=items,
            )

    if _looks_like_csv(data):
        text = _decode_text(data)
        items = _parse_csv(text, year, month)
        if items:
            return ParsedStatement(
                issuer=_detect_issuer(text),
                period_start=items[0].date,
                period_end=items[-1].date,
                items=items,
            )

    text = extract_pdf_text(data)
    return parse_statement_text(text, year, month)


def extract_pdf_text(data: bytes) -> str:
    try:
        from pypdf import PdfReader
        from pypdf.errors import FileNotDecryptedError, PdfReadError
    except ImportError as exc:
        raise StatementError("no_text", "pypdf is not installed") from exc

    try:
        reader = PdfReader(io.BytesIO(data), strict=False)
    except PdfReadError as exc:
        raise StatementError("no_text", "invalid pdf") from exc

    if reader.is_encrypted:
        try:
            unlocked = reader.decrypt("")
        except (FileNotDecryptedError, NotImplementedError, PdfReadError) as exc:
            raise StatementError("encrypted", "password protected pdf") from exc
        if unlocked == 0:
            raise StatementError("encrypted", "password protected pdf")

    pages: list[str] = []
    for page in reader.pages:
        try:
            pages.append(page.extract_text() or "")
        except Exception:
            pages.append("")
    text = "\n".join(pages).strip()
    if not text:
        raise StatementError("no_text", "no extractable text")
    return text


def parse_statement_text(text: str, reference_year: int, reference_month: int) -> ParsedStatement:
    closing_year, closing_month = _closing_from_text(text, reference_year, reference_month)
    period_start, period_end = _period_from_text(text, closing_year, closing_month)

    items: list[ParsedItem] = []
    seen: set[tuple[str, str, int]] = set()
    for raw_line in _candidate_lines(text):
        parsed = _parse_line(raw_line, closing_year, closing_month, period_start, period_end)
        if parsed is None:
            continue
        key = (parsed.date, _norm_desc(parsed.description), int(round(parsed.amount * 100)))
        if key in seen:
            continue
        seen.add(key)
        items.append(parsed)

    items.sort(key=lambda item: (item.date, item.description))
    items = _reclassify_account_statement(items)
    return ParsedStatement(
        issuer=_detect_issuer(text),
        period_start=period_start.isoformat() if period_start else (items[0].date if items else None),
        period_end=period_end.isoformat() if period_end else (items[-1].date if items else None),
        items=items,
    )


def suggest_icon(description: str) -> str:
    hay = _norm_desc(description)
    for icon, keywords in ICON_KEYWORDS:
        for keyword in keywords:
            if _norm_desc(keyword) in hay:
                return icon
    return "other"


def _candidate_lines(text: str) -> Iterable[str]:
    compact = re.sub(r"[ \t]+", " ", text.replace("\u00a0", " "))
    yielded: set[str] = set()
    for raw in compact.splitlines():
        line = raw.strip(" -•\t")
        if len(line) < 8:
            continue
        if line not in yielded:
            yielded.add(line)
            yield line
    # Some issuers break amount onto the next line; also scan flattened text.
    flat = re.sub(r"\s+", " ", compact)
    for match in re.finditer(
        r"(\d{1,2}\s+[A-Za-zÀ-ÿ]{3,9}(?:\s+\d{4})?|\d{1,2}[/-]\d{1,2}(?:[/-]\d{2,4})?)\s+"
        r"(.{3,80}?)\s+"
        r"(?:R\$\s*)?(-?\d{1,3}(?:\.\d{3})*,\d{2}|-?\d+,\d{2})",
        flat,
        flags=re.I,
    ):
        line = " ".join(part.strip() for part in match.groups() if part)
        if line not in yielded:
            yielded.add(line)
            yield line


def _parse_line(
    line: str,
    closing_year: int,
    closing_month: int,
    period_start: date | None,
    period_end: date | None,
) -> ParsedItem | None:
    amount, amount_span = _extract_amount(line)
    if amount is None or amount_span is None:
        return None
    if amount == 0 or abs(amount) > 1_000_000:
        return None

    before = line[: amount_span[0]].strip(" -–·|")
    after = line[amount_span[1] :].strip()
    if after and not re.fullmatch(r"(D|C|CR|DB|\*)", after, re.I):
        return None

    parsed_date, date_span = _extract_date(before, closing_year, closing_month, period_start, period_end)
    if parsed_date is None or date_span is None:
        return None

    description = before[date_span[1] :].strip(" -–·|:")
    description = re.sub(r"\s+", " ", description)
    description = description.strip("* ")
    if len(description) < 2 or SKIP_DESC_RE.search(description):
        return None
    if re.fullmatch(r"[\d./\-]+", description):
        return None

    kind = "expense"
    if PAYMENT_RE.search(description) or amount < 0:
        kind = "payment"
    if REFUND_RE.search(description):
        kind = "refund"

    return ParsedItem(
        date=parsed_date.isoformat(),
        description=_clean_merchant(description),
        amount=round(abs(amount), 2),
        kind=kind,
        suggested_icon=suggest_icon(description),
    )


def _extract_amount(line: str) -> tuple[float | None, tuple[int, int] | None]:
    matches = list(BR_AMOUNT_RE.finditer(line))
    us_style = False
    if not matches:
        matches = list(US_AMOUNT_RE.finditer(line))
        us_style = True
        if not matches:
            return None, None
    match = matches[-1]
    raw = match.group(0)
    value = _parse_us_amount(raw) if us_style else _parse_br_amount(raw)
    if value is None:
        return None, None
    prefix = line[: match.start()].rstrip()
    if prefix.endswith("-"):
        value = -abs(value)
    return value, match.span()


def _parse_br_amount(raw: str) -> float | None:
    s = raw.replace("R$", "").replace(" ", "").replace("\u00a0", "")
    negative = s.startswith("-") or s.endswith("-")
    s = s.strip("-")
    if not re.fullmatch(r"\d{1,3}(?:\.\d{3})*,\d{2}|\d+,\d{2}", s):
        return None
    try:
        value = float(s.replace(".", "").replace(",", "."))
    except ValueError:
        return None
    return -value if negative else value


def _parse_us_amount(raw: str) -> float | None:
    s = raw.replace("R$", "").replace(" ", "").replace("\u00a0", "")
    negative = s.startswith("-") or s.endswith("-")
    s = s.strip("-")
    if not re.fullmatch(r"\d{1,3}(?:,\d{3})*\.\d{2}|\d+\.\d{2}", s):
        return None
    try:
        value = float(s.replace(",", ""))
    except ValueError:
        return None
    return -value if negative else value


def _extract_date(
    text: str,
    closing_year: int,
    closing_month: int,
    period_start: date | None,
    period_end: date | None,
) -> tuple[date | None, tuple[int, int] | None]:
    iso = ISO_DATE_RE.search(text)
    if iso:
        try:
            parsed = date(int(iso.group(1)), int(iso.group(2)), int(iso.group(3)))
        except ValueError:
            parsed = None
        if parsed:
            return parsed, iso.span()

    numeric = NUMERIC_DATE_RE.search(text)
    if numeric:
        day = int(numeric.group(1))
        month = int(numeric.group(2))
        year_raw = numeric.group(3)
        year = _coerce_year(year_raw, month, closing_year, closing_month)
        try:
            parsed = date(year, month, day)
        except ValueError:
            parsed = None
        if parsed:
            return parsed, numeric.span()

    named = NAMED_DATE_RE.search(text)
    if named:
        month = _month_num(named.group(2))
        if month:
            day = int(named.group(1))
            year = _coerce_year(named.group(3), month, closing_year, closing_month)
            try:
                parsed = date(year, month, day)
            except ValueError:
                parsed = None
            if parsed:
                if period_start and period_end and not (period_start <= parsed <= period_end):
                    shifted = _shift_into_period(parsed, period_start, period_end)
                    if shifted:
                        parsed = shifted
                return parsed, named.span()
    return None, None


def _coerce_year(raw: str | None, month: int, closing_year: int, closing_month: int) -> int:
    if raw:
        year = int(raw)
        if year < 100:
            year += 2000
        return year
    if month > closing_month:
        return closing_year - 1
    return closing_year


def _shift_into_period(parsed: date, start: date, end: date) -> date | None:
    for delta in (0, -1, 1):
        try:
            candidate = parsed.replace(year=parsed.year + delta)
        except ValueError:
            continue
        if start <= candidate <= end:
            return candidate
    return None


def _closing_from_text(text: str, fallback_year: int, fallback_month: int) -> tuple[int, int]:
    fatura = FATURA_MONTH_RE.search(text)
    if fatura:
        month = _month_num(fatura.group(1))
        if month:
            return int(fatura.group(2)), month
    due = DUE_RE.search(text)
    if due:
        month = _month_num(due.group(2))
        if month:
            return int(due.group(3)), month
    return fallback_year, fallback_month


def _period_from_text(text: str, closing_year: int, closing_month: int) -> tuple[date | None, date | None]:
    match = PERIOD_RE.search(text)
    if not match:
        return None, None
    start_month = _month_num(match.group(2))
    end_month = _month_num(match.group(4))
    if not start_month or not end_month:
        return None, None
    end_year = int(match.group(5)) if match.group(5) else closing_year
    start_year = end_year if start_month <= end_month else end_year - 1
    try:
        start = date(start_year, start_month, int(match.group(1)))
        end = date(end_year, end_month, int(match.group(3)))
    except ValueError:
        return None, None
    return start, end


def _month_num(raw: str | None) -> int | None:
    if not raw:
        return None
    key = (
        raw.strip()
        .lower()
        .replace("ç", "c")
        .replace("ã", "a")
        .replace("á", "a")
        .replace("é", "e")
        .replace("ê", "e")
        .replace("í", "i")
        .replace("ó", "o")
        .replace("ô", "o")
        .replace("ú", "u")
    )
    return MONTHS.get(key[:3]) or MONTHS.get(key)


def _detect_issuer(text: str) -> str:
    hay = _norm_desc(text[:4000])
    head = _norm_desc(text[:80])
    if head == "nu" or head.startswith("nu "):
        return "nubank"
    checks = (
        ("nubank", ("nubank", "nu pagamentos")),
        ("itau", ("banco itau", "itaú", "itau unibanco", "banco itaú")),
        ("inter", ("banco inter", "inter pag")),
        ("c6", ("c6 bank", "banco c6")),
        ("santander", ("santander",)),
        ("bradesco", ("bradesco",)),
        ("bb", ("banco do brasil",)),
        ("picpay", ("picpay",)),
        ("mercado-pago", ("mercado pago", "mercadopago")),
    )
    for issuer, needles in checks:
        if any(needle in hay for needle in needles):
            return issuer
    return "unknown"


def _clean_merchant(description: str) -> str:
    cleaned = re.sub(r"\s+", " ", description).strip(" -*")
    cleaned = re.sub(r"\b(sao paulo|são paulo|brasil|brazil|br)\b", "", cleaned, flags=re.I)
    cleaned = re.sub(r"\s+", " ", cleaned).strip(" -*")
    if "*" in cleaned:
        left, right = cleaned.split("*", 1)
        # "IFOOD *IFOOD" -> "IFOOD"
        if _norm_desc(left) and _norm_desc(left) in _norm_desc(right):
            cleaned = right.strip()
        elif _norm_desc(right) and _norm_desc(right) in _norm_desc(left):
            cleaned = left.strip()
    return cleaned[:80].strip() or description[:80].strip()


def _norm_desc(value: str) -> str:
    lowered = value.lower()
    out: list[str] = []
    prev_space = False
    for char in lowered:
        if char.isalnum():
            out.append(char)
            prev_space = False
        elif not prev_space:
            out.append(" ")
            prev_space = True
    return "".join(out).strip()


def _looks_like_csv(data: bytes) -> bool:
    if data[:5] == b"%PDF-" or _looks_like_ofx(data):
        return False
    sample = _decode_text(data[:4000])
    if "," not in sample and ";" not in sample:
        return False
    header = sample.splitlines()[0].lower() if sample.splitlines() else ""
    return any(token in header for token in ("data", "date", "descr", "title", "valor", "amount"))


def _decode_text(data: bytes) -> str:
    for encoding in ("utf-8-sig", "utf-8", "latin-1"):
        try:
            return data.decode(encoding)
        except UnicodeDecodeError:
            continue
    return data.decode("utf-8", errors="replace")


def _parse_csv(text: str, reference_year: int, reference_month: int) -> list[ParsedItem]:
    sample = text[:4000]
    try:
        dialect = csv.Sniffer().sniff(sample, delimiters=",;")
    except csv.Error:
        dialect = csv.excel
        dialect.delimiter = ";" if sample.count(";") > sample.count(",") else ","

    reader = csv.DictReader(io.StringIO(text), dialect=dialect)
    if not reader.fieldnames:
        return []
    fields = {name.lower().strip(): name for name in reader.fieldnames if name}
    date_key = _first_field(fields, ("data", "date", "dt"))
    desc_key = _first_field(fields, ("descrição", "descricao", "title", "historico", "histórico", "memo", "description"))
    amount_key = _first_field(fields, ("valor", "amount", "value", "vlr"))
    if not date_key or not desc_key or not amount_key:
        return []

    items: list[ParsedItem] = []
    for row in reader:
        raw_date = (row.get(date_key) or "").strip()
        raw_desc = (row.get(desc_key) or "").strip()
        raw_amount = (row.get(amount_key) or "").strip()
        parsed_date, _ = _extract_date(raw_date, reference_year, reference_month, None, None)
        amount, _ = _extract_amount(raw_amount if re.search(r"\d", raw_amount) else f"0,00 {raw_amount}")
        if parsed_date is None:
            continue
        if amount is None:
            amount, _ = _extract_amount(raw_amount.replace(".", ",", 1) if raw_amount.count(".") == 1 else raw_amount)
        if amount is None or not raw_desc:
            continue
        kind = "expense"
        if PAYMENT_RE.search(raw_desc) or amount < 0:
            kind = "payment"
        if REFUND_RE.search(raw_desc):
            kind = "refund"
        items.append(
            ParsedItem(
                date=parsed_date.isoformat(),
                description=_clean_merchant(raw_desc),
                amount=round(abs(amount), 2),
                kind=kind,
                suggested_icon=suggest_icon(raw_desc),
            )
        )
    return _reclassify_account_statement(items)


def _reclassify_account_statement(items: list[ParsedItem]) -> list[ParsedItem]:
    """Checking-account PDFs often list purchases as negative amounts."""
    unlabeled = [
        item
        for item in items
        if item.kind == "payment" and not PAYMENT_RE.search(item.description)
    ]
    labeled = [item for item in items if PAYMENT_RE.search(item.description)]
    if len(items) >= 3 and len(unlabeled) >= len(items) * 0.7 and len(labeled) <= 2:
        for item in unlabeled:
            item.kind = "expense"
    return items


def _first_field(fields: dict[str, str], names: tuple[str, ...]) -> str | None:
    for name in names:
        if name in fields:
            return fields[name]
    for key, original in fields.items():
        if any(name in key for name in names):
            return original
    return None


def _looks_like_ofx(data: bytes) -> bool:
    sample = _decode_text(data[:1200]).upper()
    return "OFXHEADER" in sample or "<OFX>" in sample or "<STMTTRN>" in sample


def _parse_ofx(text: str, reference_year: int, reference_month: int) -> list[ParsedItem]:
    items: list[ParsedItem] = []
    for block in re.findall(r"<STMTTRN>(.*?)</STMTTRN>", text, flags=re.I | re.S):
        tags = {key.upper(): value.strip() for key, value in re.findall(r"<([A-Z0-9.]+)>([^<\r\n]*)", block, flags=re.I)}
        raw_amount = tags.get("TRNAMT", "")
        amount, _ = _extract_amount(raw_amount)
        if amount is None:
            continue
        if raw_amount.strip().startswith("-") and amount > 0:
            amount = -amount
        posted = re.sub(r"\[.*", "", tags.get("DTPOSTED", "")).strip()
        parsed_date = None
        if re.fullmatch(r"\d{8}", posted[:8] or ""):
            try:
                parsed_date = date(int(posted[:4]), int(posted[4:6]), int(posted[6:8]))
            except ValueError:
                parsed_date = None
        if parsed_date is None:
            parsed_date, _ = _extract_date(posted, reference_year, reference_month, None, None)
        if parsed_date is None:
            continue
        desc = (tags.get("MEMO") or tags.get("NAME") or "").strip()
        if not desc:
            continue
        kind = _classify_kind(desc, amount)
        items.append(
            ParsedItem(
                date=parsed_date.isoformat(),
                description=_clean_merchant(desc),
                amount=round(abs(amount), 2),
                kind=kind,
                suggested_icon="salary" if kind == "income" else suggest_icon(desc),
            )
        )
    return items


def _classify_kind(description: str, amount: float) -> str:
    if REFUND_RE.search(description):
        return "refund"
    if PAYMENT_RE.search(description):
        return "payment"
    hay = (
        description.lower()
        .replace("ç", "c")
        .replace("ã", "a")
        .replace("á", "a")
        .replace("é", "e")
        .replace("ê", "e")
        .replace("í", "i")
        .replace("ó", "o")
        .replace("ô", "o")
        .replace("ú", "u")
    )
    if "aplicacao rdb" in hay or "resgate rdb" in hay:
        return "transfer"
    if amount < 0:
        return "expense"
    if "recebida" in hay or "recebido" in hay or "rendimento" in hay or "salario" in hay:
        return "income"
    return "expense"
