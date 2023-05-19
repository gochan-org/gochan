from os import path
import re
import unittest

from selenium.webdriver.common.by import By
from selenium.webdriver.remote.webelement import WebElement
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.ui import WebDriverWait

from ..options import TestingOptions

threadRE = re.compile(r".*/(\S+)/(\d+)(\+50)?.html")


def send_post(postForm:WebElement, name="", email="", subject="", message="", file="", password=""):
	postForm.find_element(by=By.NAME, value="postname").clear()
	postForm.find_element(by=By.NAME, value="postname").send_keys(name)
	postForm.find_element(by=By.NAME, value="postemail").clear()
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


def make_post(options: TestingOptions, url: str, runner: unittest.TestCase):
	options.goto_page(url)
	WebDriverWait(options.driver, 10).until(
		EC.element_to_be_clickable((By.CSS_SELECTOR, "form#postform input[type=submit]")))

	valProp = options.driver.find_element(by=By.CSS_SELECTOR, value="form#postform input[type=submit]").get_property("value")
	runner.assertEqual(valProp, "Post")
	form = options.driver.find_element(by=By.CSS_SELECTOR, value="form#postform")
	send_post(form,
		options.name,
		options.email,
		options.subject,
		options.message % options.name,
		path.abspath(options.upload_path),
		options.password)
	WebDriverWait(options.driver, 10).until(
		EC.url_matches(threadRE))

def delete_post(options: TestingOptions, postID:int, password:str):
	options.driver.find_element(by=By.CSS_SELECTOR, value=("input#check%s"%postID)).click()
	if password != "":
		delPasswordInput = options.driver.find_element(
			by=By.CSS_SELECTOR,
			value="input#delete-password")
		delPasswordInput.clear()
		delPasswordInput.send_keys(password)
	options.driver.find_element(
		by=By.CSS_SELECTOR,
		value="input[name=delete_btn]").click()
	options.driver.switch_to.alert.accept()
