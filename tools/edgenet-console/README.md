<p align="center"><img src="https://res.cloudinary.com/dtfbvvkyp/image/upload/v1566331377/laravel-logolockup-cmyk-red.svg" width="400"></p>


## Console User Admin



Here is the full example with creating admin user and getting token:

Creating a admin / service account user called k8sadmin

sudo kubectl create serviceaccount console -n kube-system

Give the user admin privileges

sudo kubectl create clusterrolebinding console --clusterrole=cluster-admin --serviceaccount=kube-system:k8sadmin

Get the token

sudo kubectl -n kube-system describe secret $(sudo kubectl -n kube-system get secret | (grep console || echo "$_") | awk '{print $1}') | grep token: | awk '{print $2}'

