<?php

return [
    'api' => [
        'server' => env('KUBERNETES_API_SERVER','https://api.edge-net.org'),
        'prefix' => '/api/v1'
    ],
    'dashboard' => env('KUBERNETES_DASHBOARD_SERVER','https://dashboard.edge-net.org'),

];