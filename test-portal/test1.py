#!/usr/bin/env python

# Copyright 2018 US Ignite
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#
# This code is adapted from the Google App Engine Guestbook Example at
# https://cloud.google.com/appengine/docs/standard/python/getting-started/creating-guestbook
# and uses the NYU style guidelines at:
# https://ssl.engineering.nyu.edu/collaborate
# 


# [START imports]
import os
import urllib

from google.appengine.api import users
from google.appengine.ext import ndb

import jinja2
import webapp2
import logging

JINJA_ENVIRONMENT = jinja2.Environment(
  loader=jinja2.FileSystemLoader(os.path.dirname(__file__)),
  extensions=['jinja2.ext.autoescape'],
  autoescape=True)
# [END imports]
#
# A User of the system.  Note that passwords are not stored here, and ATM the only
# identifying information is the email address.
#
class User(ndb.Model):
  """A model for representing a User of Sundew
  Note that passwords are not stored here, and ATM the only identifying information is the email address.
  We store:
  The email
  A boolean which tells us whether the user has agreed to the AUP.  The user cannot use
  Sundew until he has agreed to the AUP
  """
  email = ndb.StringProperty(indexed = True, required = True)
  agreed_to_AUP = ndb.BooleanProperty('ok', indexed=False, required = True, default = False)

def create_or_find_user(user_email):
  user_record = User.query(User.email == user_email).fetch()
  if len(user_record) < 1:
    user_record = User(email=user_email, agreed_to_AUP = False)
    user_record.put()
    return user_record
  else:
    return user_record[0]

def get_current_user_nickname_and_email():
  user = users.get_current_user()
  if (not user):
    return None, None
  if (user.email()):
    return user.nickname(), user.email()
  return user.nickname(), user.nickname()



# [START main_page]
class MainPage(webapp2.RequestHandler):

  def get(self):
    all_users = [('rick@mcgeer.com', False), ('rick.mcgeer@us-ignite.org', True), ('discount.yoyos@gmail.com', True), ('jcappos@nyu.edu', True), ('matthew.c.hemmings@gmail.com',True)]
    
    
    values = {"user": "None", "email": "None", "display_next": False}
    nickname, email = get_current_user_nickname_and_email()

    if nickname:
      values['display_next'] = True
  
      values['user'] = email
      values["url"] = users.create_logout_url(self.request.uri)
      values["url_linktext"] = 'Logout'
      record = create_or_find_user(email)
      if record.agreed_to_AUP:
        values["next_url"] = "/dashboard"
        values["next_url_linktext"] = "Dashboard"
      else:
        values["next_url"] = "/aup"
        values["next_url_linktext"] = "Agree to the AUP"
    else:
      values["url"] = users.create_login_url(self.request.uri)
      values["url_linktext"] = 'Login'

    template = JINJA_ENVIRONMENT.get_template('index.html')
    user_query = User.query()
    stored_users = user_query.fetch()
    values['users'] = stored_users    
    self.response.write(template.render(values))
    if len(stored_users) < len(all_users):
      next_user = all_users[len(stored_users)]
      new_user  = User(email=next_user[0], agreed_to_AUP = next_user[1])
      new_user.put()
    else:
      ndb.delete_multi(User.query().fetch(keys_only=True))
    
class Dashboard(webapp2.RequestHandler):
  def get(self):
    nickname, email = get_current_user_nickname_and_email()
    formatted_email = email.replace('.','-').replace('@','-')
    query_url = 'http://head.sundew.ch-geni-net.instageni.washington.edu:8181/?user=' \
      + formatted_email
    response = urllib.urlopen(query_url).read()
    self.response.write('Dashboard for ' + email + '<BR><HR><BR>' + response.replace('\n','<BR>'))


class AUP(webapp2.RequestHandler):
  def get(self):
    nickname, email = get_current_user_nickname_and_email()
    template = JINJA_ENVIRONMENT.get_template('aup.html')
    self.response.write(template.render(email=email))

class AUP_AGREE(webapp2.RequestHandler):
  def post(self):
    email = self.request.get('email')
    record = create_or_find_user(email)
    record.agreed_to_AUP = True
    record.put()
    nickname, email = get_current_user_nickname_and_email()
    response = urllib.urlopen('http://head.sundew.ch-geni-net.instageni.washington.edu:8181/?user=' + email.replace('.','-').replace('@','-')).read()
    self.response.write('Dashboard for ' + email + '<BR><HR><BR>' + response.replace('\n','<BR>'))

app = webapp2.WSGIApplication([
  ('/', MainPage),
  ('/dashboard', Dashboard),
  ('/aup', AUP),
  ('/aup_agree', AUP_AGREE)
  ], debug=True)
# [END app]
