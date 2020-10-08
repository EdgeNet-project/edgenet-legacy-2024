<?php

return [
    'api' => [
        'server' => env('KUBERNETES_API_SERVER','https://api.edge-net.org'),
        /*
         * Certification Authority certificate
         * Specify int he env file the CA certificate from the kubernetes API server:
         * # cat /etc/kubernetes/pki/ca.crt | base64 -w0
         */
        'ca' => env('KUBERNETES_API_CA', null),
        'prefix' => '/api/v1'
    ],
    'dashboard' => [
        'server' => env('KUBERNETES_DASHBOARD','https://dashboard.edge-net.org'),
    ]


];