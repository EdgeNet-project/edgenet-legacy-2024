#!/bin/bash
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root" 
   exit 1
fi

apt-get update && apt-get install -y apt-transport-https -y
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
cat << EOF >/etc/apt/sources.list.d/kubernetes.list
deb http://apt.kubernetes.io/ kubernetes-xenial main
EOF
apt-get update 
apt-get install docker.io kubelet kubeadm kubectl kubernetes-cni -y

##Right now, this is where the node is getting the secret straight from
##The head node.  THis will not work once we move to a whitelist access.
##It will need to move to the portal.

swapoff -a
read -p 'Enter site name (site.edge-net.io): ' sitename

SECRET=$(curl https://sundewcluster.appspot.com/add_node?node_name=$sitename)

##This curl is where it was asking to join the cluster and passing its site name
##please change it to whatever you need for your new API
##curl https://headnode.edge-net.org:8080/add_node?sitename=$sitename
##
OLDHOST=hostname
hostname $sitename.edge-net.io
$SECRET
hostname $OLDHOST
