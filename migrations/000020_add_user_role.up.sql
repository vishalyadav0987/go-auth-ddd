-- Authorisation role for each user. Every existing and new user is a
-- regular USER; promote administrators explicitly, e.g.:
--   UPDATE users SET role = 'ADMIN' WHERE id = '<user-id>';
ALTER TABLE users
ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT 'USER';

ALTER TABLE users
ADD CONSTRAINT chk_users_role
CHECK (role IN ('USER', 'ADMIN'));
