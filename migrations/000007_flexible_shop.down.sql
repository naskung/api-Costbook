-- ย้อนกลับ: เอาความสามารถของร้านหลายรูปแบบออก (ข้อมูลใน column ใหม่จะหายไป)
ALTER TABLE invoice_numbers DROP CONSTRAINT IF EXISTS invoice_numbers_pkey;
DELETE FROM invoice_numbers WHERE doc_type <> 'tax_invoice';
ALTER TABLE invoice_numbers ADD PRIMARY KEY (year);
ALTER TABLE invoice_numbers DROP COLUMN IF EXISTS doc_type;

ALTER TABLE invoice_items DROP COLUMN IF EXISTS stock_cut;
ALTER TABLE invoices
    DROP COLUMN IF EXISTS doc_type,
    DROP COLUMN IF EXISTS prices_include_vat,
    DROP COLUMN IF EXISTS items_total,
    DROP COLUMN IF EXISTS discount;

ALTER TABLE stock_movements
    DROP COLUMN IF EXISTS unit_cost,
    DROP COLUMN IF EXISTS supplier;

DROP TABLE IF EXISTS cost_presets;
ALTER TABLE product_costs
    DROP COLUMN IF EXISTS kind,
    DROP COLUMN IF EXISTS quantity,
    DROP COLUMN IF EXISTS unit,
    DROP COLUMN IF EXISTS unit_price,
    DROP COLUMN IF EXISTS percent;

DROP TABLE IF EXISTS product_fields;
DROP INDEX IF EXISTS products_code_key;
ALTER TABLE products
    DROP COLUMN IF EXISTS code,
    DROP COLUMN IF EXISTS category,
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS track_stock,
    DROP COLUMN IF EXISTS custom_fields;

ALTER TABLE company
    DROP COLUMN IF EXISTS vat_registered,
    DROP COLUMN IF EXISTS prices_include_vat,
    DROP COLUMN IF EXISTS use_cost_page;
