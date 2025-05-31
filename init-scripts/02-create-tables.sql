create sequence public.clients_id_seq;

alter sequence public.clients_id_seq owner to gogymrest;

alter sequence public.clients_id_seq owned by public.clients.id;

create sequence public.users_id_seq;

alter sequence public.users_id_seq owner to gogymrest;

alter sequence public.users_id_seq owned by public.users.id;

create sequence public.states_id_seq;

alter sequence public.states_id_seq owner to gogymrest;

alter sequence public.states_id_seq owned by public.states.id;

create sequence public.memberships_id_seq;

alter sequence public.memberships_id_seq owner to gogymrest;

alter sequence public.memberships_id_seq owned by public.memberships.id;

create sequence public.client_memberships_id_seq;

alter sequence public.client_memberships_id_seq owner to gogymrest;

alter sequence public.client_memberships_id_seq owned by public.client_memberships.id;

create sequence public.gyms_id_seq;

alter sequence public.gyms_id_seq owner to gogymrest;

alter sequence public.gyms_id_seq owned by public.gyms.id;

create sequence public.membership_gyms_id_seq;

alter sequence public.membership_gyms_id_seq owner to gogymrest;

alter sequence public.membership_gyms_id_seq owned by public.membership_gyms.id;

create sequence public.client_passes_id_seq;

alter sequence public.client_passes_id_seq owner to gogymrest;

alter sequence public.client_passes_id_seq owned by public.client_passes.id;

create sequence public.gym_reservations_id_seq;

alter sequence public.gym_reservations_id_seq owner to gogymrest;

alter sequence public.gym_reservations_id_seq owned by public.gym_reservations.id;

create sequence public.table_name_id_seq;

alter sequence public.table_name_id_seq owner to gogymrest;

alter sequence public.table_name_id_seq owned by public.gym_stats.id;

create sequence public.user_gyms_id_seq;

alter sequence public.user_gyms_id_seq owner to gogymrest;

alter sequence public.user_gyms_id_seq owned by public.user_gyms.id;

create sequence public.user_clients_id_seq;

alter sequence public.user_clients_id_seq owner to gogymrest;

alter sequence public.user_clients_id_seq owned by public.user_clients.id;

create sequence public.gym_machines_id_seq;

alter sequence public.gym_machines_id_seq owner to gogymrest;

alter sequence public.gym_machines_id_seq owned by public.gym_machines.id;


create table public.clients
(
    id                integer generated always as identity
        constraint clients_pk
            primary key,
    name              varchar(128),
    cif               varchar(13),
    dob               date,
    trade_register_no varchar(16),
    country_id        integer,
    state_id          integer,
    city              varchar(64),
    street_name       varchar(64),
    street_no         varchar(16),
    building          varchar(16),
    floor             varchar(8),
    apartment         varchar(8),
    created_on        date default now(),
    updated_on        date default now(),
    created_by        integer,
    updated_by        integer
);

alter table public.clients
    owner to gogymrest;

create unique index clients_cif_uindex
    on public.clients (cif);

create table public.users
(
    id              integer generated always as identity
        constraint users_pk
            primary key,
    full_name       varchar(128),
    username        varchar(64),
    password_hashed varchar(512),
    cif             varchar(13),
    email           varchar(128),
    created_on      date default now(),
    updated_on      date default now()
);

alter table public.users
    owner to gogymrest;

create table public.countries
(
    id       integer not null
        constraint countries_pk
            primary key,
    name     varchar(128),
    iso_code varchar(3)
);

alter table public.countries
    owner to gogymrest;

create table public.states
(
    id         integer generated always as identity
        constraint states_pk
            primary key,
    name       varchar(128),
    iso_code   varchar(3),
    country_id integer
);

alter table public.states
    owner to gogymrest;

create index states_country_id_index
    on public.states (country_id);

create table public.memberships
(
    id        integer generated always as identity
        constraint memberships_pk
            primary key,
    name      varchar(128),
    is_active boolean default false,
    days_no   integer default 30,
    level     integer default 0
);

alter table public.memberships
    owner to gogymrest;

create table public.client_memberships
(
    id            integer generated always as identity
        constraint client_memberships_pk
            primary key,
    client_id     integer,
    membership_id integer,
    starting_from date,
    ending_on     date,
    status        varchar(8),
    created_on    date default now(),
    updated_on    date default now(),
    created_by    integer,
    updated_by    integer,
    canceleted_on date
);

comment on column public.client_memberships.status is 'active/inactive/freezed';

alter table public.client_memberships
    owner to gogymrest;

create index client_memberships_client_id_index
    on public.client_memberships (client_id);

create index client_memberships_membership_id_index
    on public.client_memberships (membership_id);

create table public.gyms
(
    id      integer generated always as identity
        constraint gyms_pk
            primary key,
    name    varchar(128),
    members integer
);

alter table public.gyms
    owner to gogymrest;

create table public.membership_gyms
(
    id            integer generated always as identity
        constraint membership_gyms_pk
            primary key,
    membership_id integer,
    gym_id        integer,
    created_on    date default now(),
    created_by    integer,
    updated_on    date default now(),
    updated_by    integer
);

alter table public.membership_gyms
    owner to gogymrest;

create index membership_gyms_gym_id_index
    on public.membership_gyms (gym_id);

create index membership_gyms_membership_id_index
    on public.membership_gyms (membership_id);

create table public.gym_machines
(
    id         integer generated always as identity
        constraint gym_machines_pk
            primary key,
    gym_id     integer,
    machine_id integer,
    created_on date default now(),
    updated_on date default now(),
    created_by integer,
    updated_by integer
);

alter table public.gym_machines
    owner to gogymrest;

create index gym_machines_gym_id_index
    on public.gym_machines (gym_id);

create index gym_machines_machine_id_index
    on public.gym_machines (machine_id);

create table public.machines
(
    id         integer not null
        constraint machines_pk
            primary key,
    name       varchar(128),
    created_on date default now(),
    updated_on date default now(),
    created_by integer,
    updated_by integer
);

alter table public.machines
    owner to gogymrest;

create table public.client_passes
(
    id         integer generated always as identity
        constraint client_passes_pk
            primary key,
    gym_id     integer,
    client_id  integer,
    created_on date default now(),
    action     varchar(3),
    created_by integer
);

comment on column public.client_passes.action is 'IN/OUT';

alter table public.client_passes
    owner to gogymrest;

create index client_passes_client_id_index
    on public.client_passes (client_id);

create index client_passes_gym_id_index
    on public.client_passes (gym_id);

create table public.gym_reservations
(
    id         integer generated always as identity
        constraint gym_reservations_pk
            primary key,
    gym_id     integer,
    client_id  integer,
    from_date  date,
    to_date    integer,
    created_on date default now(),
    created_by integer
);

alter table public.gym_reservations
    owner to gogymrest;

create index gym_reservations_client_id_index
    on public.gym_reservations (client_id);

create index gym_reservations_gym_id_index
    on public.gym_reservations (gym_id);

create table public.gym_stats
(
    id                   integer generated always as identity
        constraint gym_stats_pk
            primary key,
    gym_id               integer,
    max_people           integer default 0,
    max_resevations      integer default 0,
    current_people       integer default 0,
    current_reservations integer default 0,
    current_combined     integer default 0
);

alter table public.gym_stats
    owner to gogymrest;

create table public.user_gyms
(
    id        integer generated always as identity
        constraint user_gyms_pk
            primary key,
    user_id   integer,
    gym_id    integer,
    crated_on date
);

alter table public.user_gyms
    owner to gogymrest;

create index user_gyms_gym_id_index
    on public.user_gyms (gym_id);

create index user_gyms_user_id_index
    on public.user_gyms (user_id);

create table public.user_clients
(
    id         integer generated always as identity
        constraint user_clients_pk
            primary key,
    user_id    integer,
    client_id  integer,
    created_on date default now()
);

alter table public.user_clients
    owner to gogymrest;

create index user_clients_client_id_index
    on public.user_clients (client_id);

create index user_clients_user_id_index
    on public.user_clients (user_id);

