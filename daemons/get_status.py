import subprocess
import json
call = "/usr/bin/kubectl get nodes -o json"
def get_node_record(a_node):
   status = a_node['status']
   host_address = filter(lambda x: x['type'] == 'Hostname', status['addresses'])
   hostname = host_address[0]['address'] if len(host_address) > 0 else None
   ready_list = filter(lambda x: x['type'] == 'Ready', status['conditions'])
   ready = ready_list[0]['status'] if len(ready_list) > 0 else False
   return {"node": hostname, "ready": ready}
nodes = subprocess.check_output(call, shell=True)
node_struct = json.loads(nodes)
result_list = [get_node_record(record) for record in node_struct["items"]]
print json.dumps(result_list)
