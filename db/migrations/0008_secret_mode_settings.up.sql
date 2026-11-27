INSERT INTO gameserver_settings (matchmaking_mode, tickrate, image, load_timeout, cpu_affinity)
SELECT 14, tickrate, image, load_timeout, cpu_affinity
FROM gameserver_settings
WHERE matchmaking_mode = 1
ON CONFLICT (matchmaking_mode) DO NOTHING;
