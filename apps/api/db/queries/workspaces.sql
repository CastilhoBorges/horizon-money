-- name: CreateWorkspace :one
insert into workspaces (user_id, name, currency, initial_balance, initial_balance_date)
values (@user_id, @name, @currency, @initial_balance, @initial_balance_date)
returning *;

-- name: GetWorkspace :one
select * from workspaces
where id = @id and user_id = @user_id;

-- name: ListWorkspaces :many
select * from workspaces
where user_id = @user_id and (sqlc.arg(include_archived)::boolean or archived_at is null)
order by id;

-- name: UpdateWorkspace :execrows
update workspaces
set name = @name, currency = @currency, initial_balance = @initial_balance,
    initial_balance_date = @initial_balance_date, low_balance_threshold = @low_balance_threshold,
    forecast_divisor = @forecast_divisor, updated_at = now()
where id = @id and user_id = @user_id;

-- name: SetWorkspaceArchived :execrows
update workspaces
set archived_at = case when sqlc.arg(archived)::boolean then coalesce(archived_at, now()) end,
    updated_at = now()
where id = @id and user_id = @user_id;

-- name: DeleteWorkspace :execrows
delete from workspaces
where id = @id and user_id = @user_id;

-- name: WorkspaceExists :one
select exists (select 1 from workspaces where id = @id and user_id = @user_id);

-- name: ListForecastCategories :many
select * from forecast_categories
where workspace_id = @workspace_id and user_id = @user_id
order by id;

-- name: DeleteForecastCategories :exec
delete from forecast_categories
where workspace_id = @workspace_id and user_id = @user_id;

-- name: CreateForecastCategory :exec
insert into forecast_categories (user_id, workspace_id, name, monthly_amount)
values (@user_id, @workspace_id, @name, @monthly_amount);
