ALTER TABLE builds          ADD COLUMN uploaded_by TEXT NOT NULL DEFAULT '';
ALTER TABLE upload_sessions ADD COLUMN uploaded_by TEXT NOT NULL DEFAULT '';
