#!/usr/bin/env python
#
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
# A node in the system.  Note this obsoletes the previous node-list entry
#
class Node(ndb.Model):
    """A model for a node in the system.  
    We store:
    The node name, a string
    The IPv4 address, a string
    The date when the node was added to the system, a DateTimeProperty
    The location of the node, a lat long
    4. city for the node, a string
    5. region for the node, a string (possibly null)
    6. country for the node, a string
    7. Whether the node is currently ready for use (a Boolean)
    """
    name = ndb.StringProperty(indexed = True, required = True)
    address = ndb.StringProperty(indexed = True, required = True)
    date_added = ndb.DateTimeProperty(required = True)
    location = ndb.GeoPtProperty(required = True)
    city = ndb.StringProperty(required = True)
    region = ndb.StringProperty(required = True)
    country = ndb.StringProperty(required = True)
    ready = ndb.BooleanProperty(required = True, default = False, indexed = True)




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


from config import head_node_url, kubernetes_head_node, kubernetes_head_text     
# head_node_url = 'http://head.sundew.ch-geni-net.instageni.washington.edu:8181/' 
# head_node_url = 'https://head.sundewproject.org:8181/'
# kubernetes_head_node = 'https://head.sundewproject.org/'
# kubernetes_head_text = 'edgeNet Head Node'

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
# Fetch the current secret from the headnode for node_join purposes.  The current
# secret will last for 10 minutes.  This is a thin wrapper over fetch_from_url.  
# It is called from two places, add_node and get_setup
#
# returns: the current secret
# 
def fetch_secret():
    query_url =  head_node_url + 'get_secret' 
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
# tell the head node to add a new worker to the DNS...
# This is a hack since namecheap uses IP addresses for authentication and Google App Engine 
# moves IP addresses.  Will go away when we go to Google-managed DNS
#
# namespace: namespace to create
# returns: None (fail to connect) or structure with success/failure information
# side effect: on success, creates the namespace on the head node
#
def add_node_to_dns(site_name, ip_address):
    query_url = '%sadd_node?sitename=%s&ip_address=%s' % (head_node_url, site_name, ip_address)
    return fetch_from_url(query_url, 2)

#

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
# an add-node exception, thrown if a request to add a node fails for some reason
# Possible reasons are:
# 1. This wasn't a valid node name.  See the comments above check_node_name for what constitutes a valid node name
# 2. The IP address wasn't a valid IPv4 address or wasn't routable
# 3. There is already a node in the database with that name, IP address, or both
# The body of the Exception has an error message with the reason
#
class AddNodeException(Exception):
    pass

#
# Check that an_IP_address is a valid IPv4 address. 
# an_IP_address: a string to be checked for being a valid IPv4 address
# Returns: True if it is, False otherwise
# TODO: check to make sure it's routable.
#

def check_ip(an_IP_address):
    IP_array = an_IP_address.split('.')
    if (len(IP_array) != 4): return False
    for i in range(4):
        try:
            val = int(IP_array[i])
            if val < 0 or val > 255: return False
        except ValueError:
            return False
    return True

#
# Check that a_name is a valid host name for a domain name.  Note that only the host
# name should be given, not the FQDN: 'www' is valid, 'www.yahoo.com' is not. 
# a_name: a string to be checked for being a valid host name
# Returns: True if it is, False otherwise
# Rules: valid characters are alphanumerics, hyphen, the name cannot begin or end with a hyphen,
#        and must be of length <= 63, and can't be empty
#
def check_node_name(a_name):
    if (len(a_name) > 63 or len(a_name) == 0): return False
    result = reduce(lambda x, y: x and (y.isdigit() or y.isalpha() or (y == '-')), a_name, True)
    return result and a_name[0] != '-' and a_name[-1] != '-'

#
# Convert a string with two floats to a GeoPt.  Assumes the caller did the error-checking
# geo_string: the string to be converted
# returns: a GeoPt
#
def convert_string_to_GeoPt(geo_string):
    coords = geo_string.split(',')
    if len(coords) == 1:
        return ndb.GeoPt(float(coords[0]))
    return ndb.GeoPt(float(coords[0]), float(coords[1]))

#
# add a Node to the table.  Throws an add-node exception if the node fails to add
# node_name: the node name
# ip_address: the ip address
# location: the location as a lat-long
# Returns: no return
# side effects: adds Node to the table
# throws: an AddNodeException
#
def add_node(node_name, ip_address, location = None, city = None, region = None, country = None):
    if not check_node_name(node_name):
        raise AddNodeException("%s isn't a valid node name." % node_name)
    if not check_ip(ip_address):
        raise AddNodeException("%s isn't a valid IPv4 address" % ip_address )
    current_nodes = Node.query(Node.name == node_name or Node.address == ip_address).fetch()
    if len(current_nodes) > 0:
        raise AddNodeException('There is already a node with name %s or address %s or both in the node list' % (node_name, ip_address))
    record = Node(name = node_name, address = ip_address, date_added = datetime.datetime.now(), location = convert_string_to_GeoPt(location), city = city, region = region, country = country)
    record.put()

#
# Get out the node records and return them as a record which can be serialized and sent to display on a web page.
#
# no parameters
# returns: an array of node records, each of which is a dictionary with the fields of a Node
# side effects: none
#

def get_node_records():
    nodes = Node.query().fetch()
    return [{
        "name": node.name, "address": node.address, "date": node.date_added.strftime("%H:%M:%S %A, %B %d, %Y"),
        "location": ("{0:.2f}".format(node.location.lat), "{0:.2f}".format(node.location.lon)),
        "city": node.city, "region": node.region, "country": node.country, "ready": node.ready
    } for node in nodes]



#
# A utility to add the current node list and timestamp to the values structure which 
# gets passed to the templates...
# 
# values: a dictionary of values to be used in writing a template
# returns: No return value
# side effects: updates values with the current node list and time
#

def add_node_record_to_values(values):
    values["nodes"] = get_node_records()
    values["num_nodes"] = len(values["nodes"])
    values["active_nodes"] = len([node for node in values["nodes"] if node["ready"]])
    # values["nodes"] = Node.query().fetch()
    # current_nodelist = get_current_nodelist_record()
    # if current_nodelist:
    #     values["nodes"] = json.loads(current_nodelist.nodes)
    #     values["node_fetch_time"] = current_nodelist.time.strftime("%H:%M:%S %A, %B %d, %Y")
    # else:
    #     values["nodes"] = None

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
# URL: /aup_agree
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
# URL: /download_config
# GET request
# No arguments -- email is found from the login
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
# A little utility to trim the '.edge-net.io' off a name returned by the headnode
# node: a node name 
# returns: <foo> if node = <foo>.edge_net.io, or node otherwise
# side effects: none
# HACK and TODO: hardcoded '.edge-net.io' should be replaced with a config!
#

def trim_node_name(node):
    index = node.find('.edge-net.io')
    if (index == -1): return node
    return node[:index]

#
# Update the nodes in the db
# Intended to be run as a cron job
# URL: /update_nodes
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
                # store_new_nodelist(nodes)
                node_names = [trim_node_name(node) for node in nodes]
                stored_nodes = Node.query().fetch()
                for node in stored_nodes:
                    node.ready = node.name in node_names
                    node.put()
                self.response.out.write('%d nodes found' % len(nodes))
        else:
            self.response.out.write('Node list fetch failed')

#
# update the configuration files in the db
# This is intended to be run as a cron job, though it does output stuff for testing.
# URL: /update_configs
# GET request
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


#
# A little utility to get a header from a request, returns None if there is no header for that request
# request: the request to get the header from
# header: the header to get the value for
# returns: the value of the header, None if not present
# 
def get_header(request, header):
    if header in request.headers:
        return request.headers[header]
    else:
        return None

#
# Write the current headnode secret to the client
# 
# response: an instance of webapp2.RequestHandler.response
# secret: the secret to write
# returns: no return value
# side effects: writes the secret
# 

def write_secret(response, secret):
    response.headers['Content-Type'] = 'text/csv'
    response.out.write(secret)
    # response.headers['Content-Disposition'] = "attachment; filename=add_node.sh"
    # template = JINJA_ENVIRONMENT.get_template('views/add_node.sh')
    # response.write(template.render({"secret": secret}))


#
# Add a node to the DB and the cluster.  This is called by the node that wants to be added, with a supplied name.
# tasks:
# 1. Get the current add-node token from the headnode.
# 2. Add the node to the DB -- if this fails, return a 500 and why.
# 3a. If successful in addition, send the setup script to the node
# 3b. If note, send a 500 with the error message
# URL: /add_node?node_name=name
# GET request
#
class AddNode(webapp2.RequestHandler):
    def get(self):
        secret = fetch_secret()
        if not secret:
            self.response.set_status(500)
            self.response.write('Unable to fetch secret from head node, try add_node again')
        else:
            name = self.request.get('node_name')
            address = self.request.remote_addr
            location = get_header(self.request, 'X-Appengine-Citylatlong')
            city = get_header(self.request, 'X-Appengine-City')
            region = get_header(self.request, 'X-Appengine-Region')
            country = get_header(self.request, 'X-Appengine-Country')
            try:
                add_node(name, address, location, city, region, country)
                write_secret(self.response, secret)
                add_node_to_dns(name, address)
            except AddNodeException as add_exception:
                self.response.set_status(500)
                self.response.write('/add_node failure: %s' % str(add_exception))
#
# Deliver the current secret to an existing node.  This is a backup in case we were able to add a node to the DB
# but their subsequent call to add to the head node failed, or they lost contact with the headnode and aren't on the dashboard, etc.
# Tasks:
# 1. Check to see if the node and ip_address is in the DB
# 2. If not, return with failure
# 3. If true, fetch the secret from the headnode and send it to the node
# sends a 500 if the node isn't in the DB or if we failed to fetch a secret
#
# URL: /get_secret?node_name=name
# GET request
# address fetched from header
#
class GetSecret(webapp2.RequestHandler):
    def get(self):
        name = self.request.get('node_name')
        address = self.request.remote_addr
        node_records = Node.query(Node.name == name and Node.address == address).fetch()
        if (len(node_records) == 0):
            self.response.set_status(500)
            self.response.write('There is no node in the database with name %s and ip_address %s' % (node_name, ip_address))
        else:
            secret = fetch_secret()
            if not secret:
                self.response.set_status(500)
                self.response.write('Unable to fetch secret from head node, try get_secret again')
            else:
                write_secret(self.response, secret)

#
# Confirm that namespace namespace has been created.  This should only be called by the head node
# TODO: check the requesting IP!  This is a callback after a call to request the namespace, because
# we didn't want to wait for the headnode to get done.
# URL: /confirm_namespace
# POST request
# Arguments: {namespace: namespace_name}
# Side effect: changes namespace state for the appropriate record to NAMESPACE_ASSIGNED (user sees this as his
#              slice is ready to use)
# TODO: Check the requesting IP to make sure that we are getting called by the headnode.
#


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

#
# show the requesting IP address.
# URL: /show_ip
# GET request
# primarily for debugging/information
#

class ShowIP(webapp2.RequestHandler):
    def get(self):
        self.response.write('Your IP is %s' % self.request.remote_addr)

#
# show the requesting headers.
# URL: /show_headers
# GET request
# primarily for debugging/information -- see what headers and values GAE puts on.
#

class ShowHeaders(webapp2.RequestHandler):
    def get(self):
        result = ["%s: %s"  % (key, self.request.headers[key]) for key in self.request.headers]
        self.response.write("\n".join(result))

#
# Show the configuration parameters.  Primarily for debugging
# URL: /show_config
# GET request
#

class ShowConfig(webapp2.RequestHandler):
    def get(self):
        self.response.write('head_node_url: %s\n, kubernetes_head_node: %s\n' % (head_node_url, kubernetes_head_node))

#
# Show the configuration parameters.  Primarily for debugging
# URL: /show_config
# GET request
#

class ShowConfig(webapp2.RequestHandler):
    def get(self):
        self.response.write('head_node_url: %s\n, kubernetes_head_node: %s\n' % (head_node_url, kubernetes_head_node))


#
# Show the nodes.  This is really a test for the Node entity in the data store, just to see how I can dig them out and how the types
# serialize.  Acts as a test/debug for get_node_records
# URL: /show_nodes
# GET request
#
class ShowNodes(webapp2.RequestHandler):
    def get(self):
        self.response.write(json.dumps(get_node_records()))


#
# Set up the routes.  Note that this must agree with the paths in app.yaml and cron.yaml
#
        
app = webapp2.WSGIApplication([
    ('/', MainPage),
    ('/next_page/', NextPage),
    ('/next_page/aup_agree', AUP_AGREE),
    ('/download_config', DownloadConfig),
    ('/update_nodes', UpdateNodes),
    ('/update_configs', UpdateConfigs),
    ('/add_node', AddNode),
    ('/get_secret', GetSecret),
    ('/confirm_namespace', ConfirmNamespace),
    ('/show_ip', ShowIP),
    ('/show_headers', ShowHeaders),
    ('/show_config', ShowConfig),
    ('/show_nodes', ShowNodes)
    ], debug=True)
# [END app]
