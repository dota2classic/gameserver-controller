ALTER TABLE gameserver_settings
    ADD COLUMN load_timeout int NOT NULL DEFAULT 90;