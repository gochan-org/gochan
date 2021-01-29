from os import path
#
# Use a macro like this {exact macro name}
#
class macro():
	def __init__(self, macroname, postgres, mysql):
		self.macroname = macroname
		self.postgres = postgres
		self.mysql = mysql
	
# macros
macros = [
	macro("serial pk", "BIGSERIAL PRIMARY KEY", "BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY"),
	macro("fk to serial", "BIGINT", "BIGINT"),
	macro("drop fk", "DROP CONSTRAINT", "DROP FOREIGN KEY")
]	

def compileOutIfs(text, flag):
	lines = text.splitlines()
	newText = ""
	compile = True
	for i in lines:
		if "#IF" in i:
			if flag in i:
				compile = True
			else:
				compile = False
		elif "#ENDIF" in i:
				compile = True
		elif compile:
			newText += i + "\n"
	return newText

def dofile(filestart):
	print("building " + filestart + " sql file")
	masterfileIn = open(filestart + "master.sql", 'r')
	masterfile = masterfileIn.read()
	masterfileIn.close()

	postgresProcessed = compileOutIfs(masterfile, "POSTGRES")
	mysqlProcessed = compileOutIfs(masterfile, "MYSQL")

	for item in macros:
		macroCode = "{" + item.macroname + "}"
		postgresProcessed = postgresProcessed.replace(macroCode, item.postgres)
		mysqlProcessed = mysqlProcessed.replace(macroCode, item.mysql)
		
	def hasError(text):
		if '{' in text or '}' in text:
			return True
			
	error = hasError(postgresProcessed)
	error = error or hasError(mysqlProcessed)

	i = open(filestart + "postgres.sql", 'w')
	i.write(postgresProcessed)
	i.close()

	i = open(filestart + "mysql.sql", 'w')
	i.write(mysqlProcessed)
	i.close()
		
	if error:
		input("Error processing macros, files still contain curly braces (might be in comments?), press any key to continue")
	
dofile(path.join("..", "initdb_"))
dofile(path.join("..", "sql", "preapril2020migration", "initdb_"))
dofile(path.join("..", "sql", "preapril2020migration", "oldDBMigration_"))