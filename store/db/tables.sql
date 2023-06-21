
CREATE TABLE IF NOT EXISTS mailing (
    id        bigserial PRIMARY KEY,
    email     text      NOT NULL,
    code      text
);

CREATE TABLE IF NOT EXISTS reserved (
    id        bigserial PRIMARY KEY,
    handle    text      NOT NULL,
    email     text      NOT NULL,
    code      text,
    redeemed  boolean
);

CREATE TABLE IF NOT EXISTS accounts (
    
    id         bigserial PRIMARY KEY,
    
    code       text,
    
    created    bigint  NOT NULL,
    updated    bigint  NOT NULL,
    deleted    bool,
    
    email      text NOT NULL,
    pass       text NOT NULL
);

CREATE TABLE IF NOT EXISTS change_email (
    ref_id   bigint   REFERENCES accounts(id) ON DELETE CASCADE,
    email    text     NOT NULL,
    code     text     NOT NULL
);

CREATE TABLE IF NOT EXISTS change_password (
    ref_id   bigint   REFERENCES accounts(id) ON DELETE CASCADE,
    pass     text     NOT NULL,
    code     text     NOT NULL
);

CREATE TABLE IF NOT EXISTS personas (
    
    acc_id bigint    REFERENCES accounts(id) ON DELETE CASCADE,
    id     bigserial PRIMARY KEY,
    
    admin     boolean,
    default_p boolean,
    deleted   boolean,
    
    created bigint NOT NULL,
    updated bigint NOT NULL,
    
    visibility visibility NOT NULL,
    
    slug   text NOT NULL,
    handle text NOT NULL,
    name   text NOT NULL,
    avatar text
);

CREATE TABLE IF NOT EXISTS pronouns (
    ref_id  bigint  REFERENCES personas(id) ON DELETE CASCADE,
    pronoun text    NOT NULL
);

CREATE TABLE IF NOT EXISTS logins (
    acc_id    bigint  REFERENCES accounts(id) ON DELETE CASCADE,
    p_id      bigint  REFERENCES personas(id) ON DELETE CASCADE,
    token     text    NOT NULL,
    since     int     NOT NULL
);

CREATE TABLE IF NOT EXISTS event (

    id       bigserial  PRIMARY KEY,
    ref_id   bigint     REFERENCES personas(id) ON DELETE CASCADE,

    slug     text     UNIQUE NOT NULL,
    created  bigint   NOT NULL,
    updated  bigint   NOT NULL,
    deleted  bool,

    words int NOT NULL,

    visibility visibility NOT NULL,
    name       text,
    summary    text,
    timezone   text   NOT NULL,
    start      bigint NOT NULL,
    finish     bigint,
    weekly     bool
);

CREATE TABLE IF NOT EXISTS event_setting (
    ref_id    bigint  REFERENCES event(id) ON DELETE CASCADE,
    setting  text    NOT NULL
);

CREATE TABLE IF NOT EXISTS event_category (
    ref_id    bigint  REFERENCES event(id) ON DELETE CASCADE,
    category  text    NOT NULL
);

CREATE TABLE IF NOT EXISTS event_tag (
    ref_id  bigint  REFERENCES event(id) ON DELETE CASCADE,
    tag     text    NOT NULL
);

CREATE TABLE IF NOT EXISTS event_span (

    ref_id bigint REFERENCES event(id) ON DELETE CASCADE,

    p     int  NOT NULL,
    span  int  NOT NULL,

    kind  paragraph  NOT NULL,
    text  text       NOT NULL,
    link  text,

    b boolean,
    i boolean,
    u boolean
);

CREATE TABLE IF NOT EXISTS post (
    
    id       bigserial  PRIMARY KEY,
    ref_id   bigint     REFERENCES personas(id) ON DELETE CASCADE,
    thread   bigint     REFERENCES post(id) ON DELETE CASCADE,
    
    slug     text     UNIQUE NOT NULL,
    created  bigint   NOT NULL,
    updated  bigint   NOT NULL,
    deleted  bool,
    pinned   bool,
    locked   bool,
    
    -- Specifies what order posts should be in.
    idx  int  NOT NULL,
    
    visibility visibility,
    name       text,
    summary    text,
    words      int    NOT NULL
);

CREATE TABLE IF NOT EXISTS post_kind (
    ref_id  bigint    REFERENCES post(id) ON DELETE CASCADE,
    kind    postkind  NOT NULL
);

CREATE TABLE IF NOT EXISTS post_category (
    ref_id    bigint  REFERENCES post(id) ON DELETE CASCADE,
    category  text    NOT NULL
);

CREATE TABLE IF NOT EXISTS post_tag (
    ref_id  bigint  REFERENCES post(id) ON DELETE CASCADE,
    tag     text    NOT NULL
);

CREATE TABLE IF NOT EXISTS post_span (
    
    ref_id bigint REFERENCES post(id) ON DELETE CASCADE,
    
    p     int  NOT NULL,
    span  int  NOT NULL,
    
    kind  paragraph  NOT NULL,
    text  text       NOT NULL,
    link  text,
    
    b boolean,
    i boolean,
    u boolean
);

CREATE TABLE IF NOT EXISTS profile (

    id      bigserial PRIMARY KEY,
    ref_id  bigint    REFERENCES personas(id) ON DELETE CASCADE,
    
    slug    text UNIQUE NOT NULL,
    created bigint  NOT NULL,
    updated bigint  NOT NULL,
    
    available   boolean     NOT NULL,
    visibility  visibility  NOT NULL,
    
    name    text,
    summary text,
    
    duration_start  duration NOT NULL,
    duration_end    duration NOT NULL,
    
    website text,
    email   text,
    discord text
);

CREATE TABLE IF NOT EXISTS profile_tag (
    ref_id  bigint  REFERENCES profile(id) ON DELETE CASCADE,
    tag     text    NOT NULL
);

CREATE TABLE IF NOT EXISTS profile_language (
    ref_id    bigint  REFERENCES profile(id) ON DELETE CASCADE,
    language  text    NOT NULL
);

CREATE TABLE IF NOT EXISTS profile_compensation (
    ref_id        bigint  REFERENCES profile(id) ON DELETE CASCADE,
    compensation  text    NOT NULL
);

CREATE TABLE IF NOT EXISTS profile_medium (
    ref_id  bigint  REFERENCES profile(id) ON DELETE CASCADE,
    medium  text    NOT NULL
);

CREATE TABLE IF NOT EXISTS profile_project (
    
    id        bigserial PRIMARY KEY,
    ref_id    bigint    REFERENCES profile(id) ON DELETE CASCADE,
    
    name      text  NOT NULL,
    link      text,
    teamname  text,
    teamlink  text,
    
    start    bigint  NOT NULL,
    finish   bigint  NOT NULL
);

CREATE TABLE IF NOT EXISTS profile_project_role (
    
    id      bigserial PRIMARY KEY,
    ref_id  bigint    REFERENCES profile_project(id) ON DELETE CASCADE,
    
    name     text NOT NULL,
    comment  text
);

CREATE TABLE IF NOT EXISTS profile_project_role_skill (
    ref_id  bigint REFERENCES profile_project_role(id) ON DELETE CASCADE,
    skill   text   NOT NULL
);

CREATE TABLE IF NOT EXISTS profile_project_role_duty (
    ref_id  bigint REFERENCES profile_project_role(id) ON DELETE CASCADE,
    duty    text   NOT NULL
);

CREATE TABLE IF NOT EXISTS profile_advertised (
    id      bigserial  PRIMARY KEY,
    ref_id  bigint     REFERENCES profile(id) ON DELETE CASCADE,
    skill   text       NOT NULL
);

CREATE TABLE IF NOT EXISTS profile_advertised_example (
    id        bigserial  PRIMARY KEY,
    ref_id    bigint     REFERENCES profile_advertised(id) ON DELETE CASCADE,
    alttext   text       NOT NULL,
    title     text       NOT NULL,
    project   text       NOT NULL,
    info      text       NOT NULL,
    kind      text       NOT NULL,
    filename  text,
    format    text,
    aspect    float
);

CREATE TABLE IF NOT EXISTS file (
    post     bigint  REFERENCES post(id) ON DELETE CASCADE,
    profile  bigint  REFERENCES profile(id) ON DELETE CASCADE,
    event    bigint  REFERENCES event(id) ON DELETE CASCADE,
    persona  bigint  REFERENCES personas(id) ON DELETE CASCADE,
    file     text    NOT NULL
);
