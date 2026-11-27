-- name: CreateTransfer :one
insert into transfers (user_id, from_workspace_id, to_workspace_id, amount_sent, amount_received, date, description)
values (@user_id, @from_workspace_id, @to_workspace_id, @amount_sent, @amount_received, @date, sqlc.narg(description))
returning *;

-- name: GetTransfer :one
select * from transfers
where id = @id and user_id = @user_id;

-- name: DeleteTransfer :execrows
delete from transfers
where id = @id and user_id = @user_id;

-- name: ListWorkspaceTransfers :many
select * from transfers
where user_id = @user_id
  and (from_workspace_id = @workspace_id or to_workspace_id = @workspace_id)
  and (sqlc.narg(from_date)::date is null or date >= sqlc.narg(from_date)::date)
  and date <= @to_date
order by date, id;
