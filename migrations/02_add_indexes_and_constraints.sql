-- +goose Up

CREATE INDEX idx_departments_parent_id
    ON departments (parent_id);

CREATE INDEX idx_employees_department_id
    ON employees (department_id);

CREATE UNIQUE INDEX uq_departments_parent_name
    ON departments (parent_id, name)
    WHERE parent_id IS NOT NULL;

CREATE UNIQUE INDEX uq_departments_root_name
    ON departments (name)
    WHERE parent_id IS NULL;


-- +goose Down

DROP INDEX IF EXISTS uq_departments_root_name;
DROP INDEX IF EXISTS uq_departments_parent_name;
DROP INDEX IF EXISTS idx_employees_department_id;
DROP INDEX IF EXISTS idx_departments_parent_id;