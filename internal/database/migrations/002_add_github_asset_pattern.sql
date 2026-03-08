-- Add customizable GitHub release asset filename pattern
ALTER TABLE products ADD COLUMN source_github_asset_pattern TEXT DEFAULT '';
