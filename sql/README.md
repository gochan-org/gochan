# SQL string macros
To make writing SQL queries for gochan that can be used on MySQL, PostgreSQL, and SQLite easier without having to write a bunch of `switch sqlConfig.DBType` blocks or `query = "SELECT * FROM " + sqlConfig.DBprefix + "table..."`, gochan uses a replacer that replaces certain strings with an appropriate string when running queries through the `gcsql` package.

## Positional parameters
Currently, Gochan exclusively uses positional parameters, though named parameters may be supported in the future. All query strings should use MySQL/MariaDB-style `?` positional parameters, and the `gcsql` package will convert them to the appropriate format for SQL driver when preparing the statements.

## Configuration-based replacers
Input     | Output
----------|-------------------------
DBPREFIX  | value of `config.SQLConfig.DBprefix`
DBNAME    | value of `config.SQLConfig.DBname`
DBVERSION | value of `gcsql.DatabaseVersion`

## IP address handling
The function `IP_CMP(ip1, ip2)` is provided for MySQL, PostgreSQL, and SQLite for comparing two IP addresses (VARBINARY(16), INET, or string). It returns -1 if ip1 < ip2, 0 if they are equal, and 1 if ip1 > ip2. It expects both parameters to be IPv4 or both to be IPv6; mixing types will result in undefined behavior.
When searching for an IP address, rather than using `SELECT ... WHERE ip = ?`, you should use `SELECT ... WHERE IP_CMP(ip, ?) = 0` to ensure compatibility across all supported database types and avoid potential issues with SQLite's comparison behavior.

## SQL IP address macros (deprecated)
Input            | MySQL/MariaDB           | PostgreSQL  | SQLite
-----------------|-------------------------|-------------|-------------
RANGE_START_ATON | INET6_ATON(range_start) | range_start | INET6_ATON(range_start)
RANGE_START_NTOA | INET6_NTOA(range_start) | range_start | INET6_NTOA(range_start)
RANGE_END_ATON   | INET6_ATON(range_end)   | range_end   | INET6_ATON(range_end)
RANGE_END_NTOA   | INET6_NTOA(range_end)   | range_end   | INET6_NTOA(range_end)
IP_ATON          | INET6_ATON(ip)          | ip          | INET6_ATON(ip)
IP_NTOA          | INET6_NTOA(ip)          | ip          | INET6_NTOA(ip)
PARAM_ATON       | INET6_ATON(?)           | $#          | INET6_ATON($#)
PARAM_NTOA       | INET6_NTOA(?)           | $#          | INET6_NTOA($#)

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
