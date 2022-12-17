#!/usr/bin/env python3

# Don't forget to set NewThreadDelay and ReplyDelay to 0 in
# the config (with gochan running) before running, otherwise some posts may be rejected
# by the server

import argparse
from os import path
import re
import unittest
import json
from urllib.request import urlopen
from urllib.parse import urljoin
from selenium import webdriver

from selenium.webdriver.chrome.webdriver import WebDriver
from selenium.webdriver.edge.webdriver import WebDriver # skipcq:  PYL-W0404
from selenium.webdriver.firefox.webdriver import WebDriver # skipcq:  PYL-W0404
from selenium.webdriver.chrome.options import Options  as ChromeOptions
from selenium.webdriver.edge.options import Options as EdgeOptions
from selenium.webdriver.firefox.options import Options as FirefoxOptions
from selenium.webdriver.common.by import By
from selenium.webdriver.remote.webelement import WebElement
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.ui import WebDriverWait

from .util.localstorage import LocalStorage
from .util.qr import openQR,closeQR,qrIsVisible

testingSite = "http://192.168.56.3"
testingName = "Selenium"
testingEmail = "selenium@gochan.org#noko"
testingMessage = "Hello, from Selenium!\n(driver is %s)"
testingSubject = "Selenium post creation"
testingUploadPath = "../../html/notbanned.png"
testingPassword = "12345"
testingBoard = "test"

threadRE = re.compile(r".*/(\S+)/(\d+)(\+50)?.html")
boardTitleRE = re.compile('/(\w+)/ - (.+)')
browser = ""
headless = False
keepOpen = False

def boardExists(board:str):
	req = urlopen(urljoin(testingSite, "boards.json"))
	boards = json.load(req)['boards']
	for entry in boards:
		if entry['board'] == board:
			return True
	return False


def gotoPage(driver: WebDriver, page: str):
	driver.get(urljoin(testingSite, page))


def isLoggedIn(driver: WebDriver):
	gotoPage(driver, "manage?action=login")
	return driver.find_element(by=By.CSS_SELECTOR, value="h1#board-title").text == "Dashboard"


def loginToStaff(driver: WebDriver, username = "admin", pw = "password"):
	if isLoggedIn(driver):
		return
	gotoPage(driver, "manage")
	driver.find_element(by=By.NAME, value="username").send_keys(username)
	driver.find_element(by=By.NAME, value="password").send_keys(pw)
	driver.find_element(by=By.CSS_SELECTOR, value="input[value=Login]").click()


def makePostOnPage(url: str, runner: unittest.TestCase):
	gotoPage(runner.driver, url)
	WebDriverWait(runner.driver, 10).until(
		EC.element_to_be_clickable((By.CSS_SELECTOR, "form#postform input[type=submit]")))

	valProp = runner.driver.find_element(by=By.CSS_SELECTOR, value="form#postform input[type=submit]").get_property("value")
	runner.assertEqual(valProp, "Post")
	form = runner.driver.find_element(by=By.CSS_SELECTOR, value="form#postform")
	sendPost(form,
		testingName,
		testingEmail,
		testingSubject,
		testingMessage % runner.driver.name,
		path.abspath(testingUploadPath),
		testingPassword)
	WebDriverWait(runner.driver, 10).until(
		EC.url_matches(threadRE))


def sendPost(postForm:WebElement, name="", email="", subject="", message="", file="", password=""):
	postForm.find_element(by=By.NAME, value="postname").send_keys(name)
	postForm.find_element(by=By.NAME, value="postemail").send_keys(email)
	postForm.find_element(by=By.NAME, value="postsubject").send_keys(subject)
	postForm.find_element(by=By.NAME, value="postmsg").send_keys(message)
	if file != "":
		postForm.find_element(by=By.NAME, value="imagefile").send_keys(file)
	if password != "":
		passwordInput = postForm.find_element(by=By.CSS_SELECTOR, value="input#postpassword")
		passwordInput.clear()
		passwordInput.send_keys(password)
	postForm.find_element(by=By.CSS_SELECTOR, value="input[type=submit]").click()


def deletePost(driver:WebDriver, postID:int, password:str):
	driver.find_element(by=By.CSS_SELECTOR, value=("input#check%s"%postID)).click()
	if password != "":
		delPasswordInput = driver.find_element(
			by=By.CSS_SELECTOR,
			value="input#delete-password")
		delPasswordInput.clear()
		delPasswordInput.send_keys(password)
	driver.find_element(
		by=By.CSS_SELECTOR,
		value="input[name=delete_btn]").click()
	driver.switch_to.alert.accept()


def expectInputIsEmpty(input):
	def _predicate(driver:WebDriver):
		target = driver.find_element(*input)
		return target.get_property("value") == ""
	return _predicate


class TestRunner(unittest.TestCase):
	def setUp(self):
		if browser == "firefox":
			options = FirefoxOptions()
			options.headless = headless
			self.driver = webdriver.Firefox(options=options)
		elif browser in ("chrome", "chromium"):
			options = ChromeOptions()
			options.headless = headless
			if keepOpen:
				options.add_experimental_option("detach", True)
			self.driver = webdriver.Chrome(options=options)
		elif browser == "edge":
			options = EdgeOptions()
			options.headless = headless
			if keepOpen:
				options.add_experimental_option("detach", True)
			self.driver = webdriver.Edge(options=options)
		else:
			self.fail("Unrecognized --browser option '%s'" % browser)
		return super().setUp()

	def test_qr(self):
		gotoPage(self.driver, testingBoard)
		self.assertIn("/test/ - Testing board", self.driver.title)
		elem = self.driver.find_element(by=By.ID, value="board-subtitle")
		self.assertIn("Board for testing stuff", elem.text)
		openQR(self.driver)
		self.assertTrue(qrIsVisible(self.driver),
			"Confirm that the QR box was properly opened")
		closeQR(self.driver)
		self.assertFalse(qrIsVisible(self.driver),
			"Confirm that the QR box was properly closed")

	def test_localStorage(self):
		gotoPage(self.driver, testingBoard)
		localStorage = LocalStorage(self.driver)
		localStorage["foo"] = "bar"
		self.assertEqual(localStorage.get("foo"), "bar",
			"making sure that LocalStorage.get works")
		self.assertGreater(localStorage.__len__(), 0)

	def test_staffLogin(self):
		loginToStaff(self.driver)
		self.assertEqual(
			self.driver.find_element(by=By.CSS_SELECTOR, value="header h1").text,
			"Dashboard",
			"Testing staff login"
		)

	def test_makeBoard(self):
		if boardExists("seleniumtesting"):
			raise Exception("Board /seleniumtests/ already exists")
		loginToStaff(self.driver)
		gotoPage(self.driver, "manage?action=boards")

		# fill out the board creation form
		self.driver.find_element(by=By.NAME, value="dir").\
			send_keys("seleniumtesting")
		self.driver.find_element(by=By.NAME, value="title").\
			send_keys("Selenium testing")
		self.driver.find_element(by=By.NAME, value="subtitle").\
			send_keys("Board for testing Selenium")
		self.driver.find_element(by=By.NAME, value="description").\
			send_keys("Board for testing Selenium")
		self.driver.find_element(by=By.NAME, value="docreate").click()
		self.driver.switch_to.alert.accept()
		WebDriverWait(self.driver, 10).until(
			EC.presence_of_element_located((
				By.CSS_SELECTOR,
				'div#topbar a[href="/seleniumtesting/"]')))

		makePostOnPage("seleniumtesting", self)

	def test_makeThread(self):
		makePostOnPage(testingBoard, self)

		threadID = threadRE.findall(self.driver.current_url)[0][1]
		cur_url = self.driver.current_url
		deletePost(self.driver, int(threadID), "")
		WebDriverWait(self.driver, 10).until(
			EC.url_changes(cur_url))
		self.assertNotIn("Error :c", self.driver.title, "No errors when we try to delete the post we just made")

	def test_moveThread(self):
		if not boardExists("test2"):
			loginToStaff(self.driver)
			gotoPage(self.driver, "manage?action=boards")

			# fill out the board creation form
			self.driver.find_element(by=By.NAME, value="dir").\
				send_keys("test2")
			self.driver.find_element(by=By.NAME, value="title").\
				send_keys("Testing board #2")
			self.driver.find_element(by=By.NAME, value="subtitle").\
				send_keys("Board for testing thread moving")
			self.driver.find_element(by=By.NAME, value="description").\
				send_keys("Board for testing thread moving")
			self.driver.find_element(by=By.NAME, value="docreate").click()
			self.driver.switch_to.alert.accept()
			WebDriverWait(self.driver, 10).until(
				EC.presence_of_element_located((By.CSS_SELECTOR, 'div#topbar a[href="/test2/"]')))

		gotoPage(self.driver, testingBoard)
		WebDriverWait(self.driver, 10).until(
			EC.element_to_be_clickable((By.CSS_SELECTOR, "form#postform input[type=submit]")))

		form = self.driver.find_element(by=By.CSS_SELECTOR, value="form#postform")
		sendPost(form,
			testingName,
			testingEmail,
			testingSubject,
			testingMessage % self.driver.name,
			path.abspath(testingUploadPath),
			testingPassword)
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
			"/test2/ - Testing board #2",
			"Verify that we properly moved the thread to /test2/")

		deletePost(self.driver, int(threadID), "")
		WebDriverWait(self.driver, 10).until(
			EC.url_changes(cur_url))
		self.assertNotIn("Error :c", self.driver.title,
			"No errors when we try to delete the moved thread")

	def tearDown(self):
		if not keepOpen:
			self.driver.close()
		return super().tearDown()

def startBrowserTests(testBrowser:str, testHeadless=False, testKeepOpen=False, site="", board="", upload="", singleTest = ""):
	global browser
	global testingSite
	global testingBoard
	global testingUploadPath
	global headless
	global keepOpen
	browser = testBrowser
	headless = testHeadless
	keepOpen = testKeepOpen
	if headless:
		keepOpen = False
	if site != "":
		testingSite = site
	if board != "":
		testingBoard = board
	if upload != "":
		testingUploadPath = upload

	print("Using browser %s (headless: %s) on site %s" % (browser, headless, testingSite))
	suite:unittest.TestSuite = None
	if singleTest == "":
		suite = unittest.defaultTestLoader.loadTestsFromTestCase(TestRunner)
	else:
		suite = unittest.defaultTestLoader.loadTestsFromName(singleTest,TestRunner)
	unittest.TextTestRunner(verbosity=3, descriptions=True).run(suite)


def parseArgs(argParser:argparse.ArgumentParser):
	testable_browsers = ("firefox","chrome","chromium", "edge")

	argParser.add_argument("--browser", choices=testable_browsers, required=True)
	argParser.add_argument("--site", default=testingSite,
		help=("Sets the site to be used for testing, defaults to %s" % testingSite))
	argParser.add_argument("--board", default=testingBoard,
		help="Sets the board to be used for testing")
	argParser.add_argument("--headless", action="store_true",
		help="If set, the driver will run without opening windows (overrides --keepopen if it is set)")
	argParser.add_argument("--keepopen", action="store_true",
		help="If set, the browser windows will not automatically close after the tests are complete")
	argParser.add_argument("--singletest", default="",
		help="If specified, only the test method with this name will be run")
	return argParser.parse_args()

if __name__ == "__main__":
	parser = argparse.ArgumentParser(description="Browser testing via Selenium")
	args = parseArgs(parser)
	try:
		startBrowserTests(args.browser, args.headless, args.keepopen, args.site, args.board)
	except KeyboardInterrupt:
		print("Tests interrupted, exiting")
