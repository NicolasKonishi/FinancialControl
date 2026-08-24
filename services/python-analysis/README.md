# Financial Analysis (Python)

Small FastAPI service used by the Go backend for monthly statistics.

## Run

```bash
cd services/python-analysis
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
uvicorn main:app --reload --port 8000
```

## Contract

`POST /analysis/monthly`

Go sends transactions + categories for a month. Python returns totals and expense breakdown by category.

This service does **not** own persistence. Go remains the source of truth.
