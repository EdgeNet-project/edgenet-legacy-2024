import os
import json
def valid_line(line):
    transfer = line.find('->') >= 0
    address_valid = line.find('*') < 0
    return transfer and address_valid

def get_traceroute(container, address):
    traceroute_result = os.popen("../paris_traceroute.sh %s %s" % (container, address)).read()
    lines = traceroute_result.split('\n')
    valid_lines = [line for line in lines if valid_line(line)]
    bodies = [line.split(':')[1] for line in valid_lines]
    addresses = [body.split('->')[0].strip() for body in bodies]
    return addresses