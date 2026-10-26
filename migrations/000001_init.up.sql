CREATE SCHEMA ffmpeg_app;

CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE ffmpeg_app.users (
    id            UUID            PRIMARY KEY,
    username      citext NOT NULL UNIQUE CHECK (
        username ~* '^[a-z][a-z0-9_]{2,29}$'
    ),
    password_hash TEXT   NOT NULL
);
