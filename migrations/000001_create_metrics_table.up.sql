CREATE TABLE metrics (
    id serial primary key,
    type varchar(50),
    name varchar(255),
    value float,
    delta bigint
);

CREATE INDEX idx_metrics_name ON metrics(name);

CREATE UNIQUE INDEX idx_unique_type_name ON metrics(name, type);
