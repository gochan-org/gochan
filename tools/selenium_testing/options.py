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
default_upload = "html/static/notbanned.png"
default_post_password = "12345"
default_board1 = "test"
default_board2 = "test2"
default_staff_board = "selenium"
default_admin_username = "selenium_admin"
default_admin_password = "password"
default_moderator_username = "selenium_mod"
default_moderator_password = "password"
default_janitor_username = "selenium_janitor"
default_janitor_password = "password"


class TestingOptions:
	browser: str
	driver: WebDriver
	headless: bool
	__keep_open: bool
	site: str
	board1: str
	board2: str
	staff_board: str
	name: str
	email: str
	subject: str
	message: str
	upload_path: str
	post_password: str
	admin_username: str
	admin_password: str
	moderator_username: str
	moderator_password: str
	janitor_username: str
	janitor_password: str

	@property
	def keep_open(self):
		return self.__keep_open


	@keep_open.setter
	def keep_open(self, ko:bool):
		self.__keep_open = ko and not self.headless


	@staticmethod
	def from_dict(src_dict:dict[str,object]):
		options = TestingOptions(src_dict.get("browser", ""), src_dict.get("headless", False), src_dict.get("keep_open", False))
		options.site = src_dict.get("site", default_site)
		options.board1 = src_dict.get("board1", default_board1)
		options.board2 = src_dict.get("board2", default_board2)
		options.staff_board = src_dict.get("staff_board", default_staff_board)
		options.name = src_dict.get("name", default_name)
		options.email = src_dict.get("email", default_email)
		options.subject = src_dict.get("subject", default_subject)
		options.message = src_dict.get("message", default_message)
		options.upload_path = src_dict.get("upload", default_upload)
		options.post_password = src_dict.get("post_password", default_post_password)
		options.admin_username = src_dict.get("admin_username", default_admin_username)
		options.admin_password = src_dict.get("admin_password", default_admin_password)
		options.moderator_username = src_dict.get("moderator_username", default_moderator_username)
		options.moderator_password = src_dict.get("moderator_password", default_moderator_password)
		options.janitor_username = src_dict.get("janitor_username", default_janitor_username)
		options.janitor_password = src_dict.get("janitor_password", default_janitor_password)
		return options


	def __init__(self, browser: str, headless=False, keep_open=False):
		self.browser = browser
		self.headless = headless
		self.keep_open = keep_open
		self.site = default_site
		self.board1 = default_board1
		self.board2 = default_board2
		self.staff_board = default_staff_board
		self.name = default_name
		self.email = default_email
		self.subject = default_subject
		self.message = default_message
		self.upload_path = default_upload
		self.post_password = default_post_password
		self.admin_username = default_admin_username
		self.admin_password = default_admin_password

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
			case ""|None:
				raise ValueError("browser argument is required")
			case _:
				raise ValueError("Unrecognized browser argument %s" % browser)


	def board_exists(self, board: str):
		boards_json_url = urljoin(self.site, "boards.json")
		if not boards_json_url.lower().startswith("http"):
			raise ValueError(f"Invalid site URL (expected http): {self.site}")
		req = urlopen(boards_json_url)
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
