CREATE TABLE green_tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL UNIQUE,
    postgres_version VARCHAR(50) NOT NULL,
    postgres_major INT NOT NULL,
    postgres_minor INT NOT NULL,
    image_digest TEXT,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert default green tag for current setup
INSERT INTO green_tags (name, postgres_version, postgres_major, postgres_minor, active)
VALUES ('default', '16.4', 16, 4, true);

-- Add foreign key to tickets table to enforce green_tag existence
ALTER TABLE tickets 
ADD CONSTRAINT fk_tickets_green_tag 
FOREIGN KEY (green_tag_ref) 
REFERENCES green_tags (name);
