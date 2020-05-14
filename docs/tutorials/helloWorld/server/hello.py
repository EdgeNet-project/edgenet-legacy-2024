from flask import Flask
import json
import os
import datetime
import sys
from flask_wtf import FlaskForm
from wtforms import StringField, SubmitField
from wtforms.validators import DataRequired
from flask import render_template
from flask import request
sys.path.append('.')


from config import Config
app = Flask(__name__)
app.config.from_object(Config)
hosts =  {"bbn.edge-net.io":{"lon":"-97.822000", "lat":"37.751000"},
"berkeley-gdp.edge-net.io":{"lon":"-97.822000", "lat":"37.751000"},
"case.edge-net.io":{"lon":"-81.525800", "lat":"41.602500"},
"cenic-2.edge-net.io":{"lon":"-122.636000", "lat":"38.957600"},
"cenic.edge-net.io":{"lon":"-122.636000", "lat":"38.957600"},
"clemson.edge-net.io":{"lon":"-82.837400", "lat":"34.683400"},
"cornell-2.edge-net.io":{"lon":"-97.822000", "lat":"37.751000"},
"cornell.edge-net.io":{"lon":"-97.822000", "lat":"37.751000"},
"edgenet.planet-lab.eu":{"lon":"2.338700", "lat":"48.858200"},
"edgenet1.planet-lab.eu":{"lon":"2.328100", "lat":"48.860700"},
"edgenet2.planet-lab.eu":{"lon":"2.328100", "lat":"48.860700"},
"gatech-2.edge-net.io":{"lon":"-84.397300", "lat":"33.774600"},
"gatech.edge-net.io":{"lon":"-84.397300", "lat":"33.774600"},
"gpeni-2.edge-net.io":{"lon":"-97.822000", "lat":"37.751000"},
"gpeni.edge-net.io":{"lon":"-97.822000", "lat":"37.751000"},
"hawaii-2.edge-net.io":{"lon":"-97.822000", "lat":"37.751000"},
"hawaii.edge-net.io":{"lon":"-97.822000", "lat":"37.751000"},
"ilabt.edge-net.io":{"lon":"4.000000", "lat":"50.833300"},
"illinois-2.edge-net.io":{"lon":"-88.206200", "lat":"40.104700"},
"illinois.edge-net.io":{"lon":"-88.206200", "lat":"40.104700"},
"iu-2.edge-net.io":{"lon":"-86.469200", "lat":"39.230300"},
"iu.edge-net.io":{"lon":"-86.469200", "lat":"39.230300"},
"kettering.edge-net.io":{"lon":"-83.749800", "lat":"43.057300"},
"louisiana.edge-net.io":{"lon":"-91.188600", "lat":"30.403000"},
"metrodatacenter-2.edge-net.io":{"lon":"-83.113100", "lat":"40.110400"},
"metrodatacenter.edge-net.io":{"lon":"-83.113100", "lat":"40.110400"},
"missouri-2.edge-net.io":{"lon":"-97.822000", "lat":"37.751000"},
"missouri.edge-net.io":{"lon":"-97.822000", "lat":"37.751000"},
"naist.edge-net.io":{"lon":"135.833300", "lat":"34.683300"},
"northwestern.edge-net.io":{"lon":"-87.684200", "lat":"42.059800"},
"nps.edge-net.io":{"lon":"-121.793500", "lat":"36.621700"},
"nysernet-2.edge-net.io":{"lon":"-97.822000", "lat":"37.751000"},
"nysernet.edge-net.io":{"lon":"-97.822000", "lat":"37.751000"},
"nyu.edge-net.io":{"lon":"-97.822000", "lat":"37.751000"},
"osu.edge-net.io":{"lon":"-82.755300", "lat":"39.907200"},
"princeton-2.edge-net.io":{"lon":"-97.822000", "lat":"37.751000"},
"princeton.edge-net.io":{"lon":"-97.822000", "lat":"37.751000"},
"sox-2.edge-net.io":{"lon":"-84.397300", "lat":"33.774600"},
"sox.edge-net.io":{"lon":"-84.397300", "lat":"33.774600"},
"stanford-2.edge-net.io":{"lon":"-122.163900", "lat":"37.423000"},
"stanford.edge-net.io":{"lon":"-122.163900", "lat":"37.423000"},
"uchicago-2.edge-net.io":{"lon":"-87.604600", "lat":"41.782100"},
"uchicago.edge-net.io":{"lon":"-87.604600", "lat":"41.782100"},
"ucla-2.edge-net.io":{"lon":"-118.441400", "lat":"34.064800"},
"ucla.edge-net.io":{"lon":"-118.441400", "lat":"34.064800"},
"ucsd-2.edge-net.io":{"lon":"-117.276700", "lat":"32.848700"},
"ucsd.edge-net.io":{"lon":"-117.276700", "lat":"32.848700"},
"uky-2.edge-net.io":{"lon":"-97.822000", "lat":"37.751000"},
"uky-3.edge-net.io":{"lon":"-97.822000", "lat":"37.751000"},
"uky.edge-net.io":{"lon":"-97.822000", "lat":"37.751000"},
"umich.edge-net.io":{"lon":"-97.822000", "lat":"37.751000"},
"umkc-2.edge-net.io":{"lon":"-94.573700", "lat":"39.038300"},
"umkc.edge-net.io":{"lon":"-94.573700", "lat":"39.038300"},
"utdallas.edge-net.io":{"lon":"-96.777600", "lat":"32.767300"},
"uvm.edge-net.io":{"lon":"-73.082500", "lat":"44.442100"},  
"vcu.edge-net.io":{"lon":"-97.822000", "lat":"37.751000"},
} 
class DownloadForm(FlaskForm):
    nickname = StringField('Nickname', validators=[DataRequired()])
    submit = SubmitField('Get Yaml!')

hellos = []

def make_output_strings(hello_array):
    hello_strings = ['from host %s, user %s at %s' % (hello["hostname"], hello["username"], hello["timestamp"]) for hello in hello_array]
    return '\n'.join(hello_strings)

@app.route('/get_yaml')
def get_yaml():
    # form = DownloadForm()
    return render_template('download.html', title='Get YAML!')

anonymous_user = 0

@app.route('/download')
def download():
    form = DownloadForm()
    nickname = request.args['nickname']
    if (nickname == None):
        nickname = 'anonymous%d' % anonymous_user
        anonymous_user = anonymous_user + 1
    return render_template('deploy.yaml', nickname=nickname)
    
    

@app.route('/')
def hello_word():
    return 'Hello, World'

@app.route('/hello/<hostname>/<username>')
def hello_hostname(hostname, username):
    hellos.append({"hostname": hostname, "username": username, "timestamp": datetime.datetime.now().isoformat()})
    return 'hello %s at %s' % (username, hostname)

@app.route('/clear')
def clear():
    hellos = []
    return 'Hellos cleared'

@app.route('/show_hellos')
def show_hellos():
    return make_output_strings(hellos)

@app.route('/user_hellos/<username>')
def user_hellos(username):
    return make_output_strings([hello for hello in hellos if hello["username"] == username])

@app.route('/site_hellos/<sitename>')
def site_hellos(sitename):
    return make_output_strings([hello for hello in hellos if hello["hostname"] == sitename])
   

@app.route('/get_hellos')
def get_hellos():
    result = []
    for hello in hellos:
        host = hello['hostname']
        if host in hosts: 
            record = hosts[host]
            result.append({'hostname': host, 'username': hello['username'], 'lat': record['lat'], 'lng': record['lon'], 'timestamp': hello['timestamp']})
    return json.dumps(result)

def valid_line(line):
    transfer = line.find('->') >= 0
    address_valid = line.find('*') < 0
    return transfer and address_valid



@app.route('/get_traceroute/<container>/<address>')
def get_traceroute(container, address):
    traceroute_result = os.popen("./paris_traceroute.sh %s %s" % (container, address)).read()
    lines = traceroute_result.split('\n')
    valid_lines = [line for line in lines if valid_line(line)]
    bodies = [line.split(':')[1] for line in valid_lines]
    addresses = [body.split('->')[0].strip() for body in bodies]
    return json.dumps(addresses) 