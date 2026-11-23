CREATE TABLE IF NOT EXISTS purchases (
    id SERIAL PRIMARY KEY,
    item_id INTEGER REFERENCES items(id),
    quantity INTEGER NOT NULL CHECK(quantity>0),
    purchased_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);