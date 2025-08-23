-- name: GetUserByEmail :one
SELECT * FROM users 
WHERE lower(email) = lower($1)
LIMIT 1
;

-- name: GetUserByID :one
SELECT * FROM users 
WHERE id = $1
LIMIT 1
;


-- name: GetUserByUsername :one
SELECT * FROM users 
WHERE username = $1
LIMIT 1
;

-- name: CreateUser :one
INSERT INTO users (
  id, email, name, username, phone, created_at
) VALUES ( $1, $2, $3, $4, $5, NOW() )
RETURNING *;


-- name: UpdateUserByID :one
UPDATE users
  SET 
  email = $2,
  name = $3,
  username = $4,
  phone = $5
  WHERE id = $1
RETURNING *;


-- name: DeleteUserByID :exec
DELETE FROM users
  WHERE id = $1;
