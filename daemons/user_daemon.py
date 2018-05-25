#!/usr/bin/python
from flask import Flask, request, send_file, jsonify
import werkzeug # for proper 400 Bad Request handling
import subprocess
from io import StringIO

key = '/etc/letsencrypt/live/head.sundewproject.org/privkey.pem'
cert = '/etc/letsencrypt/live/head.sundewproject.org/cert.pem'

app = Flask(__name__)



class MissingRequestArgError(werkzeug.exceptions.BadRequest):
    """Exception class for missing arguments in requests"""
    pass




def log(logString):
    print('[log]: ' + str(logString))



def noblock_call(cmd, cwd="."):
    try:
      result = subprocess.Popen(cmd, shell=False, stdout=subprocess.PIPE,
              bufsize=1, cwd=cwd)
      return jsonify(status="Acknowledged")
    except subprocess.CalledProcessError as e:
      return "Error: " + repr(e)



def make_call(cmd, cwd='.'):
    try:
        result = subprocess.Popen(cmd, shell=False, stdout=subprocess.PIPE,
                bufsize=1, cwd=cwd).communicate()
        f = StringIO()
        f.write((unicode(result[0],"utf-8")))
        f.seek(0)
        return result
    except subprocess.CalledProcessError as e:
        return "Error: " + repr(e)



@app.route("/")
def hello():
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
    try:
      cmd = ['./get_nodes.sh']
      log(cmd)
      log('node request')
      result = make_call(cmd)
      return result
    except subprocess.CalledProcessError as e:
      return "Error in Get Nodes: " + repr(e)



@app.route("/get_secret")
def get_secret():
    cmd = ['sudo', 'kubeadm', 'token', 'create', '--ttl', '30s']
    log ('token request')
    result = make_call(cmd)
    return result



if __name__ == "__main__":
    context = (cert, key)
    app.run(host='0.0.0.0', port=8181, debug=True, threaded=True,
            ssl_context=context)

