"""
Financial analysis service.

Go owns business CRUD. This service only computes statistics from
transactions that Go sends over HTTP/JSON.
"""

from __future__ import annotations

from collections import defaultdict
from typing import Literal

from fastapi import FastAPI, HTTPException, Query, Request
from pydantic import BaseModel, Field

from statement_parser import ParsedStatement, StatementError, parse_statement_bytes

app = FastAPI(title="Financial Analysis", version="0.1.0")


class Category(BaseModel):
    id: int
    name: str
    description: str | None = None


class Transaction(BaseModel):
    id: int | None = None
    category_id: int
    type: Literal["income", "expense"]
    description: str = ""
    amount: float = Field(gt=0)
    date: str | None = None


class MonthlyAnalysisRequest(BaseModel):
    year: int
    month: int = Field(ge=1, le=12)
    transactions: list[Transaction]
    categories: list[Category] = []


class CategoryBreakdown(BaseModel):
    category_id: int
    category_name: str
    total: float


class MonthlyAnalysisResponse(BaseModel):
    year: int
    month: int
    total_income: float
    total_expense: float
    balance: float
    by_category: list[CategoryBreakdown]
    transaction_count: int


@app.get("/health")
def health() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/analysis/monthly", response_model=MonthlyAnalysisResponse)
def monthly_analysis(payload: MonthlyAnalysisRequest) -> MonthlyAnalysisResponse:
    names = {c.id: c.name for c in payload.categories}
    expense_by_category: dict[int, float] = defaultdict(float)

    total_income = 0.0
    total_expense = 0.0

    for tx in payload.transactions:
        if tx.type == "income":
            total_income += tx.amount
        else:
            total_expense += tx.amount
            expense_by_category[tx.category_id] += tx.amount

    by_category = [
        CategoryBreakdown(
            category_id=category_id,
            category_name=names.get(category_id, f"Category {category_id}"),
            total=round(total, 2),
        )
        for category_id, total in sorted(expense_by_category.items(), key=lambda item: item[1], reverse=True)
    ]

    return MonthlyAnalysisResponse(
        year=payload.year,
        month=payload.month,
        total_income=round(total_income, 2),
        total_expense=round(total_expense, 2),
        balance=round(total_income - total_expense, 2),
        by_category=by_category,
        transaction_count=len(payload.transactions),
    )


class StatementItem(BaseModel):
    date: str
    description: str
    amount: float
    kind: str
    suggested_icon: str


class StatementParseResponse(BaseModel):
    issuer: str
    period_start: str | None = None
    period_end: str | None = None
    items: list[StatementItem]


@app.post("/statements/parse", response_model=StatementParseResponse)
async def parse_statement(
    request: Request,
    year: int | None = Query(default=None, ge=2000, le=2100),
    month: int | None = Query(default=None, ge=1, le=12),
) -> StatementParseResponse:
    data = await request.body()
    if len(data) > 8 * 1024 * 1024:
        raise HTTPException(status_code=413, detail="file too large")
    if not data:
        raise HTTPException(status_code=400, detail="empty")
    try:
        parsed: ParsedStatement = parse_statement_bytes(data, year, month)
    except StatementError as exc:
        raise HTTPException(status_code=400, detail=exc.code) from exc
    return StatementParseResponse(
        issuer=parsed.issuer,
        period_start=parsed.period_start,
        period_end=parsed.period_end,
        items=[
            StatementItem(
                date=item.date,
                description=item.description,
                amount=item.amount,
                kind=item.kind,
                suggested_icon=item.suggested_icon,
            )
            for item in parsed.items
        ],
    )
