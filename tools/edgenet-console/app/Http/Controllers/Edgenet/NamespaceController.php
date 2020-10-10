<?php

namespace App\Http\Controllers\Edgenet;

class NamespaceController extends EdgenetController
{

    public function get($name = null)
    {
        $response = $this->getRequest('/api/v1/namespaces', []);

        return response()->json(json_decode($response->getBody()), $response->getStatusCode());
    }
}
