#!/usr/bin/python
#
# A utility called to update the (currently, in-file) database of nodes/ip addresses
# This is a command-line utility over the routines in node_db_lib.py.  
#
# usage: node_db.py add/delete <name> <ip_address>
# all parameters are positional and are required
#
# This is very much an MVP piece of code -- it will need to be replaced with a SQL DB fairly soon.
#

import sys
from node_db_lib import read_db, write_db, delete_entry, add_entry
from functools import reduce

#
# The database file name
#

db_file_name = 'node_db.json'


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
def ok_name(a_name):
  if (len(a_name) > 63 or len(a_name) == 0): return False
  result = reduce(lambda x, y: x and (y.isdigit() or y.isalpha() or (y == '-')), a_name, True)
  return result and a_name[0] != '-' and a_name[-1] != '-'

#
# Print a usage message and exit.  This is called by check_args if there are
# any problems with the command line.
# Side effect: exits with an error
# 
def print_usage_and_exit():
  print 'Usage: node_db.py add/delete host_name ip_address'
  exit(1)

#
# Check the command line arguments for validity -- that there are three arguments,
# 1. The first is add or delete
# 2. The second is a valis host_name (see ok_name)
# 3. The third is a valid ip address (see check_IP)
# Returns: (command, host_name, address) as a triple
# Side Effects: exits with an error message if there is an error
# 

def check_args():
  if (len(sys.argv) != 4): print_usage_and_exit()
  command = sys.argv[1]
  host_name = sys.argv[2]
  address = sys.argv[3]
  if (command != 'add' and command != 'delete'): print_usage_and_exit()
  if (not ok_name(host_name)): print_usage_and_exit()
  if (not check_ip(address)): print_usage_and_exit()
  return (command, host_name, address)

#
# Main routine.
# 1. Get the command, host_name, and address from the command line
# 2. Read the DB
# 3. Execute the add or delete command
# 4. Write the DB
#

if __name__ == '__main__':
  (command, host_name, address) = check_args()
  host_list = read_db()
  if (command == 'add'):
    host_list = add_entry(host_list, host_name, address)
  else:
    host_list = delete_entry(host_list, host_name, address)
  write_db(host_list)


    

