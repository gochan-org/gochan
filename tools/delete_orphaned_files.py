#!/usr/bin/env python3

from argparse import ArgumentParser
from os import path
import os
import glob

"""
Searches for and deletes orphaned files that don't appear to be attached to a post but weren't deleted
"""

def delete_orphans(board:str, dry_run:bool=False, thread_subdir:str="res", thumb_subdir:str="thumb", upload_subdir:str="src"):
	board_path = path.abspath(board)
	if not path.exists(board) or not path.isdir(board):
		raise FileNotFoundError(f"Board directory '{board_path}' does not exist or is not a directory")

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
	for root, _, files in os.walk(res_path):
		for file in files:
			with open(path.join(root, file), "r", encoding="utf-8") as f:
				thread_data += f.read()

	for root, _, files in os.walk(src_path):
		for file in files:
			if not file in thread_data:
				file_path = path.join(src_path, file)
				print(f"Deleting {file_path}")
				if not dry_run:
					os.remove(file_path)
	for root, _, files in os.walk(thumb_path):
		for file in files:
			if not file in thread_data:
				file_path = path.join(thumb_path, file)
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
	args = parser.parse_args()
	for board in args.boards:
		delete_orphans(board, args.dry_run)