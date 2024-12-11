import json
from urllib.parse import urljoin
from urllib.request import urlopen

from selenium import webdriver
from selenium.webdriver.remote.webdriver import WebDriver
from selenium.webdriver.chrome.options import Options  as ChromeOptions
from selenium.webdriver.edge.options import Options as EdgeOptions
from selenium.webdriver.firefox.options import Options as FirefoxOptions

default_site = "http://192.168.56.3"
default_name = "Selenium"
default_email = "selenium@gochan.org#noko"
default_message = "Hello, from Selenium!\n(driver is %s)"
default_subject = "Selenium post creation"
default_upload = "./html/static/notbanned.png"
default_password = "12345"
default_board1 = "test"
default_board2 = "selenium2"
default_staff_username = "admin"
default_staff_password = "password"

class TestingOptions:
	browser: str
	driver: WebDriver
	headless: bool
	keep_open: bool
	site: str
	board1: str
	board2: str
	name: str
	email: str
	subject: str
	message: str
	upload_path: str
	password: str
	staff_username: str
	staff_password: str
	def __init__(self, browser: str, headless=False, keep_open=False):
		self.browser = browser
		self.headless = headless
		self.keep_open = keep_open
		self.site = default_site
		self.board1 = default_board1
		self.board2 = default_board2
		self.name = default_name
		self.email = default_email
		self.subject = default_subject
		self.message = default_message
		self.upload_path = default_upload
		self.password = default_password
		self.staff_username = default_staff_username
		self.staff_password = default_staff_password

		match browser:
			case "firefox":
				options = FirefoxOptions()
				options.headless = headless
				self.driver = webdriver.Firefox(options=options)

			case "chrome":
				options = ChromeOptions()
				options.headless = headless
				if self.keep_open:
					options.add_experimental_option("detach", True)
				self.driver = webdriver.Chrome(options=options)

			case "chromium":
				options = ChromeOptions()
				options.headless = headless
				if self.keep_open:
					options.add_experimental_option("detach", True)
				self.driver = webdriver.Chrome(options=options)

			case "edge":
				options = EdgeOptions()
				options.headless = headless
				if keep_open:
					options.add_experimental_option("detach", True)
				self.driver = webdriver.Edge(options=options)

			case _:
				raise ValueError("Unrecognized browser argument %s" % browser)


	def board_exists(self, board: str):
		req = urlopen(urljoin(default_site, "boards.json"))  # skipcq: BAN-B310
		boards = json.load(req)['boards']
		for entry in boards:
			if entry['board'] == board:
				return True
		return False

	def goto_page(self, page: str):
		self.driver.get(urljoin(self.site, page))


	def close(self):
		if not self.keep_open:
			self.driver.close()
