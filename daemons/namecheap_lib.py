
# 
# A set of routines to manipulate the namecheap DNS server, which is where the edge-net domain names are hosted.
# This should be imported into any daemon that wants to interact with namecheap
#
import urllib
import urllib2
import sys
from xml.dom.minidom import parseString
import json
import sys

# 
# namecheap_config hosts configuration information, including Api Keys and user names.
# 

from namecheap_config import *

#
# The domain name we're interested in -- edge-net.io
#
sld = 'edge-net'
tld = 'io'

#
# A little utility to parse records which come back from namecheap.  Keeps only
# those records where the record type is returned
#

def main_records(hostRecords):
    return filter(lambda x: x[2] != 'Type', hostRecords)

#
# Executes a GET request on a URL, and returns the data from the response.
# anURL: the URL for the get request
# fields: the arguments to the get request, as a dictionary of the form {'parameter': value}
# Returns: the response, which is one of xml/json/text/html...
# Side Effect: executes the get request
#
def exec_get(anURL, fields):
    # http = urllib3.PoolManager()
    data = urllib.urlencode(fields)
    # get_url = '%s?%s' % (anURL, build_get_string(fields))
    get_url = '%s?%s' % (anURL, data)
    req = urllib2.Request(get_url)
    httpResponse = urllib2.urlopen(req)
    return  httpResponse.read()

#
# Executes a POST request on a URL, with arguments in the fields onject, and returns the data from the response.
# anURL: the URL for the post request
# fields: the arguments to the post request, as a dictionary of the form {'parameter': value}
# Returns: the response, which is one of xml/json/text/html...
# Side Effect: executes the post request
#

def exec_post(anURL, fields):
    # http = urllib3.PoolManager()
    field_data = urllib.urlencode(fields)
    req = urllib2.Request(anURL, field_data)
    httpResponse = urllib2.urlopen(req)
    return  httpResponse.read()

#
# builds the authentication information for a namecheap API request.  
#
def build_authentication():
    return {'ApiUser': ApiUser, 'UserName': ApiUser, 'ApiKey': authentication, 'clientIP': clientIP}

#
# Turns a record of fields into a string for a get request
#
def build_get_string(fields):
    field_strings = ['%s=%s' % (key, fields[key]) for key in fields]
    return '&'.join(field_strings)

# 
# build a getHosts command for a domain
# domain: a string of the form 'sld.td'
# Returns: a record with the fields for the namecheap API.  This is:
#    tld: Top-level domain as a string
#    sld: Second-level domain as a string
#    Command: the namecheam command, which is: 'namecheap.domains.dns.getHosts'
#
def build_get_hosts(domain):
    parts = domain.split('.')
    result = build_authentication()
    result['tld'] = parts[1]
    result['sld'] = parts[0]
    result['Command'] = 'namecheap.domains.dns.getHosts'
    return result

#
# Build the set hosts command for a domain and a host list.  H
# domain: a string of the form 'sld.td'
# host_list: a list of records of the form {'host': <host name>, 'address': <ipv4 address>}
# Returns: a record with the fields for the namecheap API.  This is:
#    tld: Top-level domain as a string
#    sld: Second-level domain as a string
#    HostName<n>: nth HostName
#    Address<n>: nth Address
#    RecordType<n>: Domain record type for the nth host.  Always 'A' for us
#    Command: the namecheam command, which is: 'namecheap.domains.dns.setHosts'
#
def build_set_hosts(domain, host_list):
    parts = domain.split('.')
    result = build_authentication()
    result['tld'] = parts[1]
    result['sld'] = parts[0]
    for i in range(len(host_list)):
        index = i + 1
        result['HostName%d' % index] = host_list[i]["host"]
        result['Address%d' % index] = host_list[i]["address"]
        result['RecordType%d' % index] = 'A'
    result['Command'] = 'namecheap.domains.dns.setHosts'
    return result

#
# get the hosts for a domain
# domain: a string of the form 'sld.td'
# Returns: a list of hosts, where each host is a tuple (<host name>, <v4 address>, <record type>)
# Side Effects: issues a request to namecheap to get the hosts for the domain
# TODO: check for an error return
#

def get_hosts(domain):
    # xmlResult = parseString(execCommand(getHostsURL()))
    xmlResult = parseString(exec_post(host, build_get_hosts(domain)))
    hostRecords = xmlResult.getElementsByTagName('host')
    return  [(aRecord.getAttribute('Name'), aRecord.getAttribute('Address'), aRecord.getAttribute('Type')) for aRecord in hostRecords]

#
# set the hosts for a domain
# domain: a string of the form 'sld.td'
# host_list: a list of records of the form {'host': <host name>, 'address': <ipv4 address>}
# Returns: the parsed xml from namecheap
# Side effects: executes the setHosts request
# Todo: check for error return
#
def set_hosts(domain, host_list):
    return parseString(exec_post(host, build_set_hosts(domain, host_list)))

