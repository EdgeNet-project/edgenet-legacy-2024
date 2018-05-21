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
import httplib
import sys
# import cloudstorage
import datetime
import json

from google.appengine.api import app_identity
from google.appengine.api import users
from google.appengine.ext import ndb
from google.appengine.api import mail


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
# Since we aren't using CloudStorage at present, this is not live code
#
bucket_name = os.environ.get('BUCKET_NAME', app_identity.get_default_gcs_bucket_name())
NULL_NAMESPACE_NAME = "No Namespace"
#
# Status Values for User.namespace_status.  A namespace_status is one of
# NO_NAMESPACE, NAMESPACE_REQUESTED, NAMESPACE_ASSIGNED.  These values
# should NEVER be changed without updating the datastore simultaneously
#
NO_NAMESPACE = 0
NAMESPACE_REQUESTED = 1
NAMESPACE_ASSIGNED = 2


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
  the configuration file, a text blob which the user can use to access his namespace
  """
  email = ndb.StringProperty(indexed = True, required = True)
  namespace = ndb.StringProperty(indexed = True, required = True, default = NULL_NAMESPACE_NAME)
  namespace_status = ndb.IntegerProperty(indexed = True, required = True, default = NO_NAMESPACE)
  agreed_to_AUP = ndb.BooleanProperty('ok', indexed=False, required = True, default = False)
  approved = ndb.BooleanProperty('approved', indexed=False, required = True, default = False)
  administrator = ndb.BooleanProperty(indexed=True, required = True, default = False) 
  has_config = ndb.BooleanProperty(indexed = True, required = True, default = False)
  config = ndb.TextProperty(indexed = False, required = False)


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
# get the current nodelist (most recent snapshot)
# returns: the nodelist record if there is one that has active set to true, None otherwise
#
def get_current_nodelist_record():
  active =  NodeList.query(NodeList.active == True).fetch()
  if len(active) == 0:
    return None
  return active[0]

#
# Store a list of nodes as the current nodelist, with a timestamp to show it's current
# nodelist: the list of nodes as an array of strings
# returns: no return
# side effects: stores the current nodelist
#
def store_new_nodelist(nodelist):
  active = get_current_nodelist_record()
  # Note that the current active record is no longer the active record
  if (active): 
    active.active = False
    active.put()
  record = NodeList(active = True, time = datetime.datetime.now(), nodes = json.dumps(nodelist))
  record.put()

# 
# Tweak the user email so it's a valid Kubernetes namespace.  A namespace is a field in an FQDN, so it
# can only contain the characters in a field in an FQDN: 0-9, -, _, +, a-z.  A valid email address contains
# those characters and @ and '.'.  So we replace those characters with a '-', and, since both email and FQDNs 
# are case-insensitive, set everything to lower-case.  So Foo.Bar@waldo.com becomes foo-bar-waldo-com.  To ensure there
# are no conflicts, check to see if there is already such a namespace, and if so add a '0' to the end, then '1', etc..
# return the resulting namespace
# email: the user's email as a string
# returns: the modified email address
# side effects: None
#

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
# user_email: the user's email as a string
# returns: the user record for this email if one is found, or None
# side effects: None
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
# user_email: the user's email as a string
# returns: the user record corresponding to that email address
# side effects: creates the record if it didn't exist
#


def create_or_find_user(user_email):
  user_email = user_email.lower()
  user_record = find_user(user_email)
  if user_record: return user_record
  # namespace = make_new_namespace(user_email)
  user_record = User(email=user_email,  agreed_to_AUP = False, approved = False, administrator = False)
  user_record.put()
  return user_record


#
# Hook to the builtin Google App Engine authentication.  Get the user's email and nickname (Google App Engine defined)
# We will use the user's email (all lower case) as our user identifier.  If users.get_current_user() returns None (no
# login), this procedure returns None, None
# 
# returns: the user's nickname and email address as a tuple, nickname first
# side effects: none
# note: should it just return None if there is no email address?
#

def get_current_user_nickname_and_email():
  user = users.get_current_user()
  if not user:
    return None, None
  if (user.email()):
    return user.nickname(), user.email()
  return user.nickname(), user.nickname()

   
# head_node_url = 'http://head.sundew.ch-geni-net.instageni.washington.edu:8181/' 
head_node_url = 'https://head.sundewproject.org:8181/'
kubernetes_head_node = 'https://head.sundewproject.org/'
kubernetes_head_text = 'edgeNet Head Node'

#
# A little utility to pull data from an URL.  Robust against some failures, since one of the servers we talk to can be
# slow on the first query...
#
# query_url: the url do do the fetch
# max_retries: the number of times to retry in the event of error
# returns: the response from the query
# side effects: none
#

def fetch_from_url(query_url, max_retries = 2):
  retries = 0
  while retries < max_retries:
    try:
      response = urllib.urlopen(query_url).read()
      return response
    except httplib.HTTPException:
      retries = retries + 1
  # didn't get it after max_retries, punting...
  return None
#
# Fetch the config file for namespace namespace.  This is a thin wrapper over
# fetch_from_url, broken out separately because it is called from a few places
#
# namespace: namespace to fetch the config file for
# returns: the config file or None if there was an error
# 
def fetch_config_file(namespace):
  query_url = '%s?user=%s' % (head_node_url, namespace)
  return fetch_from_url(query_url, 2)

#
# Make a new namespace with namespace.  Return success or failure...
#
# namespace: namespace to create
# returns: None (fail to connect) or structure with success/failure information
# side effect: on success, creates the namespace on the head node
#
def make_namespace_on_head_node(namespace):
  query_url = '%smake-user?user=%s' %(head_node_url, namespace)
  return fetch_from_url(query_url, 2)

#
# Utilities to send mail.  There are two email notifications: an email to administrators
# that accounts are pending approval, and an email to users when their account has been approved
#

#
# Send a notification to administrators to approve records which have agreed to the AUP but not yet been approved
# user_records_for_approval: The list of user records which (a) have an AUP signed; (b) have not been approved;
#      and (c) for whom an approval email has not been sent
# returns: None
# side effect: sends mail to the administrators to get them to approve these records, and marks the 
#      records as having the email sent
#
def send_approval_request(user_record_for_approval):
  admins = User.query(User.administrator == True).fetch()
  url = "https://console.cloud.google.com/datastore/stats?project=sundewcluster"
  admin_emails = [admin.email for admin in admins]
  toLine = ",".join(admin_emails)
  sender = "daemon@sundewcluster.appspotmail.com"
  user_string = "User " + user_record_for_approval.email
  subjectLine = "User Waiting for Approval"
  body = user_string + " is waiting for approval.  Please go to " + url + " to handle this request."
  mail.send_mail(sender = sender, subject = subjectLine, to=toLine, body=body)

#
# Send an email to a user that his account has been approved 
# user_record: The user record that has been approved
# returns: None
# side effect: sends mail to the user that his account has been approved, and marks the records as
#     email_sent
#
def send_approval(user_record):
  toLine = userRecord.email
  sender = "daemon@sundewcluster.appspotmail.com"
  subjectLine = "Edge Net Account for user " + userRecord.email + " approved"
  url = "https://sundewcluster.appspot.com"
  body = "Your edge-net account has been approved with namespace " + user_record.namespace + ".\n"
  body += "Go to " + url + " and log in as " + user_record.email +" and download your configuration file.\n"
  body += "You can use this to manage namespace " + user_record.namespace + " at the head node dashboard or with kubectl."
  mail.send_mail(sender = sender, subject = subjectLine, to=toLine, body=body)
  


#
# A utility to add the current node list and timestamp to the values structure which 
# gets passed to the templates...
# 
# values: a dictionary of values to be used in writing a template
# returns: No return value
# side effects: updates values with the current node list and time
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
# user_record: the record of the user (see class User for fields)
# request: an instance of webapp2.RequestHandler.request
# response: an instance of webapp2.RequestHandler.response
# returns: no return value
# side effects: writes the appropriate next page on response
#

def display_next_page(user_record, request, response):
  logout_url = users.create_logout_url(request.uri)
  values = {"email": user_record.email, "logout": logout_url, "admin": user_record.administrator}
  add_node_record_to_values(values)
  if user_record.approved:

    values["namespace"] = user_record.namespace
    values["kubernetes_head_node"] = kubernetes_head_node
    values["kubernetes_head_text"] = kubernetes_head_text
    values["title"] = "edgeNet Dashboard for " + user_record.email
    if user_record.administrator: 
      # add the admin dashboard code here: fetch all the user records from the database and let the administrator 
      # administer them
      pass
    template = JINJA_ENVIRONMENT.get_template('views/dashboard.html')
    response.write(template.render(values))
  elif user_record.agreed_to_AUP:
    values["title"] = "edgeMet Pending Approval"
    template = JINJA_ENVIRONMENT.get_template('views/pending.html')
    response.write(template.render(values))
  else:
    values["title"] = "edgeNet AUP"
    template = JINJA_ENVIRONMENT.get_template('views/aup.html')
    response.write(template.render(values))

# 
# [START main_page]
# Displays the main page and gets information for the link.  Just displays the welcome page and a link to the next page, which will vary depending on the user
#
class MainPage(webapp2.RequestHandler):
  def get(self):
    nickname, email = get_current_user_nickname_and_email()
    logged_in = True if email else False
    values = {"logged_in": logged_in, "email": email, "logout": users.create_logout_url(self.request.uri), "title": "edgeNet Main Page", "bucket": bucket_name}
    add_node_record_to_values(values)
    template = JINJA_ENVIRONMENT.get_template('views/index.html')
    self.response.write(template.render(values))

#
# Handler for next_page.  All this does is get the user_record and then call display_next_page, above, to do the 
# actual next_page rendering
#

class NextPage(webapp2.RequestHandler): 
  def get(self):
    nickname, email = get_current_user_nickname_and_email()
    # login required for this page, so email is not None
    user_record = create_or_find_user(email)
    display_next_page(user_record, self.request, self.response)

#
# Handles and AUP agreement: catches the AUP agreement, updates the record in the database, then calls display_next_page 
# to do the rendering.
#


class AUP_AGREE(webapp2.RequestHandler):
  def post(self):
    nickname, email = get_current_user_nickname_and_email()
    record = create_or_find_user(email)
    record.agreed_to_AUP = True
    record.put()
    send_approval_request(record)
    display_next_page(record, self.request, self.response)

#
# Write a configuration file to the user. 
# 
# response: an instance of webapp2.RequestHandler.response
# returns: no return value
# side effects: writes the configuration file to the user
# 

def write_config(response, config):
  response.headers['Content-Type'] = 'text/csv'
  response.headers['Content-Disposition'] = "attachment; filename=sundew.cfg"
  response.out.write(config)


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
    values = {"email": email, "namespace": user_record.namespace, "logout": users.create_logout_url(self.request.uri), "no_response": False, "approved": user_record.approved, "aup": user_record.agreed_to_AUP}
    template = JINJA_ENVIRONMENT.get_template('views/config_error.html')
    if user_record.has_config:
      write_config(self.response, user_record.config)
    elif user_record.namespace:
      head_node_response = fetch_config_file(user_record.namespace)
      if head_node_response:
        user_record.config = head_node_response
        user_record.has_config = True
        user_record.put()
        write_config(self.response, user_record.config)
      else:
        values["no_response"] = True
        self.response.write(template.render(values))
    else:
      self.response.write(template.render(values))

#
# Update the nodes in the db
#
class UpdateNodes(webapp2.RequestHandler):
  def get(self):
    response = fetch_from_url(head_node_url + 'nodes')
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

#
# update the configuration files in the db
# This is intended to be run as a cron job, though it does output stuff for testing.
#
class UpdateConfigs(webapp2.RequestHandler):
  # Variables to hold the status and the errors
  def get(self):
    namespaces_created = configs_stored = []
    errors = []
    # first, find the users in the db who have been approved but for whom there is no namespace
    new_users = User.query(User.namespace_status == NO_NAMESPACE).fetch()
    new_users = [user for user in new_users if user.approved]
    # for each such user, tell the head node to create a namespace
    for user in new_users:
      # create the namespace
      namespace = make_new_namespace(user.email)
      result = make_namespace_on_head_node(namespace)
      # result is a json structure with one field: Success or Failure
      # Three outcomes: 
      # (1) No connectivity, in which case result = None
      # (2) Failure in which case status["status"] = "Failure"
      # (3) Success, in which case status["status"] = "Success".  In this case we
      #     record the user's namespace in the DB
      # If a failure, record the error
      if result:
        status = json.loads(result)
        if (status["status"] == "Acknowledged"):
          user.namespace = namespace
          user.namespace_status = NAMESPACE_REQUESTED
          user.put()
          namespaces_created.append(namespace)
          send_approval(user)
        else: 
          errors.append('Namespace creation failed for ' + namespace)
      else:
        errors.append('No response from server on attempt to create namespace ' + namespace)

    #
    # Get and store configuration files for any user with a namespace but no config file
    # Will either succeed or fail to connect.
    #

    noConfigs = User.query(User.namespace_status == NAMESPACE_ASSIGNED and User.has_config == False)
    
    for user_record in noConfigs:
      head_node_response = fetch_config_file(user_record.namespace)
      if head_node_response:
          user_record.config = head_node_response
          user_record.has_config = True
          user_record.put()
          configs_stored.append(user_record.namespace)
      else:
        errors.append('No response from server on attempt to get config for  namespace ' + user_record.namespace)

    # Turn the arrays into strings and write the result for debugging

    create_string = ", ".join(namespaces_created)
    config_string = ", ".join(configs_stored)
    error_string = "\n".join(errors)

    self.response.out.write('Namespaces created: %s\n Configurations stored for: %s\n Errors: %s\n' % (create_string, config_string, error_string))


class ConfirmNamespace(webapp2.RequestHandler):
  def post(self):
    namespace_name = self.request.get('namespace')
    if not namespace_name:
      self.response.out.write('No Namespace sent!')
    else:
      records = User.query(User.namespace == namespace_name).fetch()
      if (len(records) == 0):
        result = {'outcome': 'Failure', 'reason': 'No records for namespace ' + namespace_name + ' found!'}
      elif len(records) == 1:
        records[0].namespace_status = NAMESPACE_ASSIGNED
        result = {'outcome': 'Success', 'reason': 'Namespace ' + namespace_name + ' confirmed!'}
        records[0].put()
      else:
        result = {'outcome': 'Failure', 'reason': 'Multiple records for namespace ' + namespace_name + ' found!'}
      self.response.out.write(json.dumps(result))



    
app = webapp2.WSGIApplication([
  ('/', MainPage),
  ('/next_page/', NextPage),
  ('/next_page/aup_agree', AUP_AGREE),
  ('/download_config', DownloadConfig),
  ('/update_nodes', UpdateNodes),
  ('/update_configs', UpdateConfigs),
  ('/confirm_namespace', ConfirmNamespace)
  ], debug=True)
# [END app]
