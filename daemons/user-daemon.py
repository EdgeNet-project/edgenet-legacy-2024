#!/usr/bin/python
from flask import Flask, request, send_file
import subprocess
from io import StringIO

app = Flask(__name__)
 
@app.route("/")
def hello():
    try:
        user = request.args.get('user')
        cmd = ['../user_files/scripts/make-config.sh','default','-n',user]
	print(cmd)
        print('request for user: ' + user)
        result = subprocess.Popen(cmd,shell=False,stdout=subprocess.PIPE,bufsize=1).communicate()
        f = StringIO()
        f.write((unicode(result[0],"utf-8")))
        f.seek(0)
        return result
    except subprocess.CalledProcessError as e:
        return "Error: %s " % (str(e))

@app.route("/nodes")
def get_nodes():
    try:
      cmd = ['./get_nodes.sh']
      print(cmd)
      result = subprocess.Popen(cmd,shell=False,stdout=subprocess.PIPE,bufsize=1).communicate()
      f = StringIO()
      f.write(unicode(result[0],"utf-8"))
      f.seek(0)
      return result
    except subprocess.CalledProcessError as e:
      return "Error in Get Nodes: %s " % (str(e))    
 
if __name__ == "__main__":
    app.run(host='0.0.0.0',port=8181,debug=True)
