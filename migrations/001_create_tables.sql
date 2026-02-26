CREATE TABLE IF NOT EXISTS jobs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id      VARCHAR(255) NOT NULL UNIQUE,
    image_url   TEXT NOT NULL,
    status      VARCHAR(50)  NOT NULL DEFAULT 'completed',
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS predictions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id     VARCHAR(255)     NOT NULL REFERENCES jobs(job_id) ON DELETE CASCADE,
    x          DOUBLE PRECISION NOT NULL,
    y          DOUBLE PRECISION NOT NULL,
    width      DOUBLE PRECISION NOT NULL,
    height     DOUBLE PRECISION NOT NULL,
    confidence DOUBLE PRECISION NOT NULL,
    class      VARCHAR(255)     NOT NULL,
    class_id   INTEGER          NOT NULL,
    created_at TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_predictions_job_id ON predictions(job_id);
