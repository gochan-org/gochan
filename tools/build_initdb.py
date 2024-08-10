#!/usr/bin/env python3

import argparse
from os import path

class macro():
	""" Use a macro like this {exact macro name} """
	def __init__(self, macroname, postgres, mysql, sqlite3):
		self.macroname = macroname
		self.postgres = postgres
		self.mysql = mysql
		self.sqlite3 = sqlite3


# macros
macros = [
	macro(
		"serial pk","BIGSERIAL PRIMARY KEY",
		"BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY",
		"INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL"),
	macro("fk to serial", "BIGINT", "BIGINT", "BIGINT"),
	macro("drop fk", "DROP CONSTRAINT", "DROP FOREIGN KEY", "DROP CONSTRAINT"),
	macro("inet", "INET", "VARBINARY(16)", "VARCHAR(45)")
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
	with open(filestart + "master.sql", 'r') as masterfileIn:  # skipcq: PTC-W6004
		masterfile = masterfileIn.read()

	postgresProcessed = compileOutIfs(masterfile, "POSTGRES")
	mysqlProcessed = compileOutIfs(masterfile, "MYSQL")
	sqlite3Processed = compileOutIfs(masterfile, "SQLITE3")

	for item in macros:
		macroCode = "{" + item.macroname + "}"
		postgresProcessed = postgresProcessed.replace(macroCode, item.postgres)
		mysqlProcessed = mysqlProcessed.replace(macroCode, item.mysql)
		sqlite3Processed = sqlite3Processed.replace(macroCode, item.sqlite3)

	error = hasError(postgresProcessed)
	error = error or hasError(mysqlProcessed)

	with open(filestart + "postgres.sql", 'w') as i:
		i.write(postgresProcessed)

	with open(filestart + "sqlite3.sql", 'w') as i:
		i.write(sqlite3Processed)

	with open(filestart + "mysql.sql", 'w') as i:
		i.write(mysqlProcessed)

	if error:
		input(
			"Error processing macros, files still contain curly braces (might be in comments?)\n",
			"press any key to continue")


if __name__ == "__main__":
	parser = argparse.ArgumentParser(description="gochan build script")
	dofile(path.join("..", "sql", "initdb_"))
	parser.add_argument("--preapril2020",
			action="store_true",
			help="Also build the legacy (pre-April 2020 migration) database schema used for testing gochan-migrate.")
	args = parser.parse_args()
	if args.preapril2020:
		dofile(path.join("..", "sql", "preapril2020migration", "oldDBMigration_"))
