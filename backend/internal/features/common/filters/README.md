# Common filters

Define a `FieldMap` in each feature using trusted database columns or trusted
GORM expressions. Keys are the public field names; lookups are case-insensitive
and trim whitespace. Never copy a client-provided SQL identifier or expression
into the map.

```go
fields := filters.FieldMap{
    "name": {
        Column: clause.Column{
            Table: "users",
            Name:  "name",
        },
        Type:     filters.String,
        Nullable: true,
    },
    "age": {
        Column: clause.Column{
            Table: "users",
            Name:  "age",
        },
        Type: filters.Integer,
    },
}
model, err := filters.ToFilterRequestModel(request, fields)
if err != nil { return err }
result, err := filters.Find[User](db.WithContext(ctx).Model(&User{}), model, fields)
if err != nil { return err }
response := filters.FromFilterResponseModel(result)
```

`FilterRequest` has the same JSON fields as Team Organization's filter request.
It supports `Equals`, `GreaterThan`, `GreaterThanOrEquals`, `LessThan`,
`LessThanOrEquals`, `Contains`, `StartsWith`, `EndsWith`, and `IsNull`.
`groupPattern` uses zero-based criterion references such as
`{0} OR ({1} AND (NOT {2}))`; unreferenced criteria are appended with AND.
Global search is a contains search over mapped string fields. Timezone and
`offsetMinutes` are parsed and retained but do not transform comparisons, as in
the source implementation. Temporal part extraction uses PostgreSQL SQL.

Filtering and ordering are applied before the count and paged result queries.
`ApplyFilter` can be used separately for projections and joined queries. Pass
an existing GORM query with the required joins; use `Field.Expression` for
server-defined GORM expressions when a mapped field is not a simple column.
`EnumValues` maps enum names to their database values.

The JSON-based content hash is specific to the Go model and is not intended to
share cache keys with the C# model. Go's date parsing and system timezone
identifiers do not support every format or platform-specific identifier that
.NET accepts. PostgreSQL temporal operations require a PostgreSQL connection;
the integration tests exercise the non-temporal query path with SQLite and
check the generated PostgreSQL temporal SQL without a live server.
