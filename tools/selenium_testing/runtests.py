#!/usr/bin/env python3

# Don't forget to set NewThreadDelay and ReplyDelay to 0 in
# the config (with gochan running) before running, otherwise some posts may be rejected
# by the server

from argparse import ArgumentParser
import unittest

from .options import (TestingOptions, default_site, default_name, default_email, default_message, default_subject,
	default_upload, default_post_password, default_board1, default_board2, default_staff_board, default_admin_username,
	default_admin_password, default_moderator_username, default_moderator_password, default_janitor_username, default_janitor_password)
from .tests import SeleniumTestCase
from .tests.test_mgmt import TestManageActions
from .tests.test_posting import TestPosting

options:TestingOptions = None

def start_tests(dict_options:dict[str,object]=None):
	global options
	options = TestingOptions.from_dict(dict_options)
	single_test = dict_options.get("single_test", "")
	print("Using browser %s (headless: %s) on site %s" % (options.browser, options.headless, options.site))
	if single_test == "":
		suite = unittest.suite.TestSuite()
		SeleniumTestCase.add(suite, options, TestPosting)
		SeleniumTestCase.add(suite, options, TestManageActions)
		unittest.TextTestRunner(verbosity=3, descriptions=True).run(suite)
	else:
		import importlib.util

		rindex = -1
		try:
			rindex =  single_test.rindex(":")
		except ValueError:
			raise ValueError("Single test must be of the format /path/to/test.py:TestCaseClass")
		test_location = single_test[:rindex]
		test_class = single_test[rindex+1:]
		if test_location == "" or test_class == "":
			raise ValueError("Single test must be of the format /path/to/test.py:TestCaseClass")
		print("Single test module location:", test_location)
		print("Single test class:", test_class)

		spec = importlib.util.spec_from_file_location(test_class, test_location)
		module = importlib.util.module_from_spec(spec)
		module.__package__ = "tools.selenium_testing.tests"
		spec.loader.exec_module(module)

		suite = unittest.suite.TestSuite()
		SeleniumTestCase.add(suite, options, module.__dict__[test_class])
		unittest.TextTestRunner(verbosity=3, descriptions=True).run(suite)
	options.close()


def close_tests():
	if options is not None:
		options.close()


def setup_selenium_args(parser:ArgumentParser):
	testable_browsers = ("firefox","chrome","chromium", "edge")

	parser.add_argument("--browser", choices=testable_browsers, required=True)
	parser.add_argument("--headless", action="store_true",
		help="If set, the driver will run without opening windows (overrides --keep-open if it is set)")
	parser.add_argument("--keep-open", action="store_true",
		help="If set, the browser windows will not automatically close after the tests are complete")
	parser.add_argument("--site", default=default_site,
		help=("Sets the site to be used for testing, defaults to %s" % default_site))
	parser.add_argument("--board1", default=default_board1,
		help="Sets the main board to be used for testing. It must already be created or tests that use it will fail")
	parser.add_argument("--board2", default=default_board2,
		help="Sets the secondary board to be used for testing. It must already be created or tests that use it will fail")
	parser.add_argument("--staff-board", default=default_staff_board,
		help="Sets the board to be used for testing board management operations. It does not need to exist before testing")
	parser.add_argument("--name", default=default_name,
		help="Sets the name to be used when posting")
	parser.add_argument("--email", default=default_email,
		help="Sets the email to be used when posting")
	parser.add_argument("--subject", default=default_subject,
		help="Sets the subject to be used when posting")
	parser.add_argument("--message", default=default_message,
		help="Sets the message to be used when posting")
	parser.add_argument("--upload_path", default=default_upload,
		help="Sets the file to be used when posting")
	parser.add_argument("--post_password", default=default_post_password,
		help="Sets the post password")
	parser.add_argument("--admin_username", default=default_admin_username,
		help="Sets the username to be used when logging in as an admin. Admin tests will fail if this does not exist")
	parser.add_argument("--admin_password", default=default_admin_password,
		help="Sets the password to be used when logging in as an admin. Admin tests will fail if this does not exist")
	parser.add_argument("--moderator_username", default=default_moderator_username,
		help="Sets the username to be used when logging in as a moderator. Moderator tests will fail if this does not exist")
	parser.add_argument("--moderator_password", default=default_moderator_password,
		help="Sets the password to be used when logging in as a moderator. Moderator tests will fail if this does not exist")
	parser.add_argument("--janitor_username", default=default_janitor_username,
		help="Sets the username to be used when logging in as a janitor. Janitor tests will fail if this does not exist")
	parser.add_argument("--janitor_password", default=default_janitor_password,
		help="Sets the password to be used when logging in as a janitor. Janitor tests will fail if this does not exist")
	parser.add_argument("--single-test", default="",
		help="If specified, only the test method with this name will be run")
	
	return parser.parse_args()


if __name__ == "__main__":
	parser = ArgumentParser(description="Browser testing via Selenium")
	args = setup_selenium_args(parser)
	
	try:
		start_tests(args.__dict__)
	except KeyboardInterrupt:
		print("Tests interrupted by KeyboardInterrupt, exiting")
		close_tests()
