
/*
    Adding/removing enum types as well as adding/removing
    their values here will be automatically handled at
    server start-up.
    
    If a removed enum or one of its values is still referenced
    in the database that will need to be handled manually.
*/

CREATE TYPE visibility AS ENUM (
    'public',
    'unlisted',
    'private'
);

CREATE TYPE postkind AS ENUM (
    'library',
    'forums'
);

CREATE TYPE paragraph AS ENUM (
    'p',
    'ol',
    'ul',
    'blockquote',
    'h2'
);

CREATE TYPE duration AS ENUM (
    'days',
    'week',
    'month',
    'months',
    'year',
    'years'
);
