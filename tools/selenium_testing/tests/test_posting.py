from os import path

from selenium.webdriver.common.by import By
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.ui import WebDriverWait

from . import SeleniumTestCase
from ..util.posting import make_post, delete_post, send_post, threadRE
from ..util import qr


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
		board_info = self.options.board_info(self.options.board1)
		self.options.goto_page(self.options.board1)
		elem = self.driver.find_element(by=By.ID, value="board-subtitle")
		self.assertIn(board_info['meta_description'], elem.text, "Verify that we're on the correct board")
		qr.openQR(self.driver)
		self.assertTrue(qr.qrIsVisible(self.driver), "Confirm that the QR box was properly opened")
		qr.closeQR(self.driver)
		self.assertFalse(qr.qrIsVisible(self.driver), "Confirm that the QR box was properly closed")


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
		WebDriverWait(self.driver, 10).until(EC.url_matches(threadRE))

		cur_url = self.driver.current_url
		threadID = threadRE.findall(cur_url)[0][1]
		self.driver.find_element(by=By.CSS_SELECTOR, value=("input#check%s"%threadID)).click()
		cur_url = self.driver.current_url
		self.driver.find_element(by=By.CSS_SELECTOR, value="input[name=move_btn]").click()
		# wait for response to move_btn
		WebDriverWait(self.driver, 10).until(EC.title_contains("Move thread #%s" % threadID))

		self.driver.find_element(by=By.CSS_SELECTOR, value="input[type=submit]").click()
		# wait for response to move request (domove=1)
		WebDriverWait(self.driver, 10).until(
			EC.url_matches(threadRE))

		expected_title = self.options.board_info(self.options.board2)['title']
		self.assertEqual(
			self.driver.find_element(by=By.CSS_SELECTOR, value="h1#board-title").text,
			f"/{self.options.board2}/ - {expected_title}",
			"Verify that we properly moved the thread to /test2/"
		)

		form = self.driver.find_element(by=By.CSS_SELECTOR, value="form#postform")
		send_post(form,
			self.options.name,
			self.options.email,
			"",
			"Reply to thread after it was moved",
			path.abspath(self.options.upload_path),
			self.options.post_password)

		delete_post(self.options, int(threadID), "")
		WebDriverWait(self.driver, 10).until(EC.url_changes(cur_url))
		self.assertNotIn("Error :c", self.driver.title, "No errors when we try to delete the moved thread")

	def test_cyclic(self):
		self.assertTrue(self.options.board_exists(self.options.cyclic_board), f"Confirming that /{self.options.cyclic_board}/ exists")

		self.options.goto_page(self.options.cyclic_board)
		WebDriverWait(self.driver, 10).until(
			EC.element_to_be_clickable((By.CSS_SELECTOR, "form#postform input[type=submit]")))
		form = self.driver.find_element(by=By.CSS_SELECTOR, value="form#postform")
		form.find_element(by=By.NAME, value="cyclic").click()
		send_post(form,
			self.options.name,
			"noko",
			"Cyclic thread test",
			"Cyclic thread OP",
			path.abspath(self.options.upload_path),
			self.options.post_password)
		WebDriverWait(self.driver, 10).until(EC.url_matches(threadRE))

		for r in range(self.options.cyclic_count + 2):
			form = self.driver.find_element(by=By.CSS_SELECTOR, value="form#postform")
			send_post(form,
				self.options.name,
				"noko",
				"",
				f"Reply {r+1}",
				path.abspath(self.options.upload_path),
				self.options.post_password)
			WebDriverWait(self.driver, 10).until(EC.url_matches(threadRE))

		# go to the thread and make sure that the first two replies are pruned
		cur_url = self.driver.current_url
		threadID = threadRE.findall(cur_url)[0][1]
		replies = self.driver.find_elements(by=By.CSS_SELECTOR, value="div.reply")
		self.assertEqual(len(replies), self.options.cyclic_count, "Verify that the cyclic thread has the correct number of replies")
		self.assertEqual(replies[0].find_element(by=By.CSS_SELECTOR, value="div.post-text").text, "Reply 3", "Verify that the first reply is the third post")
		delete_post(self.options, int(threadID), self.options.post_password)
