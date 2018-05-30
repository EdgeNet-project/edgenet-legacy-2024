# 
# Utilities  called to update the (currently, in-file) database of nodes/ip addresses
# This is kept in a JSON structure on the disk, which is just a list of records of the 
# form {name: <name>, address: <routable IPv4 address>}
#
# This is very much an MVP piece of code -- it will need to be replaced with a SQL DB fairly soon.
#

import json

#
# The database file name
#

db_file_name = 'node_db.json'


#
# Read the database from the db_file_name and parse it as JSON
# Returns: the host list 
# TODO: lots of error-checking!  Check for a bad parse, check for a bad open...
#

def read_db():
    f = open(db_file_name, 'r')
    json_string = f.read()
    f.close()
    return json.loads(json_string)
#
# Write  the database to the db_file_name and JSON
# a_host_list: the host list to be dumped 
# TODO: lots of error-checking!  Check for a null list, bad open...
#

def write_db(a_host_list):
    f = open('node_db.json', 'w')
    output = json.dumps(a_host_list)
    f.write(output)
    f.write('\n')
    f.close() 

#
# Delete the entry {name: host_name, address: ip_address} from the host_list
# a_host_list: host_list to do the deletion from
# host_name: name to be deleted
# ip_address: address to be deleted
# Returns: the list with entries deleted.  
# Note this returns a new copy of the list; the old list is unchanged
#

def delete_entry(a_host_list, host_name, ip_address):
    new_list = [entry for entry in a_host_list if entry['name'] != host_name or entry['address'] != ip_address]
    return new_list

#
# Add  the entry {name: host_name, address: ip_address} to the host_list
# a_host_list: host_list to add the new items to
# host_name: name to be added
# ip_address: address to be added
# Returns: the list with entries added.  
# Note this returns a new copy of the list; the old list is unchanged
# No effect if there is an entry with name host_name already in the list
#
def add_entry(a_host_list, host_name, ip_address):
    new_list = a_host_list[:]
    matches = [entry for entry in new_list if entry['name'] == host_name]
    if len(matches) == 0:
        new_list.append({'name': host_name, 'address': ip_address})



        

