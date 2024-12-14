from enum import Enum, auto
from selenium.webdriver.common.by import By

from ..options import TestingOptions


class StaffRole(Enum):
	Janitor = auto()
	Moderator = auto()
	Admin = auto()


def is_logged_in(options: TestingOptions):
	options.goto_page("manage/login")
	return options.driver.find_element(by=By.CSS_SELECTOR, value="h1#board-title").text == "Dashboard"


def staff_login(options: TestingOptions, role: StaffRole):
	options.goto_page("manage/logout")
	options.goto_page("manage")
	username = ""
	password = ""
	match role:
		case StaffRole.Janitor:
			username = options.janitor_username
			password = options.janitor_password
		case StaffRole.Moderator:
			username = options.moderator_username
			password = options.moderator_password
		case StaffRole.Admin:
			username = options.admin_username
			password = options.admin_password
		case _:
			raise ValueError(f"Invalid role: {role}")

	options.driver.find_element(by=By.NAME, value="username").send_keys(username)
	options.driver.find_element(by=By.NAME, value="password").send_keys(password)
	options.driver.find_element(by=By.CSS_SELECTOR, value="input[value=Login]").click()


def staff_logout(options: TestingOptions, clear_sessions:bool = False):
	options.goto_page("manage/clearmysessions" if clear_sessions else "manage/logout")