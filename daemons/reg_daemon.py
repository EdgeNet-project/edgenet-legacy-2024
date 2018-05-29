#!/usr/bin/python
#
# Run continuously, getting the ground-truth names of the hosts from the on-node database and setting the DNS server to have them.
#

import namecheap_lib import set_hosts
from node_db_lib import read_db

if __name__ == '__main__':
  host_list = read_db()
  set_hosts('edge-net.io', host_list)
