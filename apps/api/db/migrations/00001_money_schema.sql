-- The v0.1 money schema (persistence spec §5). Every owned table carries user_id, and children reference
-- their parent through (parent_id, user_id), so Postgres refuses a row that mixes two users.

-- +goose Up
create table users (
    id uuid primary key default uuidv7(),
    email text not null,
    display_name text not null,
    main_currency text not null default 'BRL' check (main_currency ~ '^[A-Z]{3}$'),
    timezone text not null default 'America/Sao_Paulo',
    savings_rate_target_min integer not null default 2000,
    savings_rate_target_max integer not null default 3000,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint users_savings_rate_target_range check (
        0 <= savings_rate_target_min
        and savings_rate_target_min <= savings_rate_target_max
        and savings_rate_target_max <= 10000
    )
);
create unique index users_email_key on users (lower(email));

create table workspaces (
    id uuid primary key default uuidv7(),
    user_id uuid not null references users (id) on delete cascade,
    name text not null check (name <> ''),
    currency text not null check (currency ~ '^[A-Z]{3}$'),
    initial_balance bigint not null,
    initial_balance_date date not null,
    low_balance_threshold bigint not null default 0,
    forecast_divisor smallint not null default 30 check (forecast_divisor between 28 and 31),
    archived_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique (id, user_id)
);
create index workspaces_user_id_idx on workspaces (user_id);

create table forecast_categories (
    id uuid primary key default uuidv7(),
    user_id uuid not null,
    workspace_id uuid not null,
    name text not null check (name <> ''),
    monthly_amount bigint not null check (monthly_amount > 0),
    created_at timestamptz not null default now(),
    foreign key (workspace_id, user_id) references workspaces (id, user_id) on delete cascade
);
create index forecast_categories_workspace_id_idx on forecast_categories (workspace_id);

create table tags (
    id uuid primary key default uuidv7(),
    user_id uuid not null references users (id) on delete cascade,
    name text not null check (char_length(name) between 1 and 30),
    color text not null check (color in ('gray', 'blue', 'yellow', 'green', 'red', 'purple', 'pink', 'brown')),
    created_at timestamptz not null default now(),
    unique (id, user_id)
);
create unique index tags_user_id_name_key on tags (user_id, lower(name));

-- One-offs and series share this table. A one-off has every repetition column null.
create table movements (
    id uuid primary key default uuidv7(),
    user_id uuid not null,
    workspace_id uuid not null,
    type text not null check (type in ('income', 'expense', 'daily', 'saving', 'card')),
    amount bigint not null check (amount > 0),
    description text check (char_length(description) <= 100),
    date date not null,
    frequency text check (frequency in ('daily', 'weekly', 'monthly', 'installments')),
    end_date date,
    anchor_day smallint check (anchor_day between 1 and 31),
    last_amount bigint check (last_amount > 0),
    parts smallint check (parts between 2 and 360),
    first_part smallint,
    series_group_id uuid,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique (id, user_id),
    foreign key (workspace_id, user_id) references workspaces (id, user_id) on delete cascade,
    constraint movements_end_on_or_after_start check (end_date >= date),
    constraint movements_kind check (
        (frequency is null and end_date is null and anchor_day is null and last_amount is null
            and parts is null and first_part is null and series_group_id is null)
        or (frequency in ('daily', 'weekly', 'monthly') and anchor_day is not null and series_group_id is not null
            and last_amount is null and parts is null and first_part is null)
        or (frequency = 'installments' and anchor_day is not null and series_group_id is not null
            and last_amount is not null and parts is not null and first_part between 1 and parts)
    )
);
create index movements_workspace_id_date_idx on movements (workspace_id, date);
create index movements_series_idx on movements (workspace_id) where frequency is not null;
create index movements_series_group_id_idx on movements (series_group_id) where series_group_id is not null;

create table movement_tags (
    movement_id uuid not null,
    tag_id uuid not null,
    user_id uuid not null,
    primary key (movement_id, tag_id),
    foreign key (movement_id, user_id) references movements (id, user_id) on delete cascade,
    foreign key (tag_id, user_id) references tags (id, user_id) on delete cascade
);
create index movement_tags_tag_id_idx on movement_tags (tag_id);

-- One row per changed occurrence of a series: a deletion, or a full snapshot that overrides it.
create table movement_exceptions (
    movement_id uuid not null,
    user_id uuid not null,
    original_date date not null,
    deleted boolean not null default false,
    date date,
    amount bigint check (amount > 0),
    description text check (char_length(description) <= 100),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    primary key (movement_id, original_date),
    unique (movement_id, original_date, user_id),
    foreign key (movement_id, user_id) references movements (id, user_id) on delete cascade,
    constraint movement_exceptions_snapshot check (
        (deleted and date is null and amount is null and description is null)
        or (not deleted and date is not null and amount is not null)
    )
);

create table movement_exception_tags (
    movement_id uuid not null,
    original_date date not null,
    user_id uuid not null,
    tag_id uuid not null,
    primary key (movement_id, original_date, tag_id),
    foreign key (movement_id, original_date, user_id)
        references movement_exceptions (movement_id, original_date, user_id) on delete cascade,
    foreign key (tag_id, user_id) references tags (id, user_id) on delete cascade
);
create index movement_exception_tags_tag_id_idx on movement_exception_tags (tag_id);

-- A transfer is one row with both sides. The workspace keys use NO ACTION, checked at the end of the
-- statement: deleting a user cascades through its transfers, but deleting one workspace that has transfers
-- fails (workspace.has_transfers).
create table transfers (
    id uuid primary key default uuidv7(),
    user_id uuid not null references users (id) on delete cascade,
    from_workspace_id uuid not null,
    to_workspace_id uuid not null,
    amount_sent bigint not null check (amount_sent > 0),
    amount_received bigint not null check (amount_received > 0),
    date date not null,
    description text check (char_length(description) <= 100),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint transfers_distinct_workspaces check (from_workspace_id <> to_workspace_id),
    constraint transfers_from_workspace_fkey foreign key (from_workspace_id, user_id) references workspaces (id, user_id),
    constraint transfers_to_workspace_fkey foreign key (to_workspace_id, user_id) references workspaces (id, user_id)
);
create index transfers_from_workspace_id_date_idx on transfers (from_workspace_id, date);
create index transfers_to_workspace_id_date_idx on transfers (to_workspace_id, date);

create table checked_days (
    workspace_id uuid not null,
    user_id uuid not null,
    date date not null,
    primary key (workspace_id, date),
    foreign key (workspace_id, user_id) references workspaces (id, user_id) on delete cascade
);

-- +goose Down
drop table checked_days;
drop table transfers;
drop table movement_exception_tags;
drop table movement_exceptions;
drop table movement_tags;
drop table movements;
drop table tags;
drop table forecast_categories;
drop table workspaces;
drop table users;
