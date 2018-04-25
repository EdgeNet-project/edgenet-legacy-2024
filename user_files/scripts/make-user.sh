CA_LOCATION="/etc/kubernetes/pki"

if [ -z "$1" ]
  then
    echo "No argument supplied"
    exit 1
fi

USER=$1

kubectl create namespace $USER
openssl genrsa -out ../keys/$USER.key 2048
openssl req -new -key ../keys/$USER.key -out ../keys/$USER.csr -subj "/CN=$USER/O=sundew"
openssl x509 -req -in ../keys/$USER.csr -CA $CA_LOCATION/ca.crt -CAkey $CA_LOCATION/ca.key -CAcreateserial -out ../keys/$USER.crt -days 500
kubectl config set-credentials $USER --client-certificate=../keys/$USER.crt  --client-key=../keys/$USER.key
sed "s/template/$USER/g" ../yml/role-template.yml > ../yml/role-$USER.yml
sed "s/template/$USER/g" ../yml/rb-template.yml > ../yml/rb-$USER.yml
#kubectl create -f ../yml/role-$USER.yml
#kubectl create -f ../yml/rb-$USER.yml
#kubectl create rolebinding $USER-rolebind --clusterrole=admin --serviceaccount=$USER:default --namespace=$USER
#kubectl -n $USER describe secret $(kubectl -n kube-system get secret | grep $USER | awk '{print $1}')
