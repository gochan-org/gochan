# SQL string macros
To make writing SQL queries for gochan that can be used on MySQL, Postgresql, and SQLite easier without having to write a bunch of `switch sqlConfig.DBType` blocks or `query = "SELECT * FROM " + sqlConfig.DBprefix + "table..."`, gochan uses a replacer that replaces certain strings with an appropriate string when running queries through the `gcsql` package.

## Configuration-based replacers
Input    | Output
---------|--------
DBPREFIX | value of `config.SQLConfig.DBprefix`
DBNAME   | value of `config.SQLConfig.DBname`

## SQL implementation-based replacers

Input                        | MySQL/MariaDB           | Postgresql  | SQLite
-----------------------------|-------------------------|-------------|-------------------
?<sup>1</sup>                | ?                       | $#          | $#
RANGE_START_ATON<sup>2</sup> | INET6_ATON(range_start) | range_start | range_start
RANGE_START_NTOA             | INET6_NTOA(range_start) | range_start | range_start
RANGE_END_ATON               | INET6_ATON(range_end)   | range_end   | range_end
RANGE_END_NTOA               | INET6_NTOA(range_end)   | range_end   | range_end
IP_ATON                      | INET6_ATON(ip)          | ip          | ip
IP_NTOA                      | INET6_NTOA(ip)          | ip          | ip
PARAM_ATON                   | INET6_ATON(?)           | $#          | $#
PARAM_NTOA                   | INET6_NTOA(?)           | $#          | $#

## Example
```Go
// SQL implementation-independent usage style used by gochan
rows, err := gcsql.QuerySQL("SELECT id, reason, RANGE_START_NTOA, RANGE_END_NTOA FROM DBNAME.DBPREFIXip_bans WHERE RANGE_START_ATON = PARAM_ATON", "192.168.56.1")
```
The above code essentially does the same as the bottom code
```Go
sqlConfig := config.GetSQLConfig()
dbPrefix := sqlConfig.DBprefix
dbName := sqlConfig.DBname // probably not necessary for most queries, but included here for documentation
db := gcsql.GetBaseDB() // don't use this unless you know what you're doing and really need to access the *sql.DB object

var stmt *sql.Stmt
var err error
switch sqlConfig.DBtype {
case "mysql":
    stmt, err = db.Prepare("SELECT id, reason, INET6_NTOA(range_start), INET6_NTOA(range_end) FROM " + dbName + "." + dbPrefix + "ip_bans WHERE INET6_ATON(range_start) = INET6_ATON(?)")
case "postgres":
    stmt, err = db.Prepare("SELECT id, reason, range_start, range_end FROM " + dbName + "." + dbPrefix + "ip_bans WHERE range_start = $1")
case "sqlite3":
    stmt, err = db.Prepare("SELECT id, reason, range_start, range_end FROM " + dbName + "." + dbPrefix + "ip_bans WHERE range_start = $1")
}
// error handling here
defer stmt.Close()
rows, err := stmt.Query("192.168.56.1")
```

## Notes
1. For numbered query parameters (which are exclusively used in gochan as of this writing, as opposed to named parameters, which may or may not end up replacing them), MySQL and MariaDB use `?` for every parameter. Postgresql and SQLite use $ followed by the parameter number (e.g., `$1`, `$2`, ...)
2. Although SQLite can store IP addresses in a `VARBINARY` column like MySQL, SQLite does not have a built-in function for converting them to and from a string like MySQL's `INET6_NTOA()` and `INET6_ATON()`, or a built-in data type to compare them to string parameters like Postgres, so `range_start < PARAM_ATON` will not work when using the sqlite3 driver.