# 
# Utilities  called to update the database of nodes and hostnames, in a sqlite3 database
#

# Schema of the database: table nodes, two columns, name and ip
# both varchars

import sqlite3

#
# The database file name.  This should move to a config.py file which is not githubbed
#

db_file_name = '/home/ubuntu/sundew-one/etc/nodesdb.sqlite'

#
# A class for a Sqllite3DB.  Really just here to hold a persistent connection
#

#
# Make a where clause for a db operation, using hostName, address, or both
# name: host name for the clause, default None
# address: ip address for the clause, default None
# Returns:
#     a string with a where clause ("" if neither name nor address specified) for use in a query
#
def make_where(name = None, address = None):
    if name and address:
        return " WHERE name='%s' and ip='%s'" % (name, address)
    if name:
        return  " WHERE name='%s'" % name
    if address:
        return  " WHERE ip='%s'" % address
    return  ""

class Sqllite3DB:
    def __init__(self): 
        self.conn = sqlite3.connect(db_file_name)

    #
    # Get a new connection if we need one.
    # No return
    # Side effect: initializes the connection if we need one
    #
    def make_connection(self):
        if not self.conn:
            self.conn = sqlite3.connect(db_file_name)

    #
    # Close the connection -- no one else can use the database if we have an open one
    # No return
    # Side effect: 
    #    Closes the connection and sets the connection to be None so we don't have a stale connection
    #
    def close_conn(self):
        if self.conn:
            self.conn.close()
            self.conn = None

    #
    # Execute a query on the database.  This is a utility used by the convenience routines here,
    # or it can be used directly.  Returns the cursor as a result.  NOTE: THIS CODE DOES NOT SANITIZE
    # ITS INPUT!  It assumes that it is being called from TRUSTED code.  DO NOT EXPOSE THIS METHOD
    # through, say, a web server field
    # query: the query to execute, as a valid SQL string
    # Returns:
    #     a cursor after the query has been executed, which holds the results
    #
    def execute_query(self, query):
        self.make_connection()
        c = self.conn.cursor()
        c.execute(query)
        return c
    #
    # Fetch nodes from the database, by host name, ip address, or both.
    # if name == None, search on ip address only; if ip address == None, 
    # search on name only.  If both == None (default) return all hosts
    # name: name to search on, default None
    # address: address to search on, default None
    # Returns:
    #     a list of the form [{'name': name, 'address': address}]
    #
    def find_hosts(self, name = None, address = None):
        query = 'SELECT * from nodes %s;' % make_where(name, address)
        cursor = self.execute_query(query)
        rows = cursor.fetchall()
        return [{'name': record[0], 'address': record[1]} for record in rows]

    #
    # Read full host_list from the database
    # Returns: the host list in the form [{"name": name, "address":ip_address}...]
    #

    def read_db(self):
        return self.find_hosts()


    #
    # Delete the entry {name: host_name, address: ip_address} from the database
    # host_name: name to be deleted -- not used if None
    # ip_address: address to be deleted -- not used if None
    # Returns: None
    #

    def delete_entry(self, host_name = None, ip_address = None):
        if ((not host_name) and (not ip_address)): return
        query = 'DELETE from nodes %s;' % make_where(host_name, ip_address)
        self.execute_query(query)
        self.conn.commit()

    #
    # Add  the entry {name: host_name, address: ip_address} to the database
    # host_name: name to be added
    # ip_address: address to be added
    # Returns: None 
    # No effect if there is an entry with name host_name or address ip_address already in the
    # database
    #
    def add_entry(self, host_name, ip_address):
        self.make_connection()
        if host_name == None or ip_address == None: return
        rows = self.find_hosts(name = host_name)
        if (len(rows) > 0): return
        rows = self.find_hosts(address = ip_address)
        if (len(rows) > 0): return
        query = "INSERT INTO nodes (name, ip) values(?, ?);"
        c = self.conn.cursor()
        c.execute(query, (host_name, ip_address))
        self.conn.commit()



        

