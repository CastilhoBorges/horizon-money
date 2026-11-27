-- name: CreateMovement :one
with new_movement as (select uuidv7() as id)
insert into movements (
    id, user_id, workspace_id, type, amount, description, date,
    frequency, end_date, anchor_day, last_amount, parts, first_part, series_group_id
)
select n.id, @user_id::uuid, @workspace_id::uuid, @type::text, @amount::bigint, sqlc.narg(description)::text, @date::date,
       sqlc.narg(frequency)::text, sqlc.narg(end_date)::date, sqlc.narg(anchor_day)::smallint,
       sqlc.narg(last_amount)::bigint, sqlc.narg(parts)::smallint, sqlc.narg(first_part)::smallint,
       case when sqlc.narg(frequency)::text is not null then coalesce(sqlc.narg(series_group_id)::uuid, n.id) end
from new_movement n
returning id, series_group_id;

-- name: UpdateMovement :execrows
update movements
set type = @type, amount = @amount, description = sqlc.narg(description), date = @date,
    frequency = sqlc.narg(frequency), end_date = sqlc.narg(end_date), anchor_day = sqlc.narg(anchor_day),
    last_amount = sqlc.narg(last_amount), parts = sqlc.narg(parts), first_part = sqlc.narg(first_part),
    series_group_id = sqlc.narg(series_group_id), updated_at = now()
where id = @id and user_id = @user_id;

-- name: DeleteMovement :execrows
delete from movements
where id = @id and user_id = @user_id;

-- name: MovementExists :one
select exists (select 1 from movements where id = @id and user_id = @user_id);

-- name: GetMovement :one
select sqlc.embed(m),
       coalesce(array_agg(mt.tag_id order by mt.tag_id) filter (where mt.tag_id is not null), '{}')::uuid[] as tag_ids
from movements m
left join movement_tags mt on mt.movement_id = m.id
where m.id = @id and m.user_id = @user_id
group by m.id;

-- name: ListGroupMovements :many
select sqlc.embed(m),
       coalesce(array_agg(mt.tag_id order by mt.tag_id) filter (where mt.tag_id is not null), '{}')::uuid[] as tag_ids
from movements m
left join movement_tags mt on mt.movement_id = m.id
where m.series_group_id = @series_group_id and m.user_id = @user_id
group by m.id
order by m.date, m.id;

-- name: ListLedgerMovements :many
-- One-offs in the range, every series that can produce an occurrence there, and every series with an
-- occurrence moved into it.
select sqlc.embed(m),
       coalesce(array_agg(mt.tag_id order by mt.tag_id) filter (where mt.tag_id is not null), '{}')::uuid[] as tag_ids
from movements m
left join movement_tags mt on mt.movement_id = m.id
where m.user_id = @user_id and m.workspace_id = @workspace_id
  and (
      (m.frequency is null and m.date between @start_date and @to_date)
      or (m.frequency is not null and m.date <= @to_date and (m.end_date is null or m.end_date >= @start_date))
      or (m.frequency is not null and exists (
          select 1 from movement_exceptions e
          where e.movement_id = m.id and not e.deleted and e.date between @start_date and @to_date
      ))
  )
group by m.id
order by m.date, m.id;

-- name: DeleteMovementTags :exec
delete from movement_tags
where movement_id = @movement_id and user_id = @user_id;

-- name: AddMovementTags :exec
insert into movement_tags (movement_id, tag_id, user_id)
select @movement_id::uuid, unnest(@tag_ids::uuid[]), @user_id::uuid;

-- name: UpsertMovementException :exec
insert into movement_exceptions (movement_id, user_id, original_date, deleted, date, amount, description)
values (@movement_id, @user_id, @original_date, @deleted, sqlc.narg(date), sqlc.narg(amount), sqlc.narg(description))
on conflict (movement_id, original_date) do update
set deleted = excluded.deleted, date = excluded.date, amount = excluded.amount,
    description = excluded.description, updated_at = now();

-- name: DeleteExceptionTags :exec
delete from movement_exception_tags
where movement_id = @movement_id and original_date = @original_date and user_id = @user_id;

-- name: AddExceptionTags :exec
insert into movement_exception_tags (movement_id, original_date, user_id, tag_id)
select @movement_id::uuid, @original_date::date, @user_id::uuid, unnest(@tag_ids::uuid[]);

-- name: ListMovementExceptions :many
select sqlc.embed(e),
       coalesce(array_agg(t.tag_id order by t.tag_id) filter (where t.tag_id is not null), '{}')::uuid[] as tag_ids
from movement_exceptions e
left join movement_exception_tags t on t.movement_id = e.movement_id and t.original_date = e.original_date
where e.user_id = @user_id and e.movement_id = any(@movement_ids::uuid[])
group by e.movement_id, e.original_date
order by e.movement_id, e.original_date;
