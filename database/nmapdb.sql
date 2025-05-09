CREATE TABLE IF NOT EXISTS scan_results (
  id SERIAL PRIMARY KEY,
  scan_id UUID NOT NULL,
  host TEXT NOT NULL,
  scanned_at TIMESTAMP NOT NULL,
  open_ports INTEGER[]
);

CREATE TABLE IF NOT EXISTS scan_status (
  scan_id UUID NOT NULL,
  host TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending', -- pending | in_progress | done | failed
  started_at TIMESTAMP,
  completed_at TIMESTAMP,
  PRIMARY KEY (scan_id, host)
);


-- docker exec -it some-postgres psql -U postgres -d nmapdb -c "
-- CREATE TABLE IF NOT EXISTS scan_status (
--   scan_id UUID NOT NULL,
--   host TEXT NOT NULL,
--   status TEXT NOT NULL DEFAULT 'pending', -- pending | in_progress | done | failed
--   started_at TIMESTAMP,
--   completed_at TIMESTAMP,
--   PRIMARY KEY (scan_id, host)
-- );"