# come

come is a declarative dsl that compiles into go apis. it uses deliberate trollish keywords and transpiles to vertically-sliced go projects using stdlib net/http and raw database/sql.

## setup

install go 1.23 or later. clone this repo. build the compiler:

```bash
go build ./cmd/comec
```

this produces a `comec` binary. move it somewhere on your path if you want global access:

```bash
mv comec /usr/local/bin/
```

## quickstart

```bash
comec init myapi
cd myapi
comec add feature users
comec build
cd generated
go run ./cmd/server
```

server runs on `:8080`.

## cli commands

| command | description |
|---------|-------------|
| `comec init <name>` | scaffold a new come project with app.come |
| `comec add feature <name>` | add a feature folder with a .come template |
| `comec build` | compile all .come files into a go project |
| `comec run` | build and run the generated api |
| `comec migrate [up\|down]` | list pending migrations |
| `comec seed` | run seed data |

`build` accepts flags: `-in` (project root, default `.`) and `-out` (output dir, default `generated`).

## project structure

a come project looks like this:

```
myapp/
  app.come              # root config (required)
  users/
    users.come          # user feature
  products/
    products.come       # product feature
  auth/
    auth.come           # auth feature
```

`app.come` is the entry point. `borrow` directives tell the compiler which feature folders to include.

the generated go project uses vertical slice architecture:

```
generated/
  cmd/server/main.go           # entry point
  internal/
    users/
      types.go                 # model, request, response types
      handler.go               # http handlers
      repository.go            # database operations
      routes.go                # route registration
    products/
      ...
  pkg/
    database/pool.go           # connection pool (postgres + sqlite)
    server/router.go           # middleware chain
    server/middleware.go       # cors, logging, recovery
    server/response.go         # json response helpers
    auth/jwt.go                # jwt sign/validate (if bouncer is used)
  migrations/
    000001_create_users.up.sql
    000001_create_users.down.sql
  go.mod
  go.sum
```

## keywords

### nogo - app name

```
nogo "myapp"
```

required. one per project. must be in `app.come`.

### pileup - database

```
pileup postgres "postgres://user:pass@localhost:5432/mydb"
pileup sqlite "./dev.db" env=DEV
```

at least one required. `postgres` and `sqlite` are supported. the first declaration is the primary driver. add `env=TAG` to mark a secondary connection that only activates when that env var is set.

the generated code auto-detects the driver from the connection string and handles parameter placeholder translation (`$1` vs `?`).

### aura - server config

```
aura {
  port 8080
  read_timeout 10s
  write_timeout 10s
  idle_timeout 60s
}
```

optional. `port` defaults to 8080. duration values accept `s`, `m`, `h` suffixes.

### unblockthehomies - cors

```
unblockthehomies "*"
```

required. sets the access-control-allow-origin header.

### bouncer - jwt auth

```
bouncer jwt {
  secret env.JWT_SECRET
  expire 24h
  algorithm hs256
}
```

optional. enables jwt authentication. `secret` can reference an env var with `env.NAME`. `expire` is a duration. `algorithm` defaults to `hs256`.

### borrow - import feature

```
borrow "./users"
borrow "./auth"
```

used in `app.come` to include feature folders. paths are relative to the project root.

### vibes - environment variables

```
vibes {
  DB_URL env.DATABASE_URL @required
  JWT_SECRET env.JWT_SECRET @required
  PORT env.PORT @default(8080)
}
```

optional. declares environment variables the app reads at startup. `@required` makes the app fail if missing. `@default` provides a fallback.

### manifest - data model

```
manifest User {
  id          uuid      @primary @default(gen_random_uuid)
  email       string    @unique @required @max(255)
  name        string    @required @max(100)
  age         int?      @min(0) @max(150)
  role        pick Role  @default(member)
  bio         string?   @optional
  created_at  timestamp @default(now)
  updated_at  timestamp @default(now) @auto_update
  
  spotlight email
  spotlight role
}
```

defines a database table and its go struct. field syntax: `name type? decorators`.

**types:** `string`, `int`, `float`, `bool`, `timestamp`, `uuid`, `json`, `bytes`, `pick EnumName`

append `?` for nullable fields (e.g. `int?`, `string?`).

**decorators:**

| decorator | meaning |
|-----------|---------|
| `@primary` | primary key |
| `@unique` | unique constraint |
| `@required` | not null |
| `@optional` | nullable in request |
| `@default(value)` | database default |
| `@min(n)` | minimum value/length |
| `@max(n)` | maximum value/length |
| `@email` | email format validation |
| `@oneof(a,b,c)` | must be one of listed values |
| `@homie(Model)` | foreign key reference |
| `@auto_update` | auto-set to current time on update |

**special defaults:**
- `@default(gen_random_uuid)` - auto-generate uuid
- `@default(now)` - auto-set current timestamp

**spotlight** creates a database index. multiple fields create a composite index.

### pick - enum

```
pick Role {
  admin
  moderator
  member
}
```

creates string constants and a validation map. used with `pick Role` type in manifests.

### yeet - route

```
yeet GET "/api/users" list_users {
  ward cors
  ward log
  ward bouncer
  ward bouncer @role(admin)
  
  vouch { ... }
  grabit { ... }
  bouncer sign { ... }
  
  hurl 200 result
  hurl 201 result
  hurl 204
  hurl 400 validation
  hurl 404 {status: "error", message: "not found"}
}
```

defines an http route. methods: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`. path params use `:name` syntax (e.g. `/api/users/:id`).

### ward - middleware

```
ward cors          # enable cors
ward log           # request logging
ward bouncer       # require jwt
ward bouncer @role(admin)  # require jwt + role check
ward recover       # panic recovery (applied globally)
```

applied per-route. `bouncer` checks the authorization header for a valid jwt. `@role(name)` also verifies the role claim.

### vouch - request validation

```
vouch {
  email @email @required
  name @required @min(1) @max(100)
  age @min(0) @max(150)
  role @oneof(admin,moderator,member)
  bio @optional
}
```

declares which fields the request body must contain and their validation rules. used with `POST` and `PUT` routes.

for `POST` routes, fields not marked `@optional` are required. for `PUT` routes, all fields are optional (partial updates).

### grabit - database query

**select (list):**

```
grabit {
  from User
  where age >= query.min_age
  where age <= query.max_age
  where role == query.role
  where name ilike query.name
  order_by query.sort_by @default(created_at)
  order_dir query.order @default(desc)
  limit query.limit @default(20) @max(100)
  offset query.offset @default(0)
}
```

**select (single):**

```
grabit {
  from User
  where id == param.id
  one
}
```

**insert:**

```
grabit {
  insert User body
}
```

**update:**

```
grabit {
  update User
  where id == param.id
  set body
}
```

**delete:**

```
grabit {
  delete from User
  where id == param.id
}
```

**value sources in where clauses:**

| source | syntax | meaning |
|--------|--------|---------|
| query param | `query.field_name` | from url query string |
| path param | `param.field_name` | from url path `:field_name` |
| request body | `body.field_name` | from json body |
| header | `header.field_name` | from request header |
| env var | `env.NAME` | from environment |
| literal | bare value | direct value |

**operators:** `==`, `!=`, `>=`, `<=`, `>`, `<`, `ilike`

### hurl - response

```
hurl 200 result                    # return query result as json
hurl 201 result                    # return created resource
hurl 204                           # no content
hurl 400 validation                # return validation errors
hurl 404 {status: "error", message: "not found"}   # custom json
```

the first 2xx hurl is the success response. 4xx hurls are used in error handling.

for list queries, `hurl 200 result` returns `{"data": [...], "total": N, "page": N, "limit": N}`.

### bouncer sign - issue token

```
bouncer sign {
  sub result.id
  role result.role
}
```

generates a jwt after the database operation. `sub` and `role` map to claims. `sub` is required. `role` is optional.

### spawnchaos - seed data

```
spawnchaos "../seed.json" root "users" unique "email"
```

copies a json file into the generated project. `root` is the json key containing the array. `unique` is the field used for idempotent upsert.

### rawgo - inline go

```
rawgo {
  func customHelper() string {
    return "hello"
  }
}
```

injects raw go code into the generated feature package. use for helpers or logic the dsl cannot express.

### reshape - custom migration

```
reshape up "ALTER TABLE users ADD COLUMN bio TEXT"
reshape down "ALTER TABLE users DROP COLUMN bio"
```

adds custom sql to the migration files beyond what manifests generate.

## generated dependencies

the generated go project depends on:

| package | purpose |
|---------|---------|
| `github.com/lib/pq` | postgres driver |
| `modernc.org/sqlite` | pure-go sqlite driver |
| `github.com/google/uuid` | uuid generation |
| `github.com/golang-jwt/jwt/v5` | jwt signing and validation |
| `golang.org/x/crypto` | bcrypt password hashing |

the compiler itself has zero dependencies (pure stdlib go).

## full example

### app.come

```
nogo "myapp"

pileup postgres "postgres://localhost/myapp"

aura {
  port 8080
  read_timeout 10s
  write_timeout 10s
}

unblockthehomies "*"

borrow "./users"
```

### users/users.come

```
pick Role {
  admin
  moderator
  member
}

manifest User {
  id          uuid      @primary @default(gen_random_uuid)
  email       string    @unique @required @max(255)
  name        string    @required @max(100)
  age         int?      @min(0) @max(150)
  role        pick Role  @default(member)
  created_at  timestamp @default(now)
  updated_at  timestamp @default(now) @auto_update

  spotlight email
  spotlight role
}

yeet GET "/api/users" list_users {
  ward cors
  ward log

  grabit {
    from User
    where age >= query.min_age
    where age <= query.max_age
    where role == query.role
    where name ilike query.name
    order_by query.sort_by @default(created_at)
    order_dir query.order @default(desc)
    limit query.limit @default(20)
    offset query.offset @default(0)
  }

  hurl 200 result
}

yeet POST "/api/users" create_user {
  ward cors
  ward log

  vouch {
    email @email @required
    name @required @min(1) @max(100)
    age @min(0) @max(150)
  }

  grabit {
    insert User body
  }

  hurl 201 result
  hurl 400 validation
}

yeet GET "/api/users/:id" get_user {
  ward cors
  ward log

  grabit {
    from User
    where id == param.id
    one
  }

  hurl 200 result
  hurl 404 {status: "error", message: "user not found"}
}

yeet PUT "/api/users/:id" update_user {
  ward cors
  ward log

  vouch {
    email @email @optional
    name @optional @min(1) @max(100)
  }

  grabit {
    update User
    where id == param.id
    set body
  }

  hurl 200 result
  hurl 404 {status: "error", message: "user not found"}
}

yeet DELETE "/api/users/:id" delete_user {
  ward cors
  ward bouncer @role(admin)
  ward log

  grabit {
    delete from User
    where id == param.id
  }

  hurl 204
}
```

### compile and run

```bash
comec build
cd generated
go run ./cmd/server
```

## how it works

1. `comec build` reads `app.come` from the project root
2. parses it into an ast using a recursive descent parser
3. resolves `borrow` paths and parses each feature `.come` file
4. validates the combined ast (model references, required declarations)
5. generates go source files per feature + shared infrastructure
6. writes everything to the output directory
7. runs `go mod tidy` on the generated project

## design philosophy

come is declarative. you describe what your api looks like (models, routes, validation, queries) and the compiler generates all the go implementation. this means:

- no boilerplate handler code to write
- no manual sql query building
- no middleware wiring
- consistent project structure across all come projects
- vertical slice architecture by default

when the dsl cannot express something, use `rawgo` blocks to inject custom go code.

## limitations

- one grabit block per route (no multi-step transactions)
- no websocket support
- no file upload handling
- no openapi/swagger generation (yet)
- auth is jwt-only
- no soft deletes
- no composite primary keys
