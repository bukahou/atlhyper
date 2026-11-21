-- 时间统一存 UTC：把带时区偏移写下的时间（容器曾按东京时区写 +09:00）换成 UTC 的 RFC 3339（秒精度，Z 结尾），
-- 与此后所有写入的格式一致，字符串比较与排序才对。
-- 只改能解析的值：解析不了的（strftime 返回 NULL）原样保留，不会被清空。
-- 不含不是时间点的列：ai_role_budget.schedule_start_time / schedule_end_time（HH:MM）。

-- +goose Up
UPDATE cluster_events SET first_timestamp = strftime('%Y-%m-%dT%H:%M:%SZ', first_timestamp)
WHERE first_timestamp IS NOT NULL AND first_timestamp <> '' AND first_timestamp NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', first_timestamp) IS NOT NULL;
UPDATE cluster_events SET last_timestamp = strftime('%Y-%m-%dT%H:%M:%SZ', last_timestamp)
WHERE last_timestamp IS NOT NULL AND last_timestamp <> '' AND last_timestamp NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', last_timestamp) IS NOT NULL;
UPDATE cluster_events SET created_at = strftime('%Y-%m-%dT%H:%M:%SZ', created_at)
WHERE created_at IS NOT NULL AND created_at <> '' AND created_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL;
UPDATE cluster_events SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', updated_at)
WHERE updated_at IS NOT NULL AND updated_at <> '' AND updated_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', updated_at) IS NOT NULL;
UPDATE notify_policy SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', updated_at)
WHERE updated_at IS NOT NULL AND updated_at <> '' AND updated_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', updated_at) IS NOT NULL;
UPDATE alerts SET starts_at = strftime('%Y-%m-%dT%H:%M:%SZ', starts_at)
WHERE starts_at IS NOT NULL AND starts_at <> '' AND starts_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', starts_at) IS NOT NULL;
UPDATE alerts SET ends_at = strftime('%Y-%m-%dT%H:%M:%SZ', ends_at)
WHERE ends_at IS NOT NULL AND ends_at <> '' AND ends_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', ends_at) IS NOT NULL;
UPDATE alerts SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', updated_at)
WHERE updated_at IS NOT NULL AND updated_at <> '' AND updated_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', updated_at) IS NOT NULL;
UPDATE alerts SET last_notified_at = strftime('%Y-%m-%dT%H:%M:%SZ', last_notified_at)
WHERE last_notified_at IS NOT NULL AND last_notified_at <> '' AND last_notified_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', last_notified_at) IS NOT NULL;
UPDATE notify_deliveries SET next_attempt_at = strftime('%Y-%m-%dT%H:%M:%SZ', next_attempt_at)
WHERE next_attempt_at IS NOT NULL AND next_attempt_at <> '' AND next_attempt_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', next_attempt_at) IS NOT NULL;
UPDATE notify_deliveries SET created_at = strftime('%Y-%m-%dT%H:%M:%SZ', created_at)
WHERE created_at IS NOT NULL AND created_at <> '' AND created_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL;
UPDATE notify_deliveries SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', updated_at)
WHERE updated_at IS NOT NULL AND updated_at <> '' AND updated_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', updated_at) IS NOT NULL;
UPDATE notify_receivers SET created_at = strftime('%Y-%m-%dT%H:%M:%SZ', created_at)
WHERE created_at IS NOT NULL AND created_at <> '' AND created_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL;
UPDATE notify_receivers SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', updated_at)
WHERE updated_at IS NOT NULL AND updated_at <> '' AND updated_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', updated_at) IS NOT NULL;
UPDATE notify_rules SET created_at = strftime('%Y-%m-%dT%H:%M:%SZ', created_at)
WHERE created_at IS NOT NULL AND created_at <> '' AND created_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL;
UPDATE notify_rules SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', updated_at)
WHERE updated_at IS NOT NULL AND updated_at <> '' AND updated_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', updated_at) IS NOT NULL;
UPDATE notify_smtp SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', updated_at)
WHERE updated_at IS NOT NULL AND updated_at <> '' AND updated_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', updated_at) IS NOT NULL;
UPDATE notify_silences SET starts_at = strftime('%Y-%m-%dT%H:%M:%SZ', starts_at)
WHERE starts_at IS NOT NULL AND starts_at <> '' AND starts_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', starts_at) IS NOT NULL;
UPDATE notify_silences SET ends_at = strftime('%Y-%m-%dT%H:%M:%SZ', ends_at)
WHERE ends_at IS NOT NULL AND ends_at <> '' AND ends_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', ends_at) IS NOT NULL;
UPDATE notify_silences SET created_at = strftime('%Y-%m-%dT%H:%M:%SZ', created_at)
WHERE created_at IS NOT NULL AND created_at <> '' AND created_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL;
UPDATE users SET created_at = strftime('%Y-%m-%dT%H:%M:%SZ', created_at)
WHERE created_at IS NOT NULL AND created_at <> '' AND created_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL;
UPDATE users SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', updated_at)
WHERE updated_at IS NOT NULL AND updated_at <> '' AND updated_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', updated_at) IS NOT NULL;
UPDATE users SET last_login_at = strftime('%Y-%m-%dT%H:%M:%SZ', last_login_at)
WHERE last_login_at IS NOT NULL AND last_login_at <> '' AND last_login_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', last_login_at) IS NOT NULL;
UPDATE user_identities SET created_at = strftime('%Y-%m-%dT%H:%M:%SZ', created_at)
WHERE created_at IS NOT NULL AND created_at <> '' AND created_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL;
UPDATE clusters SET token_issued_at = strftime('%Y-%m-%dT%H:%M:%SZ', token_issued_at)
WHERE token_issued_at IS NOT NULL AND token_issued_at <> '' AND token_issued_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', token_issued_at) IS NOT NULL;
UPDATE clusters SET created_at = strftime('%Y-%m-%dT%H:%M:%SZ', created_at)
WHERE created_at IS NOT NULL AND created_at <> '' AND created_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL;
UPDATE clusters SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', updated_at)
WHERE updated_at IS NOT NULL AND updated_at <> '' AND updated_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', updated_at) IS NOT NULL;
UPDATE audit_logs SET timestamp = strftime('%Y-%m-%dT%H:%M:%SZ', timestamp)
WHERE timestamp IS NOT NULL AND timestamp <> '' AND timestamp NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', timestamp) IS NOT NULL;
UPDATE command_history SET created_at = strftime('%Y-%m-%dT%H:%M:%SZ', created_at)
WHERE created_at IS NOT NULL AND created_at <> '' AND created_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL;
UPDATE command_history SET started_at = strftime('%Y-%m-%dT%H:%M:%SZ', started_at)
WHERE started_at IS NOT NULL AND started_at <> '' AND started_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', started_at) IS NOT NULL;
UPDATE command_history SET finished_at = strftime('%Y-%m-%dT%H:%M:%SZ', finished_at)
WHERE finished_at IS NOT NULL AND finished_at <> '' AND finished_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', finished_at) IS NOT NULL;
UPDATE settings SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', updated_at)
WHERE updated_at IS NOT NULL AND updated_at <> '' AND updated_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', updated_at) IS NOT NULL;
UPDATE ai_conversations SET created_at = strftime('%Y-%m-%dT%H:%M:%SZ', created_at)
WHERE created_at IS NOT NULL AND created_at <> '' AND created_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL;
UPDATE ai_conversations SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', updated_at)
WHERE updated_at IS NOT NULL AND updated_at <> '' AND updated_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', updated_at) IS NOT NULL;
UPDATE ai_messages SET created_at = strftime('%Y-%m-%dT%H:%M:%SZ', created_at)
WHERE created_at IS NOT NULL AND created_at <> '' AND created_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL;
UPDATE ai_providers SET last_used_at = strftime('%Y-%m-%dT%H:%M:%SZ', last_used_at)
WHERE last_used_at IS NOT NULL AND last_used_at <> '' AND last_used_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', last_used_at) IS NOT NULL;
UPDATE ai_providers SET last_error_at = strftime('%Y-%m-%dT%H:%M:%SZ', last_error_at)
WHERE last_error_at IS NOT NULL AND last_error_at <> '' AND last_error_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', last_error_at) IS NOT NULL;
UPDATE ai_providers SET status_checked_at = strftime('%Y-%m-%dT%H:%M:%SZ', status_checked_at)
WHERE status_checked_at IS NOT NULL AND status_checked_at <> '' AND status_checked_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', status_checked_at) IS NOT NULL;
UPDATE ai_providers SET created_at = strftime('%Y-%m-%dT%H:%M:%SZ', created_at)
WHERE created_at IS NOT NULL AND created_at <> '' AND created_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL;
UPDATE ai_providers SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', updated_at)
WHERE updated_at IS NOT NULL AND updated_at <> '' AND updated_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', updated_at) IS NOT NULL;
UPDATE ai_providers SET deleted_at = strftime('%Y-%m-%dT%H:%M:%SZ', deleted_at)
WHERE deleted_at IS NOT NULL AND deleted_at <> '' AND deleted_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', deleted_at) IS NOT NULL;
UPDATE ai_settings SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', updated_at)
WHERE updated_at IS NOT NULL AND updated_at <> '' AND updated_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', updated_at) IS NOT NULL;
UPDATE ai_provider_models SET created_at = strftime('%Y-%m-%dT%H:%M:%SZ', created_at)
WHERE created_at IS NOT NULL AND created_at <> '' AND created_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL;
UPDATE ai_role_budget SET daily_reset_at = strftime('%Y-%m-%dT%H:%M:%SZ', daily_reset_at)
WHERE daily_reset_at IS NOT NULL AND daily_reset_at <> '' AND daily_reset_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', daily_reset_at) IS NOT NULL;
UPDATE ai_role_budget SET monthly_reset_at = strftime('%Y-%m-%dT%H:%M:%SZ', monthly_reset_at)
WHERE monthly_reset_at IS NOT NULL AND monthly_reset_at <> '' AND monthly_reset_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', monthly_reset_at) IS NOT NULL;
UPDATE ai_role_budget SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', updated_at)
WHERE updated_at IS NOT NULL AND updated_at <> '' AND updated_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', updated_at) IS NOT NULL;
UPDATE ai_reports SET created_at = strftime('%Y-%m-%dT%H:%M:%SZ', created_at)
WHERE created_at IS NOT NULL AND created_at <> '' AND created_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL;
UPDATE slo_targets SET created_at = strftime('%Y-%m-%dT%H:%M:%SZ', created_at)
WHERE created_at IS NOT NULL AND created_at <> '' AND created_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL;
UPDATE slo_targets SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', updated_at)
WHERE updated_at IS NOT NULL AND updated_at <> '' AND updated_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', updated_at) IS NOT NULL;
UPDATE slo_route_mapping SET created_at = strftime('%Y-%m-%dT%H:%M:%SZ', created_at)
WHERE created_at IS NOT NULL AND created_at <> '' AND created_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL;
UPDATE slo_route_mapping SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', updated_at)
WHERE updated_at IS NOT NULL AND updated_at <> '' AND updated_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', updated_at) IS NOT NULL;
UPDATE aiops_dependency_graph_snapshots SET created_at = strftime('%Y-%m-%dT%H:%M:%SZ', created_at)
WHERE created_at IS NOT NULL AND created_at <> '' AND created_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL;
UPDATE aiops_incidents SET started_at = strftime('%Y-%m-%dT%H:%M:%SZ', started_at)
WHERE started_at IS NOT NULL AND started_at <> '' AND started_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', started_at) IS NOT NULL;
UPDATE aiops_incidents SET escalated_at = strftime('%Y-%m-%dT%H:%M:%SZ', escalated_at)
WHERE escalated_at IS NOT NULL AND escalated_at <> '' AND escalated_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', escalated_at) IS NOT NULL;
UPDATE aiops_incidents SET resolved_at = strftime('%Y-%m-%dT%H:%M:%SZ', resolved_at)
WHERE resolved_at IS NOT NULL AND resolved_at <> '' AND resolved_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', resolved_at) IS NOT NULL;
UPDATE aiops_incidents SET created_at = strftime('%Y-%m-%dT%H:%M:%SZ', created_at)
WHERE created_at IS NOT NULL AND created_at <> '' AND created_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL;
UPDATE aiops_incident_timeline SET timestamp = strftime('%Y-%m-%dT%H:%M:%SZ', timestamp)
WHERE timestamp IS NOT NULL AND timestamp <> '' AND timestamp NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', timestamp) IS NOT NULL;
UPDATE aiops_incident_links SET linked_at = strftime('%Y-%m-%dT%H:%M:%SZ', linked_at)
WHERE linked_at IS NOT NULL AND linked_at <> '' AND linked_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', linked_at) IS NOT NULL;
UPDATE aiops_incident_links SET unlinked_at = strftime('%Y-%m-%dT%H:%M:%SZ', unlinked_at)
WHERE unlinked_at IS NOT NULL AND unlinked_at <> '' AND unlinked_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', unlinked_at) IS NOT NULL;
UPDATE aiops_incident_signals SET first_seen = strftime('%Y-%m-%dT%H:%M:%SZ', first_seen)
WHERE first_seen IS NOT NULL AND first_seen <> '' AND first_seen NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', first_seen) IS NOT NULL;
UPDATE aiops_incident_signals SET last_seen = strftime('%Y-%m-%dT%H:%M:%SZ', last_seen)
WHERE last_seen IS NOT NULL AND last_seen <> '' AND last_seen NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', last_seen) IS NOT NULL;
UPDATE github_installations SET created_at = strftime('%Y-%m-%dT%H:%M:%SZ', created_at)
WHERE created_at IS NOT NULL AND created_at <> '' AND created_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL;
UPDATE repo_config SET created_at = strftime('%Y-%m-%dT%H:%M:%SZ', created_at)
WHERE created_at IS NOT NULL AND created_at <> '' AND created_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL;
UPDATE repo_config SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', updated_at)
WHERE updated_at IS NOT NULL AND updated_at <> '' AND updated_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', updated_at) IS NOT NULL;
UPDATE deploy_config SET created_at = strftime('%Y-%m-%dT%H:%M:%SZ', created_at)
WHERE created_at IS NOT NULL AND created_at <> '' AND created_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL;
UPDATE deploy_config SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', updated_at)
WHERE updated_at IS NOT NULL AND updated_at <> '' AND updated_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', updated_at) IS NOT NULL;
UPDATE deploy_history SET deployed_at = strftime('%Y-%m-%dT%H:%M:%SZ', deployed_at)
WHERE deployed_at IS NOT NULL AND deployed_at <> '' AND deployed_at NOT LIKE '%Z' AND strftime('%Y-%m-%dT%H:%M:%SZ', deployed_at) IS NOT NULL;

-- +goose Down
-- 换算不可逆（原偏移已丢弃），不提供回退。
