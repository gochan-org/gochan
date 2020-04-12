from os import path

class macro():
	def __init__(self, macroname, postgres, sqlite, mysql):
		self.macroname = macroname
		self.postgres = postgres
		self.sqlite = sqlite
		self.mysql = mysql
	
# macros
macros = [
	macro("serial pk", "BIGSERIAL PRIMARY KEY", "INTEGER PRIMARY KEY AUTOINCREMENT", "BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY"),
	macro("fk to serial", "BIGINT", "INTEGER", "BIGINT")
]
masterfileIn = open(path.join("..", "initdb_master.sql"), 'r')
masterfile = masterfileIn.read()
masterfileIn.close()

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

i = open(path.join("..", "initdb_postgres.sql"), 'w')
i.write(postgresProcessed)
i.close()

i = open(path.join("..", "initdb_mysql.sql"), 'w')
i.write(mysqlProcessed)
i.close()

i = open(path.join("..", "initdb_sqlite3.sql"), 'w')
i.write(sqliteProcessed)
i.close()
	
if error:
	input("Error processing macros, files still contain curly braces (might be in comments?), press any key to continue")