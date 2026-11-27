-- name: CreateTag :one
insert into tags (user_id, name, color)
values (@user_id, @name, @color)
returning *;

-- name: ListTags :many
select * from tags
where user_id = @user_id
order by lower(name), id;
