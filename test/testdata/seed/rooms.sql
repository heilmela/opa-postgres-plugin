CREATE TABLE rooms (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE room_access (
    room_id TEXT REFERENCES rooms(id),
    user_id TEXT REFERENCES users(id),
    PRIMARY KEY (room_id, user_id)
);

-- Insert rooms
INSERT INTO rooms (id, name) VALUES 
    ('room1', 'Meeting Room A'),
    ('room2', 'Conference Room B'),
    ('room3', 'Executive Suite');

-- Insert users
INSERT INTO users (id, name) VALUES 
    ('user1', 'Alice'),
    ('user2', 'Bob'),
    ('user3', 'Charlie');

-- Assign room access
INSERT INTO room_access (room_id, user_id) VALUES 
    ('room1', 'user1'),
    ('room1', 'user3'),
    ('room2', 'user1'),
    ('room2', 'user2'),
    ('room3', 'user3');