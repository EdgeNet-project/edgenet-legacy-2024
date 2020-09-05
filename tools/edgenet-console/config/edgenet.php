<?php

return [
    'api' => [
        'server' => env('EDGENET_API_SERVER', 'https://api.edge-net.org'),
        'prefix' => env('EDGENET_API_PREFIX', '/apis/apps.edgenet.io/v1alpha')
    ]
];