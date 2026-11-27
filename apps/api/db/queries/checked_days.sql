-- name: SetCheckedDay :exec
insert into checked_days (workspace_id, user_id, date)
values (@workspace_id, @user_id, @date)
on conflict do nothing;

-- name: UnsetCheckedDay :exec
delete from checked_days
where workspace_id = @workspace_id and user_id = @user_id and date = @date;

-- name: ListCheckedDays :many
select date from checked_days
where workspace_id = @workspace_id and user_id = @user_id and date between @from_date and @to_date
order by date;
