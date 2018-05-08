#!/usr/bin/python
#
# This implements the callback to the portal to confirm that a namespace has been created, 
# per a portal request, and is now available for use.
# Synopsis: 
# confirm-namespace.py <namespace>
# namespace cannot be omitted.  Note this does no error-checking aside from ensuring that 
# the namespace argument is valid.
#
import requests
import sys
if __name__ == "__main__":
  if len(sys.argv) != 2:
    print >> sys.stderr, "Error: confirm-namespace.py takes exactly one argument, not %d" % len(sys.argv) - 1
    sys.exit(1)
  r = requests.post('http://sundewcluster.appspot.com/confirm_namespace', data = {'namespace':sys.argv[1]})
  if r.status_code == requests.code.ok:
    # do something with the response, which is a JSON dictionary with two fields: outcome (Success/Failure) and reason (a text string)
    pass
  else:
    pass
    # Should log a bad request
