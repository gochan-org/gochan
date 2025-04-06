#!/usr/bin/env python3

from argparse import ArgumentParser
from os import path
import os
from glob import glob
from shutil import move

"""
Searches for and deletes orphaned files that don't appear to be attached to a post but weren't deleted
"""

def delete_orphans(board:str, dry_run:bool=False, backup_dir:str="", thread_subdir:str="res", thumb_subdir:str="thumb", upload_subdir:str="src"):
	board_path = path.abspath(board)
	if not path.exists(board) or not path.isdir(board):
		raise FileNotFoundError(f"Board directory '{board_path}' does not exist or is not a directory")

	use_backup = backup_dir != "" and backup_dir is not None
	if use_backup:
		if not path.exists(backup_dir):
			os.mkdir(backup_dir)
		elif not path.isdir(backup_dir):
			raise FileNotFoundError(f"Backup directory '{backup_dir}' already exists but is not a directory")

	print(f"Checking for orphaned files in {board_path} (dry run: {dry_run})")
	res_path = path.abspath(path.join(board, thread_subdir))
	src_path = path.abspath(path.join(board, upload_subdir))
	thumb_path = path.abspath(path.join(board, thumb_subdir))
	if not path.exists(res_path) or not path.isdir(res_path):
		raise FileNotFoundError(f"{res_path} does not exist or is not a directory")
	if not path.exists(src_path) or not path.isdir(src_path):
		raise FileNotFoundError(f"{src_path} does not exist or is not a directory")
	if not path.exists(thumb_path) or not path.isdir(thumb_path):
		raise FileNotFoundError(f"{thumb_path} does not exist or is not a directory")

	# load all HTML files in res_path into a single string, not as efficient as parsing and storing thread info but more portable
	thread_data = ""
	for file in glob(path.join(res_path, "*.html")):
		with open(file, "r", encoding="utf-8") as f:
			thread_data += f.read()

	for root, _, files in os.walk(src_path):
		for file in files:
			if not file in thread_data:
				file_path = path.join(src_path, file)
				remove_orphan(file_path, backup_dir, dry_run)
	for root, _, files in os.walk(thumb_path):
		for file in files:
			if not file in thread_data:
				file_path = path.join(thumb_path, file)
				remove_orphan(file_path, backup_dir, dry_run)

def remove_orphan(file_path:str, backup_dir:str, dry_run:bool=False):
	if backup_dir != None and backup_dir != "":
		backup_path = path.join(backup_dir, path.basename(file_path))
		print(f"Backing up {file_path} to {backup_path}")
		if not dry_run:
			move(file_path, backup_path)
	else:
		print(f"Deleting {file_path}")
		if not dry_run:
			os.remove(file_path)

if __name__ == "__main__":
	parser = ArgumentParser(description="Delete orphaned files (files not associated with a post) in the specified board directory.")
	parser.add_argument("boards",
		nargs="+",
		help="The board directories to check for orphaned files")
	parser.add_argument("--dry-run", "-d",
		action="store_true",
		help="Perform a dry run without deleting any files")
	parser.add_argument("--thread-subdir", "-r",
		type=str,
		default="res",
		help="The subdirectory in the board directory where thread HTML and JSON files are stored (default: 'res')")
	parser.add_argument("--thumb-subdir", "-t",
		type=str,
		default="thumb",
		help="The subdirectory in the board directory where thumbnail files are stored (default: 'thumb')")
	parser.add_argument("--upload-subdir", "-s",
		type=str,
		default="src",
		help="The subdirectory in the board directory where uploads are stored (default: 'src')")
	parser.add_argument("--backup-dir",
		type=str,
		help="If set, the directory to back up files to isntead of deleting them")
	args = parser.parse_args()
	for board in args.boards:
		delete_orphans(board, args.dry_run, args.backup_dir, args.thread_subdir, args.thumb_subdir, args.upload_subdir)