The following is the flask server (run as system service)
user_daemon.py

The following files are called by the flask server (not called directly)
get-namespaces.sh	- returns active namespaces
get_ips.sh	- Go template based file (not used) to return IPs
get_nodes.sh- Returns hostnames of Nodes
get_status.py	- python file that checks ready status of nodes
namecheap_lib.py	= support library for namecheap calls

The following files are likely unused:
node_db_lib.py
reg_daemon.py	
setup_node.sh	(was used to construct setup file now living in portal repo)
