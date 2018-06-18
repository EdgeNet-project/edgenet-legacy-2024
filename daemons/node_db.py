"""
node_db.py

Update the (currently, in-file) database of nodes/IP addresses.

This is a command-line utility over the routines in node_db_lib.py.

Usage:
    node_db.py add/delete <name> <ip_address>

All parameters are positional and are required.

NOTE: This is very much an MVP piece of code -- it will need to be
      replaced with a SQL DB fairly soon.
"""

import node_db_lib
import argparse
import socket


def is_public_IPv4_address(string):
    """
    <Purpose>
      Check if `string` contains a public IPv4 address.

    <Argument>
      string: The string to check

    <Exceptions>, <Side Effects>
      None

    <Returns>
      True if `string` contains a public IPv4 address, False otherwise.
    """
    try:
        # Does string have four octets?
        first_octet, second_octet, _, _ = string.split(".")
        # Are the numbers in the proper range?
        socket.inet_aton(string)
        # Check these private/nonroutable IPv4 address ranges:
        #   RFC1918: 10/8, 172.16/12, 192.168/16,
        #   RFC1122 multicast 224/28
        #   RFC6598 CGNAT: 100.64/10
        if (first_octet == "10" or
                (first_octet == "172" and 16 <= int(second_octet) < 32) or
                (first_octet == "192" and second_octet == "168") or
                (224 <= int(first_octet) < 240) or
                (first_octet == "100" and 64 <= int(second_octet) < 128)):
            return False
        else:
            return True
    except (OSError, # inet_aton couldn't parse it
             AttributeError, # argument had no split method
             ValueError, # wrong number of values to unpack on split
             ):
        return False



def is_RFC1035_label(name):
    """
    <Purpose>
      Check if a name is a well-formed RFC 1035 label.
      Labels are strings, start with a letter, do not end in a hyphen,
      contain only letters, digits, and hyphens otherwise, are at most
      63 characters long and not empty.
      Note that FQDNs are invalid by this definition as they contain dots!

    <Argument>
      name: The name to check

    <Exceptions>, <Side Effects>
      None

    <Returns>
      True if name is a well-formed RFC 1035 label, False otherwise.
    """
    if type(name) is not str:
        return False

    # Check length, first and last characters
    if not (0 < len(name) <= 63):
        return False
    if not name[0].isalpha() or name.endswith("-"):
        return False

    # Check for allowed characters throughout
    for char in name:
        if not (char.isdigit() or char.isalpha() or char == "-"):
            return False
    return True



def handle_command_line_args():
    """
    <Purpose>
      Handle the tool's command line args (via `argparse`).

    <Arguments, Exceptions>
      None

    <Side Effects>
      If the arguments do not conform to the expected format, print
      usage information and exit.

    <Returns>
      The command, host label, and IPv4 address
    """
    parser = argparse.ArgumentParser(description="Update node/IP database")
    parser.add_argument("command", choices=["add", "delete"],
            help="Action to perform on the database entry")
    parser.add_argument("host_label", help="Name of host, excluding its domain suffix")
    parser.add_argument("ipv4", help="Public IPv4 address of host")
    args = parser.parse_args()

    # Check argument contents, print usage on error
    if (not is_public_IPv4_address(args.ipv4) or
            not is_RFC1035_label(args.host_label)):
        parser.parse_args(["-h"])

    return args.command, args.host_label, args.ipv4



def main():
    """
    <Purpose>
      Grab the command line args and execute the add or delete command.
      Afterwards, print the database contents.

    <Arguments>, <Exceptions>
      None

    <Side Effects>
      Update the database according to the command-line args.

    <Returns>
      None
    """
    command, host_label, address = handle_command_line_args()

    db = node_db_lib.Sqllite3DB()
    if command == 'add':
        db.add_entry(host_list, host_name, address)
    elif command == 'delete':
        db.delete_entry(host_list, host_name, address)
    host_list = db.read_db()
    print(host_list)



if __name__ == '__main__':
    main()

