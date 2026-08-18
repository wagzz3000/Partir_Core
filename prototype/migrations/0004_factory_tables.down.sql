-- Rollback factory floor tables

DROP INDEX IF EXISTS idx_station_metrics_recorded_at;
DROP INDEX IF EXISTS idx_station_metrics_station_id;
DROP INDEX IF EXISTS idx_workers_assigned_station;
DROP INDEX IF EXISTS idx_workers_status;
DROP INDEX IF EXISTS idx_workers_type;

DROP TABLE IF EXISTS station_metrics;
DROP TABLE IF EXISTS stations;
DROP TABLE IF EXISTS workers;
