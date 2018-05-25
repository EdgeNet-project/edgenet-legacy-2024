#!/usr/bin/python
# Run continuously, getting the auto-added names from the DB and adding them to the 
import urllib3
import sys
from xml.dom.minidom import parseString
import json
from pymongo import MongoClient, ReturnDocument

host = 'http://api.sandbox.namecheap.com/xml.response'
authentication = ''
domainInfo = ''
clientIP=''
autoDomainName = ''



def mainRecords(hostRecords):
    return filter(lambda x: x[2] != 'Type', hostRecords)



def returnRecord(ip,name):
    record = Record(ip,name)
    return record

def execCommand(anURL):
    http = urllib3.PoolManager()
    httpResponse = http.request('GET', anURL)
    return  httpResponse.data


def getHosts():
    getHostsURL = '%s?Command=namecheap.domains.dns.getHosts&%s&%s&%s' % (host, authentication, domainInfo,clientIP)
    xmlResult = parseString(execCommand(getHostsURL))
    hostRecords = xmlResult.getElementsByTagName('host')
    return  [(aRecord.getAttribute('Name'), aRecord.getAttribute('Address'), aRecord.getAttribute('Type')) for aRecord in hostRecords]

def mainRecords(hostRecords):
    return filter(lambda x: x[2] != 'Type', hostRecords)

def makeSetHostURL(aHostList):
    tuples = [(i + 1, aHostList[i][0], i + 1, aHostList[i][1], i + 1, aHostList[i][2]) for i in range(len(aHostList))]
    hostStrings = ['HostName%d=%s&Address%d=%s&RecordType%d=%s' % aTuple for aTuple in tuples]
    hostString = '&'.join(hostStrings)
    setHostsURL = '%s?Command=namecheap.domains.dns.setHosts&%s&%s&%s&%s' % (host, authentication, domainInfo,clientIP,hostString)
    return setHostsURL

#client = MongoClient('mongodb://mongodb:27017/')
#db = client.gee_master
#nodeCollection = db.nodes

def hostsFromDB():
    return execCommand('https://127.0.0.1:8080/nodes')
    #nodes = nodeCollection.find({})
    #nodes = filter(lambda x:x['dnsName'].endswith(autoDomainName), nodes)
    #suffixLength = -len(autoDomainName)
    #return [(node['dnsName'][:suffixLength], node['ipAddress'], 'A') for node in nodes]

def updateAll():
    keepRecords = mainRecords(getHosts())
    hostRecords = hostsFromDB()
    setURL = makeSetHostURL(keepRecords + hostRecords)
    return execCommand(set)


