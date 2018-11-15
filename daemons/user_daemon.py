#!/usr/bin/python
"""
user_daemon.py -- Interface between the EdgeNet portal and the headnode.  
This implements a simple Flask server which handles requests from the EdgeNet 
portal and issues commands to the headnode to implement them.  All authoritative state
for the EdgeNet system is kept by the portal, which issues http requests through this server
to ensure that the headnode is maintaining this state.

This server also acts as an intermediary for DNS entries, currently kept by namecheap.com.  Namecheap
uses IP whitelisting as its major security tool, and since the portal currently runs on the Google App
Engine and this doesn't offer fixed IP addresses, we use this as the DNS intermediary.  Once we 
either switch away from namecheap and/or switch to the Google App Engine Flex, we'll move that 
to the portal.

Copyright 2018 US Ignite

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

This code uses the NYU style guidelines at:
https://github.com/secure-systems-lab/code-style-guidelines 
"""
from flask import Flask, request, send_file, jsonify, Response
import werkzeug # for proper 400 Bad Request handling
import subprocess # most portal requests turn into command executions of shell scripts
from io import StringIO
import namecheap_lib # for DNS
from get_status import node_status

#
# What are these certs used for?
#

key = '/etc/letsencrypt/live/head.sundewproject.org/privkey.pem'
cert = '/etc/letsencrypt/live/head.sundewproject.org/cert.pem'

app = Flask(__name__,static_url_path='/home/ubuntu/sundew-one/daemons/')

class MissingRequestArgError(werkzeug.exceptions.BadRequest):
    """Exception class for missing arguments in requests"""
    pass

def log(logString):
    print('[log]: ' + str(logString))


def noblock_call(cmd, cwd="."):
    """
    <Purpose>
      Issue cmd as a non-blocking shell command (in other words, issue the command and return immediately,
      without waiting for a Returns)

    <Arguments>
      cmd: the command to issue
      cwd: the directory to execute the command in.  Defaults to '.'

    <Side Effects>
      Executes the command

    <Exceptions, Returns>
      No Exceptions.  Returns an "Acknowledged" String if the process was forked successfully, and an error otherwise
    """
    try:
      result = subprocess.Popen(cmd, shell=False, stdout=subprocess.PIPE,
              bufsize=1, cwd=cwd)
      return jsonify(status="Acknowledged")
    except subprocess.CalledProcessError as e:
      return "Error: " + repr(e)

def make_call(cmd, cwd='.'):
    """
    <Purpose>
      Issue cmd as a nblocking shell command (waits for the call to return, then returns the result)

    <Arguments>
      cmd: the command to issue
      cwd: the directory to execute the command in.  Defaults to '.'

    <Side Effects>
      Executes the command

    <Exceptions, Returns>
      No Exceptions.  Returns the output if the command completed successfully, and an error otherwise
    """
    try:
        result = subprocess.Popen(cmd, shell=False, stdout=subprocess.PIPE,
                bufsize=1, cwd=cwd).communicate()
        f = StringIO()
        f.write((unicode(result[0], "utf-8")))
        f.seek(0)
        return result
    except subprocess.CalledProcessError as e:
        return "Error: " + repr(e)

#
# A reliable method to get the remote ip of a client, if behind a reverse proxy
# See: https://stackoverflow.com/questions/3759981/get-ip-address-of-visitors
#

def remote_ip(request):
    """
    <Purpose>
      Get the remote IP of a client that issued an http request

    <Arguments>
      request: the request, which should be an instance of flask.request

    <Side Effects>
      None

    <Exceptions, Returns>
      No Exceptions.  NOTE: should check if request is None.  returns the IP of the requestor.
    """
    if 'X-Forwarded-For' in request.headers:
        request.headers.getlist("X-Forwarded-For")[0].rpartition(' ')[-1]
    else:
        return request.remote_addr
    if request.environ.get('HTTP_X_FORWARDED_FOR') is None:
        return request.environ['REMOTE_ADDR']
    else:
        return request.environ['HTTP_X_FORWARDED_FOR'] # if behind a proxy


#
# The execution engines for the various requests.
#

@app.route("/")
def hello():
    """
    <Purpose>
      Add a user to the headnode (actually, a namespace, which generates a configuration file)
      Call is /?user=<username>
      Question: I think this is now unused by the portal.  If it is it should be deleted.

    <Arguments>
      None

    <Side Effects>
      Creates the user and config file

    <Exceptions, Returns>
      raises a MissingRequestArgError if no parameter user in request.  Returns the result if successful; error otherwise
    """
    if not "user" in request.args:
        raise MissingRequestArgError(description="No 'user' arg in request.")

    user = request.args.get('user')
    try:
        cmd = ['../user_files/scripts/make-config.sh', 'default', '-n', user]
        log(cmd)
        log('request for user: ' + user)
        result = make_call(cmd)
        return result
    except subprocess.CalledProcessError as e:
        return "Error: " + repr(e)



@app.route('/make-user')
def make_user():
    """
    <Purpose>
      Add a user to the headnode (actually, a namespace, which generates a configuration file)
      Call is /make-user?user=<username>

    <Arguments>
      None

    <Side Effects>
      Creates the user and config file

    <Exceptions, Returns>
      raises a MissingRequestArgError if no parameter user in request.  Returns a positive result from a successful non-blocking all if 
      the process was invoked without error; error otherwise
    """
    if not "user" in request.args:
        raise MissingRequestArgError(description="No 'user' arg in request.")

    user = request.args.get('user')
    try:
        cmd = ['sudo', './make-user.sh', user]
        log('user creation request for: ' + user)
        result = noblock_call(cmd, r'../user_files/scripts')
        return result
    except subprocess.CalledProcessError as e:
        return jsonify(status= "Fail")



@app.route("/nodes")
def get_nodes():
    """
    <Purpose>
      Get the list of nodes currently known by the head node
      Call is /nodes

    <Arguments>
      None

    <Side Effects>
      None

    <Exceptions, Returns>
      returns the list of nodes as a JSON file (see the get_nodes shell script), or error if there was an error invoking
      the get_nodes script
    """
    try:
      cmd = ['./get_nodes.sh']
      log(cmd)
      log('node request')
      result = make_call(cmd)
      return result
    except subprocess.CalledProcessError as e:
      return "Error in Get Nodes: " + repr(e)

@app.route("/get_status")
def get_status():
    """
    <Purpose>
      Get the status of nodes currently known by the head node
      Call is /get_status

    <Arguments>
      None

    <Side Effects>
      None

    <Exceptions, Returns>
      See Exceptions, Returns from get_status.node_status
    """
    return node_status()

@app.route("/get_secret")
def get_secret():
    """
    <Purpose>
      Get a shared secret that the add_node script can run on the client and pass to this node to ensure that
      the add_node is legitimate
      Call is /get_secret

    <Arguments>
      None

    <Side Effects>
      None

    <Exceptions, Returns>
      Returns the result of the command $ sudo kubeadm token create --print-join-command --ttl 30s
    """
    cmd = ['sudo', 'kubeadm', 'token', 'create', '--print-join-command','--ttl', '30s']
    log ('token request')
    result = make_call(cmd)[0]
    log(result)
    return result

@app.route("/show_ip")
def show_ip():
    """
    <Purpose>
      Test the remote_ip() routine: return the IP of the requestor
      Call: /show_ip

    <Arguments>
      None

    <Side Effects>
      None

    <Exceptions, Returns>
      Returns the IP of the requesting host
    """
   return 'Your request is from %s' % remote_ip(request)
#
# /add_node?sitename=<node_name>&ip_address=address
# called from portal, no error-checking
#
@app.route("/add_node")
def add_node():
    """
    <Purpose>
      Add a node to the DNS records for edge-net.io
      Call: /add_node?ip_address=<ip_address>&sitename=<sitename>&record_type=<A or AAA>

    <Arguments>
      None

    <Side Effects>
      Adds the record (sitename.edge-net.io, ip_address, record_type) to the DB on namecheap

    <Exceptions, Returns>
      Either a successful we-added-it or an error return if the sitename or ip address already exists
    """
    ip = str(request.args.get('ip_address'))
    site = str(request.args.get('sitename'))
    record_type = str(request.args.get('record_type'))
    hosts = namecheap_lib.get_hosts('edge-net.io')
    found = [host for host in hosts if host[0] == site or host[1] == ip]
    if (len(found) > 0):
        return Response("Error: Site name %s or address %s already exists" % (site, ip), status=500, mimetype='application/json')
    else:
        hosts.append((site, ip, record_type))
        namecheap_lib.set_hosts('edge-net.io', hosts)
	return Response("Site %s.edge-net.io added at ip %s" % (site, ip))

@app.route("/get_setup")
def get_setup():
    """
    <Purpose>
      Download the setup_node.sh script
      Call: /get_setup

    <Arguments>
      None

    <Side Effects>
      None

    <Exceptions, Returns>
      Sends the setup_node.sh file
    """
    return send_file('setup_node.sh')

@app.route("/show_headers")
def get_headers():
    """
    <Purpose>
      Show all the headers, just for testing purposes
      Call: /show_headers

    <Arguments>
      None

    <Side Effects>
      None

    <Exceptions, Returns>
      Returns the headers of the request
    """
    return jsonify(request.headers)

@app.route("/namespaces")
def get_namespaces():
     """
    <Purpose>
      Get the set of current namespaces
      Call: /get_namespaces

    <Arguments>
      None

    <Side Effects>
      None

    <Exceptions, Returns>
      Returns the current namespaces
    """
  cmd = ['./get-namespaces.sh']
  return make_call(cmd)

if __name__ == "__main__":
    #key = '/etc/letsencrypt/live/headnode.edge-net.org/privkey.pem'
    #cert = '/etc/letsencrypt/live/headnode.edge-net.org/cert.pem'
    #context = (cert, key)
    app.run(host='0.0.0.0', port=8181, debug=True, threaded=True)
            #ssl_context=context)

