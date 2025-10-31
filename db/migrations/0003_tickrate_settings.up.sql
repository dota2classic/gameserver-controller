CREATE TABLE IF NOT EXISTS gameserver_settings (
    matchmaking_mode BIGINT PRIMARY KEY,
    tickrate INT NOT NULL
);
