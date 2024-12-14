from os import path

from selenium.webdriver.common.by import By
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.ui import WebDriverWait

from . import SeleniumTestCase
from ..util.posting import make_post, delete_post, send_post, threadRE
from ..util import qr
from ..util.manage import staff_login

class TestPosting(SeleniumTestCase):
	@classmethod
	def setUpClass(cls):
		super().setUpClass()


	def checkBoards(self, board1:bool, board2:bool = False):
		if board1:
			self.assertTrue(self.options.board_exists(self.options.board1), f"Confirming that /{self.options.board1}/ exists")
		if board2:
			self.assertTrue(self.options.board_exists(self.options.board2), f"Confirming that /{self.options.board2}/ exists")


	def test_qr(self):
		self.checkBoards(True)
		self.options.goto_page(self.options.board1)
		elem = self.driver.find_element(by=By.ID, value="board-subtitle")
		self.assertIn("Board for testing stuff", elem.text)
		qr.openQR(self.driver)
		self.assertTrue(qr.qrIsVisible(self.driver),
			"Confirm that the QR box was properly opened")
		qr.closeQR(self.driver)
		self.assertFalse(qr.qrIsVisible(self.driver),
			"Confirm that the QR box was properly closed")

	def test_makeThread(self):
		self.checkBoards(True)
		make_post(self.options, self.options.board1, self)
		threadID = threadRE.findall(self.driver.current_url)[0][1]
		cur_url = self.driver.current_url
		delete_post(self.options, int(threadID), "")
		WebDriverWait(self.driver, 10).until(
			EC.url_changes(cur_url))
		self.assertNotIn("Error :c", self.driver.title, "No errors when we try to delete the post we just made")

	def test_moveThread(self):
		self.checkBoards(True, True)

		self.options.goto_page(self.options.board1)
		WebDriverWait(self.driver, 10).until(
			EC.element_to_be_clickable((By.CSS_SELECTOR, "form#postform input[type=submit]")))

		form = self.driver.find_element(by=By.CSS_SELECTOR, value="form#postform")
		send_post(form,
			self.options.name,
			self.options.email,
			self.options.subject,
			self.options.message % self.driver.name,
			path.abspath(self.options.upload_path),
			self.options.post_password)
		WebDriverWait(self.driver, 10).until(
			EC.url_matches(threadRE))

		cur_url = self.driver.current_url
		threadID = threadRE.findall(cur_url)[0][1]
		self.driver.find_element(
			by=By.CSS_SELECTOR,
			value=("input#check%s"%threadID)).click()
		cur_url = self.driver.current_url
		self.driver.find_element(
			by=By.CSS_SELECTOR,
			value="input[name=move_btn]").click()
		# wait for response to move_btn
		WebDriverWait(self.driver, 10).until(
			EC.title_contains("Move thread #%s" % threadID))

		self.driver.find_element(
			by=By.CSS_SELECTOR,
			value="input[type=submit]").click()
		# wait for response to move request (domove=1)
		WebDriverWait(self.driver, 10).until(
			EC.url_matches(threadRE))

		self.assertEqual(
			self.driver.find_element(
				by=By.CSS_SELECTOR,
				value="h1#board-title").text,
			"/test2/ - Testing board 2",
			"Verify that we properly moved the thread to /test2/")

		form = self.driver.find_element(by=By.CSS_SELECTOR, value="form#postform")
		send_post(form,
			self.options.name,
			self.options.email,
			"",
			"Reply to thread after it was moved",
			path.abspath(self.options.upload_path),
			self.options.post_password)

		delete_post(self.options, int(threadID), "")
		WebDriverWait(self.driver, 10).until(
			EC.url_changes(cur_url))
		self.assertNotIn("Error :c", self.driver.title,
			"No errors when we try to delete the moved thread")

