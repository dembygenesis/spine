-- What each role may do. The API's permission checks read this table at
-- startup; a gated route whose (resource, action) has no row here is refused
-- to everyone, at runtime, with a 403 and no other warning.
CREATE TABLE role_permissions (
    role     TEXT NOT NULL CHECK (role IN ('admin', 'member')),
    resource TEXT NOT NULL,
    action   TEXT NOT NULL,
    PRIMARY KEY (role, resource, action)
);

INSERT INTO role_permissions (role, resource, action) VALUES
    ('admin',  'organization', 'read'),
    ('admin',  'organization', 'write'),
    ('member', 'organization', 'read');
