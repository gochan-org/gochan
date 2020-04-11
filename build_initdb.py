class macro():
	def __init__(self, macroname, postgres, sqlite, mysql):
		self.macroname = macroname
		self.postgres = postgres
		self.sqlite = sqlite
		self.mysql = mysql
	
# macros
macros = [
	macro("serial pk", "bigserial PRIMARY KEY", "INTEGER PRIMARY KEY AUTOINCREMENT", "bigint NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY"),
	macro("fk to serial", "bigint", "INTEGER", "bigint")
]
masterfile = open("initdb_master.sql").read()

postgresProcessed = masterfile
sqliteProcessed = masterfile
mysqlProcessed = masterfile

for item in macros:
	macroCode = "{" + item.macroname + "}"
	postgresProcessed = postgresProcessed.replace(macroCode, item.postgres)
	mysqlProcessed = mysqlProcessed.replace(macroCode, item.mysql)
	sqliteProcessed = sqliteProcessed.replace(macroCode, item.sqlite)
	
def hasError(text):
	if '{' in text or '}' in text:
		return True
		
error = hasError(postgresProcessed)
error = error or hasError(mysqlProcessed)
error = error or hasError(sqliteProcessed)

open("initdb_postgres.sql", 'w').write(postgresProcessed)
open("initdb_mysql.sql", 'w').write(mysqlProcessed)
open("initdb_sqlite3.sql", 'w').write(sqliteProcessed)
	
if error:
	input("Error processing macros, files still contain curly braces (might be in comments?), press any key to continue")