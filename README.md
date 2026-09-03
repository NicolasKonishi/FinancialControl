# Fluxo (FinancialControl)

Plataforma de controle financeiro familiar, feita como projeto colaborativo de aprendizado e portfólio.

O objetivo é organizar **entradas**, **saídas**, **salários da família** e uma **previsão do mês**, com interface pensada para uso no celular.

---

## Visão geral

```text
                    App web (React)
                   http://localhost:5173
                            |
                            | HTTP/JSON
                            v
                     API Go (backend)
                   http://localhost:8080
                            |
              +-------------+-------------+
              |                           |
              v                           v
         SQLite                      Python
    ./data/fluxo.db            análise financeira
                              http://localhost:8000
```

Fluxo típico:

1. A família cadastra pessoas e salários mensais.
2. Lança gastos (mercado, comida, transporte…) e extras (freelancer).
   Também pode mandar o **PDF da fatura do cartão** para o app marcar o que faltou.
3. O app mostra o que entrou, o que saiu e quanto ainda dá para gastar no mês.

---

## Responsabilidades por tecnologia

### Frontend (`frontend/`) — interface

- Exibe o app **Fluxo** (mobile-first).
- Telas: **Início**, **Gastos**, **Família**.
- Modo claro / escuro.
- Fala **somente** com a API Go (não acessa o banco nem o Python diretamente).

**Não é responsabilidade do frontend:** regras de negócio pesadas, SQL ou algoritmos de análise.

### Backend Go (`cmd/`, `internal/`) — orquestração e regras da API

- Expõe a API HTTP/JSON.
- Valida dados (valor > 0, categoria existe, etc.).
- Persiste tudo no SQLite.
- Calcula a **previsão mensal** (`/forecast/monthly`) com base em:
  - soma dos salários da família;
  - entradas extras do mês (ex.: freelancer);
  - despesas do mês;
  - projeção de ritmo de gasto e “quanto pode gastar por dia”.
- Quando precisa de estatística agregada do serviço Python, monta o payload e chama o Python.

**Não é responsabilidade do Go:** UI visual nem algoritmos analíticos avançados / ML (isso fica no Python).

### Python (`services/python-analysis/`) — análise

- Serviço separado de análise financeira.
- Recebe transações/categorias via HTTP e devolve totais e breakdown.
- Pensado para crescer depois (estatística, ML, recomendações).

**Não é responsabilidade do Python:** CRUD de categorias/lançamentos nem tela do usuário.

### SQLite (`./data/fluxo.db`) — persistência

- Banco em arquivo local (sem Docker).
- Schema controlado por **migrations** versionadas.

---

## Como o produto funciona (para o usuário)

| Área | O que faz |
|------|-----------|
| **Família** | Cadastro de pessoas + salário mensal de cada uma (editar/excluir). |
| **Gastos** | Tabela do mês: entradas e saídas, com categoria e ícone. Importa extrato em PDF do cartão e lança o que ainda não estava marcado. |
| **Início** | Previsão do mês, atalhos de Saída / Entrada / Extra (freelancer). |
| **Categorias** | Vêm pré-carregadas (Comida, Mercado, Transporte, Casa, Saúde, Lazer, Salário, Freelancer), cada uma com ícone. |

A previsão responde, em resumo:

- **Disponível** = salários planejados + extras do mês  
- **Sobrando** = disponível − o que já saiu  
- **Gasto previsto** = ritmo atual projetado até o fim do mês  
- **Por dia** = quanto ainda dá para gastar nos dias restantes  

---

## Estrutura do repositório

```text
FinancialControl/
├── cmd/
│   ├── api/                 # sobe o servidor HTTP
│   └── migrate/             # aplica / reverte migrations
├── internal/
│   ├── config/              # variáveis de ambiente
│   ├── database/            # conexão SQLite
│   ├── server/              # registro de rotas + CORS
│   ├── handlers/            # HTTP (request/response)
│   ├── repository/          # SQL
│   ├── models/              # tipos de domínio / DTOs
│   ├── analysis/            # cliente HTTP → Python
│   └── middleware/          # ex.: CORS
├── migrations/              # SQL versionado (up/down)
├── frontend/                # app React + Vite (UI)
├── services/
│   └── python-analysis/     # FastAPI de análise
├── .env.example
├── go.mod
└── README.md
```

### Camadas do backend Go (por que existem)

| Pacote | Responsabilidade |
|--------|------------------|
| `cmd/*` | Só “liga” o processo (wiring). Quase sem regra de negócio. |
| `handlers` | Traduz HTTP ↔ domínio: status code, JSON, validação básica. |
| `repository` | Único lugar que escreve SQL. |
| `models` | Structs compartilhadas (Category, Transaction, Member…). |
| `server` | Monta as rotas e conecta dependências. |
| `migrations` | Evolui o schema sem SQL solto no código. |

`internal/` é convenção Go: outros módulos **não** devem importar esses pacotes.

---

## Pré-requisitos

- Go 1.22+
- Node.js 20+ (frontend)
- Python 3.11+ (serviço de análise)

---

## Como rodar localmente

Use **3 terminais**.

### 1) Migrations (primeira vez ou após puxar mudanças)

```bash
go run ./cmd/migrate -direction=up
```

Isso cria/atualiza `./data/fluxo.db`.

Rollback (se necessário):

```bash
go run ./cmd/migrate -direction=down -steps=1
```

### 2) API Go

```bash
go run ./cmd/api
```

Ao subir, o terminal imprime as URLs (API, health, frontend, etc.).

Padrões:

- API: `http://localhost:8080`
- SQLite: `./data/fluxo.db`
- Python esperado em: `http://localhost:8000`

Variáveis opcionais (veja `.env.example`):

```bash
HTTP_ADDR=:8080
SQLITE_PATH=./data/fluxo.db
PYTHON_ANALYSIS_URL=http://localhost:8000
FRONTEND_URL=http://localhost:5173
MIGRATIONS_PATH=migrations
```

### 3) Python (análise)

```bash
cd services/python-analysis
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
uvicorn main:app --reload --port 8000
```

> A UI principal de previsão usa o endpoint Go `/forecast/monthly` (não depende do Python para o fluxo diário).  
> O Python entra em `/analysis/monthly` (agregações enviadas pelo Go).

### 4) Frontend

```bash
cd frontend
npm install
npm run dev
```

Abra: **http://localhost:5173**

---

## Principais rotas da API

```text
GET    /health

GET    /categories
POST   /categories
GET    /categories/{id}
PUT    /categories/{id}
DELETE /categories/{id}

GET    /members
POST   /members
PUT    /members/{id}
DELETE /members/{id}

GET    /transactions
POST   /transactions
GET    /transactions/{id}
PUT    /transactions/{id}
DELETE /transactions/{id}

GET    /forecast/monthly?year=2026&month=8
GET    /analysis/monthly?year=2026&month=8

POST   /statements/preview   (multipart: file, wallet_id, member_id, year, month)
POST   /statements/import
```

---

## Divisão do time (projeto colaborativo)

| Pessoa / área | Foco |
|---------------|------|
| Desenvolvedor Go | API, persistência, migrations, contrato com Python, orquestração |
| Desenvolvedor Python | Análise, estatística, futuro ML |
| Frontend | Experiência do usuário (Fluxo), mobile, tema claro/escuro |

Contrato entre serviços: **HTTP + JSON**. Cada lado evolui sem quebrar o outro, desde que o contrato seja respeitado.

---

## Testes

```bash
go test ./...
```

Os testes de handlers usam store em memória (não precisam do arquivo SQLite).

---

## Bibliotecas escolhidas (e por quê)

| Biblioteca | Motivo |
|------------|--------|
| `net/http` | API sem framework — mais didático no início |
| `database/sql` + `modernc.org/sqlite` | SQL padrão + SQLite puro Go (sem CGO) |
| `golang-migrate` | Migrations profissionais (`up` / `down`) |
| React + Vite | UI moderna e rápida para o app mobile |

---

## O que ainda não faz parte do escopo

De propósito, para não complicar cedo demais:

- autenticação / JWT / multi-usuário com login
- PostgreSQL em produção (SQLite cobre bem o estágio atual)
- Redis, Kafka, Kubernetes
- machine learning avançado

Esses itens entram quando houver motivo claro de arquitetura.

---

## Princípio do projeto

```text
Simples → Entendível → Testado → Persistente → Integrado → Distribuído
```

Começamos com uma boa aplicação Go + UI útil para a família.  
Distribuímos e adicionamos tecnologia só quando o problema real pedir.
