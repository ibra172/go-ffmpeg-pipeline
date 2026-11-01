CREATE SCHEMA ffmpeg_app;

CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE ffmpeg_app.users (
    id            UUID            PRIMARY KEY,
    username      citext NOT NULL UNIQUE CHECK (
        username ~* '^[a-z][a-z0-9_]{2,29}$'
    ),
    password_hash TEXT   NOT NULL
);

CREATE TABLE ffmpeg_app.tasks (
    id UUID PRIMARY KEY,
    status        TEXT        NOT NULL CHECK (status IN ('in_progress', 'ready', 'error')),
    operation     TEXT        NOT NULL CHECK (operation IN ('thumbnail', 'transcode', 'watermark')),
    target_format TEXT        NOT NULL DEFAULT '',
    resolution    TEXT        NOT NULL DEFAULT '',
    input_path    TEXT        NOT NULL,
    output_path   TEXT        NOT NULL DEFAULT '',
    error_msg     TEXT        NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL,

    CHECK (status <> 'ready' OR output_path <> ''),

    user_id       UUID        NOT NULL REFERENCES ffmpeg_app.users(id)
);
