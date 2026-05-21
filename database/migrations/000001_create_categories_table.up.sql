CREATE TABLE IF NOT EXISTS categories (
    id SERIAL PRIMARY KEY,
    parent_id BIGINT NULL, 
    name VARCHAR(100) NOT NULL,
    icon VARCHAR(255) NOT NULL,
    status BOOLEAN DEFAULT TRUE,
    slug varchar(120) UNIQUE NULL,
    description TEXT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX idx_categories_status ON categories(status);
CREATE INDEX idx_categories_slug ON categories(slug);