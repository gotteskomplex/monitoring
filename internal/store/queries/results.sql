-- name: InsertCheckResult :execrows
INSERT INTO check_results (time, tenant_id, check_id, satellite_id, status, duration_ms, attempt, message)
VALUES (@time, @tenant_id, @check_id, @satellite_id, @status, @duration_ms, @attempt, @message)
ON CONFLICT DO NOTHING;

-- name: CountCheckResults :one
SELECT count(*) FROM check_results;

-- name: ListRecentResults :many
SELECT * FROM check_results
WHERE check_id = @check_id
ORDER BY time DESC
LIMIT @max_rows;
