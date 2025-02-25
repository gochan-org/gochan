from urllib.request import Request, urlopen
from urllib.parse import urljoin
import json

from selenium.webdriver.common.by import By
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.ui import WebDriverWait
from selenium.common.exceptions import TimeoutException

from . import SeleniumTestCase
from ..options import active_options
from ..util.manage import staff_login, StaffRole, staff_logout

actions:list[dict[str,object]] = []
assertion_message = "Confirm that we got at least one access denied page"

class TestStaffPermissions(SeleniumTestCase):
	@classmethod
	def setUpClass(cls):
		options = active_options()
		options.goto_page("manage/clearmysessions")
		options.goto_page("manage/dashboard")
		options.driver.find_element(by=By.NAME, value="username").send_keys(options.admin_username)
		options.driver.find_element(by=By.NAME, value="password").send_keys(options.admin_password)
		options.driver.find_element(by=By.CSS_SELECTOR, value="input[value=Login]").click()
		cookie:str = options.driver.get_cookie("sessiondata")['value']
		req = Request(urljoin(options.site, "manage/actions"))
		# modern browsers add pretty printing to JSON so we need to pass the session cookie to a request to get the raw action list data
		req.add_header("Cookie", f"sessiondata={cookie}")
		with urlopen(req) as resp: # skipcq: BAN-B310
			global actions
			actions = json.load(resp)


	def tearDown(self):
		staff_logout(self.options, True)


	def validate_action_access(self, action:dict[str,object], staff_perm:int):
		denied = False
		if action['perms'] > staff_perm and action['jsonOutput'] < 2:
			self.options.goto_page(f"manage/{action['id']}")
			WebDriverWait(self.driver, 10).until(
				EC.text_to_be_present_in_element((By.TAG_NAME, "p"), "You do not have permission to access this page"),
				f"Timed out while waiting to load access denied error for manage action {action['id']}")
			denied = True
		elif action['id'] not in ("logout", "clearmysessions") and action['jsonOutput'] < 2:
			self.options.goto_page(f"manage/{action['id']}")
			WebDriverWait(self.driver, 10).until(
				EC.text_to_be_present_in_element((By.CSS_SELECTOR, "h1#board-title"), action['title']),
				f"Timed out while waiting to load manage action {action['id']}")
		return denied


	def test_mod_permissions(self):
		staff_login(self.options, StaffRole.Moderator)
		num_denied = 0
		for action in actions:
			if self.validate_action_access(action, 2):
				num_denied += 1
		self.assertGreater(num_denied, 0, assertion_message)


	def test_janitor_permissions(self):
		staff_login(self.options, StaffRole.Janitor)
		num_denied = 0
		for action in actions:
			if self.validate_action_access(action, 1):
				num_denied += 1
		self.assertGreater(num_denied, 0, assertion_message)
