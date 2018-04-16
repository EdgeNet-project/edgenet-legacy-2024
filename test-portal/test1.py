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
import sys
# import cloudstorage
import datetime
import json

from google.appengine.api import app_identity
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
# get the default bucket
#
bucket_name = os.environ.get('BUCKET_NAME', app_identity.get_default_gcs_bucket_name())

#
# A User of the system.  Note that passwords are not stored here, and ATM the only
# identifying information is the email address.
#
class User(ndb.Model):
  """A model for representing a User of Sundew
  Note that passwords are not stored here, and ATM the only identifying information is the email address.
  We store:
  The email
  A namespace name; this is a modified form of the email which is 
  A boolean which tells us whether the user has agreed to the AUP.  The user cannot use
  Sundew until he has agreed to the AUP.
  A boolean which tells us if the user has been approved
  A boolean which tells us if the user is an administrator
  """
  email = ndb.StringProperty(indexed = True, required = True)
  namespace = ndb.StringProperty(indexed = True, required = True)
  agreed_to_AUP = ndb.BooleanProperty('ok', indexed=False, required = True, default = False)
  approved = ndb.BooleanProperty('approved', indexed=False, required = True, default = False)
  administrator = ndb.BooleanProperty('admin', indexed=False, required = True, default = False)


#
# Node List snapshots.  These are taken every few minutes and stored in the DB.  The current node list 
# is rendered on the welcome and dashboard pages
#

class NodeList(ndb.Model):
  """
  A model for representing the node in the system. This is obtained from the head node, and is typically done
  every few minutes as a CRON job.  Each entry has:
  1. time: a DateTimeProperty which shows when this was taken
  2. active: the last fetch.  This should only be true for one entry
  3. nodes: an array of nodes represented as a JSON string
  """
  time = ndb.DateTimeProperty(indexed = True, required = True)
  active = ndb.BooleanProperty(indexed = True, required = True, default = False)
  nodes = ndb.JsonProperty(indexed = False, required = True)

#
# get the current nodelist record from the db.  This is used in a couple of places, so it's broken out
# as a separate method.
#
def get_current_nodelist_record():
  active =  NodeList.query(NodeList.active == True).fetch()
  if len(active) == 0:
    return None
  return active[0]

#
# Store a list of nodes as the current nodelist, with a timestamp to show it's current
#
def store_new_nodelist(nodelist):
  active = get_current_nodelist_record()
  # Note that the current active record is no longer the active record
  if (active): 
    active.active = False
    active.put()
  record = NodeList(active = True, time = datetime.datetime.now(), nodes = json.dumps(nodelist))
  record.put()

# Tweak the user email so it's a valid Kubernetes namespace.  A namespace is a field in an FQDN, so it
# can only contain the characters in a field in an FQDN: 0-9, -, _, +, a-z.  A valid email address contains
# those characters and @ and '.'.  So we replace those characters with a '-', and, since both email and FQDNs 
# are case-insensitive, set everything to lower-case.  So Foo.Bar@waldo.com becomes foo-bar-waldo-com.  To ensure there
# are no conflicts, check to see if there is already such a namespace, and if so add a '0' to the end, then '1', etc..
# return the resulting namespace

def make_new_namespace(email):
  email = email.lower()
  candidate_namespace = email.replace('.','-').replace('@','-')
  count = 0
  namespace = candidate_namespace
  namespaces = User.query(User.namespace == namespace).fetch()
  while len(namespaces) > 0:
    namespace = "%s%d" % (candidate_namespace, count)
    namespaces = User.query(User.namespace == namespace).fetch()
    count = count + 1
  return namespace

# 
# get a user record corresponding to email, returning None if there is no record
# convenience method (we do this a lot)
# Returns a record, not a list.  There should never be more than one.
#
def find_user(user_email):
  user_email = user_email.lower()
  user_record = User.query(User.email == user_email).fetch()
  if len(user_record) < 1:
    return None
  else:
    return user_record[0]

#
# Called after login.  If a user is in the DB, return his record.  Otherwise create a new user record in the 
# db, with the fields:
# email = user_email, namespace = make_new_namespace(email), and agreed_to_AUP, approved, and admininistrator all False
# return the record.  After this routine, the user is in the db.
#


def create_or_find_user(user_email):
  user_email = user_email.lower()
  user_record = find_user(user_email)
  if user_record: return user_record
  agreed_to_AUP =  approved =  administrator = False
  if user_email in admin_users:
    agreed_to_AUP =  approved =  administrator = True
  namespace = make_new_namespace(user_email)
  user_record = User(email=user_email, namespace = namespace, agreed_to_AUP = agreed_to_AUP , approved = approved, administrator = administrator)
  user_record.put()
  return user_record


#
# Hook to the builtin Google App Engine authentication.  Get the user's email and nickname (Google App Engine defined)
# We will use the user's email (all lower case) as our user identifier.  Since this page requires login, get_current_user() is
# always true
#

def get_current_user_nickname_and_email():
  user = users.get_current_user()
  if (user.email()):
    return user.nickname(), user.email()
  return user.nickname(), user.nickname()

   
head_node_url = 'http://head.sundew.ch-geni-net.instageni.washington.edu:8181/' 
kubernetes_head_node = 'https://head.sundewproject.org/'
kubernetes_head_text = 'Sundew Head Node'

#
# A little utility to pull data from an URL.  Robust against some failures, since one of the servers we talk to can be
# slow on the first query...
#

def fetch_from_url(query_url, max_retries = 2):
  retries = 0
  while retries < max_retries:
    try:
      response = urllib.urlopen(query_url).read()
      return response
    except urllib.HTTPException:
      retries = retries + 1
  # didn't get it after max_retries, punting...
  return None

#
# A utility to add the current node list and timestamp to the values structure which 
# gets passed to the templates...
#

def add_node_record_to_values(values):
  current_nodelist = get_current_nodelist_record()
  if current_nodelist:
    values["nodes"] = json.loads(current_nodelist.nodes)
    values["node_fetch_time"] = current_nodelist.time.strftime("%H:%M:%S %A, %B %d, %Y")
  else:
    values["nodes"] = None

# 
# display the next page: this is the main user page, and it is one of three:
# 1. If the user is approved, display his dashboard -- mostly a link to the portal with credential information
# 2. If the user has signed the AUP but has not been approved, show a pending page
# 3. If the user has not signed the AUP, show that.
#

def display_next_page(user_record, request, response):
  logout_url = users.create_logout_url(request.uri)
  values = {"email": user_record.email, "logout": logout_url, "admin": user_record.administrator}
  add_node_record_to_values(values)
  if user_record.approved:

    values["namespace"] = user_record.namespace
    values["kubernetes_head_node"] = kubernetes_head_node
    values["kubernetes_head_text"] = kubernetes_head_text
    if user_record.administrator: 
      # add the admin dashboard code here: fetch all the user records from the database and let the administrator 
      # administer them
      x = 3
    template = JINJA_ENVIRONMENT.get_template('dashboard.html')
    response.write(template.render(values))
  elif user_record.agreed_to_AUP:
    template = JINJA_ENVIRONMENT.get_template('pending.html')
    response.write(template.render(values))
  else:
    template = JINJA_ENVIRONMENT.get_template('aup.html')
    response.write(template.render(email=user_record.email))

# 
# [START main_page]
# Displays the main page and gets information for the link.  Just displays the welcome page and a link to the next page, which will vary depending on the user
#
class MainPage(webapp2.RequestHandler):
  def get(self):
    nickname, email = get_current_user_nickname_and_email()
    values = {"email": email, "logout": users.create_logout_url(self.request.uri), "bucket": bucket_name}
    add_node_record_to_values(values)
    template = JINJA_ENVIRONMENT.get_template('index.html')
    self.response.write(template.render(values))

#
# Handler for next_page.  All this does is get the user_record and then call display_next_page, above, to do the 
# actual next_page rendering
#

class NextPage(webapp2.RequestHandler): 
  def get(self):
    nickname, email = get_current_user_nickname_and_email()
    user_record = create_or_find_user(email)
    display_next_page(user_record, self.request, self.response)

#
# Handles and AUP agreement: catches the AUP agreement, updates the record in the database, then calls display_next_page 
# to do the rendering.
#


class AUP_AGREE(webapp2.RequestHandler):
  def post(self):
    email = self.request.get('email')
    record = create_or_find_user(email)
    record.agreed_to_AUP = True
    record.put()
    display_next_page(record, self.request, self.response)

#
# Download the Kubernetest configuration file.  Catches /download_config requests.  Checks to see if the user can download the config
# file, and if so, calls the head node to get it and then downloads it to the user as a text blob, with headers to tell the user's OS
# to save it as a file.
# In the event that the user hasn't been approved, or hasn't signed the AUP, or there's a problem fetching the config file from the head node,
# an error page is displayed
#

class DownloadConfig(webapp2.RequestHandler):
  def get(self):
    nickname, email = get_current_user_nickname_and_email()
    user_record = create_or_find_user(email)
    values = {"email": email, "logout": users.create_logout_url(self.request.uri), "no_response": False, "approved": user_record.approved, "aup": user_record.agreed_to_AUP}
    template = JINJA_ENVIRONMENT.get_template('config_error.html')
    if user_record.approved:
      query_url = '%s?user=%s' % (head_node_url, user_record.namespace)
      head_node_response = fetch_from_url(query_url, 2)
      if head_node_response:
        self.response.headers['Content-Type'] = 'text/csv'
        self.response.headers['Content-Disposition'] = "attachment; filename=sundew.cfg"
        self.response.out.write(head_node_response)
      else:
        values["no_response"] - True
    else:
      self.response.write(template.render(values))

#
# Update the nodes in the db
#
class UpdateNodes(webapp2.RequestHandler):
  def get(self):
    response = fetch_from_url('http://head.sundewproject.org:8181/nodes')
    if response:
      nodes = response.split('\n')
      if len(nodes) == 0: 
        self.response.out.write('Node list fetch failed.  No nodes in list')
      else:
        if (len(nodes[-1]) == 0):
          nodes = nodes[:-1]
        store_new_nodelist(nodes)
        self.response.out.write('%d nodes found' % len(nodes))
    else:
      self.response.out.write('Node list fetch failed')



    
app = webapp2.WSGIApplication([
  ('/', MainPage),
  ('/next_page/', NextPage),
  ('/next_page/aup_agree', AUP_AGREE),
  ('/download_config', DownloadConfig),
  ('/update_nodes', UpdateNodes)
  ], debug=True)
# [END app]
