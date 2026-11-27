-- name: CreateUser :one
insert into users (email, display_name)
values (sqlc.arg(email), sqlc.arg(display_name))
returning *;

-- name: GetUser :one
select * from users
where id = sqlc.arg(id);
