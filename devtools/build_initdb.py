#!/usr/bin/env python3

from os import path

class macro():
	""" Use a macro like this {exact macro name} """
	def __init__(self, macroname, postgres, mysql):
		self.macroname = macroname
		self.postgres = postgres
		self.mysql = mysql


# macros
macros = [
	macro(
		"serial pk","BIGSERIAL PRIMARY KEY",
		"BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY"),
	macro("fk to serial", "BIGINT", "BIGINT"),
	macro("drop fk", "DROP CONSTRAINT", "DROP FOREIGN KEY")
]


def compileOutIfs(text, flag):
	lines = text.splitlines()
	newText = ""
	doCompile = True
	for i in lines:
		if "#IF" in i:
			doCompile = flag in i
		elif "#ENDIF" in i:
			doCompile = True
		elif doCompile:
			newText += i + "\n"
	return newText


def hasError(text):
	if '{' in text or '}' in text:
		return True
	return False


def dofile(filestart):
	print("building " + filestart + " sql file")
	masterfile = ""
	with open(filestart + "master.sql", 'r') as masterfileIn:
		masterfile = masterfileIn.read()

	postgresProcessed = compileOutIfs(masterfile, "POSTGRES")
	mysqlProcessed = compileOutIfs(masterfile, "MYSQL")

	for item in macros:
		macroCode = "{" + item.macroname + "}"
		postgresProcessed = postgresProcessed.replace(macroCode, item.postgres)
		mysqlProcessed = mysqlProcessed.replace(macroCode, item.mysql)

	error = hasError(postgresProcessed)
	error = error or hasError(mysqlProcessed)

	with open(filestart + "postgres.sql", 'w') as i:
		i.write(postgresProcessed)

	with open(filestart + "mysql.sql", 'w') as i:
		i.write(mysqlProcessed)

	if error:
		input(
			"Error processing macros, files still contain curly braces (might be in comments?)\n",
			"press any key to continue")


if __name__ == "__main__":
	dofile(path.join("..", "initdb_"))
	dofile(path.join("..", "sql", "preapril2020migration", "initdb_"))
	dofile(path.join("..", "sql", "preapril2020migration", "oldDBMigration_"))
