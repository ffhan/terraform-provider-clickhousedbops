-- Create test database
CREATE DATABASE IF NOT EXISTS testdb;

-- Create test table
CREATE TABLE IF NOT EXISTS testdb.users (
    id UInt32,
    user_id String,
    name String,
    email String
) ENGINE = MergeTree()
ORDER BY id;

-- Insert test data
INSERT INTO testdb.users VALUES
    (1, 'a', 'User A', 'a@example.com'),
    (2, 'a', 'User A - Second', 'a2@example.com'),
    (3, 'b', 'User B', 'b@example.com'),
    (4, 'b', 'User B - Second', 'b2@example.com'),
    (5, 'admin', 'Admin User', 'admin@example.com'),
    (6, 'public', 'Public Data', 'public@example.com');
