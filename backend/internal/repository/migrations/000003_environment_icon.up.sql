-- Add icon field to environments (defaults to 'deployed_code' for existing rows)
ALTER TABLE environments ADD COLUMN icon TEXT NOT NULL DEFAULT 'deployed_code';
