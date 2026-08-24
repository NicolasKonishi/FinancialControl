-- Family members with monthly salary used for budget forecasts.
CREATE TABLE IF NOT EXISTS members (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    name           TEXT    NOT NULL,
    monthly_salary REAL    NOT NULL DEFAULT 0 CHECK (monthly_salary >= 0),
    created_at     TEXT    NOT NULL
);

-- Category icons (food, market, transport, home, health, leisure, salary, freelance, education, other).
ALTER TABLE categories ADD COLUMN icon TEXT NOT NULL DEFAULT 'other';

-- Optional link from a transaction to a family member.
ALTER TABLE transactions ADD COLUMN member_id INTEGER REFERENCES members(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_transactions_member_id ON transactions (member_id);

-- Default categories for a household budget app (skip if name already exists).
INSERT INTO categories (name, description, icon, created_at)
SELECT 'Comida', 'Restaurantes e delivery', 'food', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE NOT EXISTS (SELECT 1 FROM categories WHERE name = 'Comida');

INSERT INTO categories (name, description, icon, created_at)
SELECT 'Mercado', 'Supermercado e feira', 'market', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE NOT EXISTS (SELECT 1 FROM categories WHERE name = 'Mercado');

INSERT INTO categories (name, description, icon, created_at)
SELECT 'Transporte', 'Uber, ônibus, combustível', 'transport', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE NOT EXISTS (SELECT 1 FROM categories WHERE name = 'Transporte');

INSERT INTO categories (name, description, icon, created_at)
SELECT 'Casa', 'Aluguel, contas, manutenção', 'home', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE NOT EXISTS (SELECT 1 FROM categories WHERE name = 'Casa');

INSERT INTO categories (name, description, icon, created_at)
SELECT 'Saúde', 'Farmácia e consultas', 'health', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE NOT EXISTS (SELECT 1 FROM categories WHERE name = 'Saúde');

INSERT INTO categories (name, description, icon, created_at)
SELECT 'Lazer', 'Streaming, passeios', 'leisure', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE NOT EXISTS (SELECT 1 FROM categories WHERE name = 'Lazer');

INSERT INTO categories (name, description, icon, created_at)
SELECT 'Salário', 'Salário mensal', 'salary', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE NOT EXISTS (SELECT 1 FROM categories WHERE name = 'Salário');

INSERT INTO categories (name, description, icon, created_at)
SELECT 'Freelancer', 'Trabalhos extras', 'freelance', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE NOT EXISTS (SELECT 1 FROM categories WHERE name = 'Freelancer');
