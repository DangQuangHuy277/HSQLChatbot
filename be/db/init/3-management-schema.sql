CREATE TABLE IF NOT EXISTS user_account
(
    id       SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL, -- Unique login identifier (e.g., student code or email)
    password VARCHAR(255)       NOT NULL  -- Hashed password (never store plain text)
);