create table if not exists authority_instances
(
    id              serial,
    uuid            varchar(255) not null unique,
    name            varchar(255) not null unique,
    url             varchar(255) not null,
    credential_type varchar(255) not null,
    role_id         varchar(255),
    role_secret     varchar(255),
    vault_role      varchar(255),
    mount_path      varchar(255),
    attributes      text,
    primary key (id)
);

create table if not exists certificates
(
    id             serial,
    serial_number  varchar not null,
    uuid           varchar not null unique,
    base64_content varchar null default null,
    meta           text    null default null,
    primary key (id)
);

CREATE INDEX IF NOT EXISTS index_certificates_serial_number ON certificates (serial_number);
CREATE INDEX IF NOT EXISTS index_certificates_uuid ON certificates (uuid);

create table if not exists discoveries
(
    id     serial,
    uuid   varchar not null unique,
    name   varchar not null unique,
    status varchar not null,
    meta   text    null default null,
    primary key (id)
);

create table if not exists discovery_certificates
(
    certificate_id bigint not null,
    discovery_id   bigint not null,
    primary key (certificate_id, discovery_id),
    foreign key (certificate_id) references certificates (id),
    foreign key (discovery_id) references discoveries (id)
);
