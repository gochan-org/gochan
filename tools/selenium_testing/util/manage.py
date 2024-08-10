from selenium.webdriver.common.by import By

from ..options import TestingOptions

def is_logged_in(options: TestingOptions):
	options.goto_page("manage/login")
	return options.driver.find_element(by=By.CSS_SELECTOR, value="h1#board-title").text == "Dashboard"


def staff_login(options: TestingOptions):
	if is_logged_in(options):
		return
	options.goto_page("manage")
	options.driver.find_element(by=By.NAME, value="username").send_keys(options.staff_username)
	options.driver.find_element(by=By.NAME, value="password").send_keys(options.staff_password)
	options.driver.find_element(by=By.CSS_SELECTOR, value="input[value=Login]").click()

