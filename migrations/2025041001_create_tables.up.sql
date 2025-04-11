CREATE TABLE forms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question TEXT NOT NULL,
    answer TEXT,
    urgency TEXT, -- calculado com base na resposta
    status TEXT NOT NULL CHECK (status IN ('draft', 'closed', 'done', 'reviewed')),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);