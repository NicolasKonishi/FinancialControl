"""
Financial analysis service.

Go owns business CRUD. This service only computes statistics from
transactions that Go sends over HTTP/JSON.
"""

from __future__ import annotations

from collections import defaultdict
from typing import Literal

from fastapi import FastAPI
from pydantic import BaseModel, Field

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
