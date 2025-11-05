ALTER TABLE gameserver_settings
    ADD COLUMN cpu_affinity boolean NOT NULL DEFAULT false;