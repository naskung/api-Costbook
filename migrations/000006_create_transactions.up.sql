-- รายรับ-รายจ่าย
-- type: income = รายรับ, expense = รายจ่าย / category = รหัสหมวด เช่น sales, material (Flutter แปลเป็นชื่อหมวด)
-- invoice_id มีค่า = รายรับจากใบกำกับ แก้ / ลบตรง ๆ ไม่ได้ ต้องยกเลิกใบกำกับ
CREATE TABLE IF NOT EXISTS transactions (
    id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    type       TEXT          NOT NULL CHECK (type IN ('income', 'expense')),
    category   TEXT          NOT NULL,
    amount     NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    date       DATE          NOT NULL,
    note       TEXT          NOT NULL DEFAULT '',
    invoice_id UUID          UNIQUE REFERENCES invoices (id),
    created_at TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS transactions_date_idx ON transactions (date);
