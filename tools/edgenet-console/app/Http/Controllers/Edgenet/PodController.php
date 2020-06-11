<?php

namespace App\Http\Controllers\Edgenet;

use App\Facades\CountryCode;

class PodController extends EdgenetController
{

    public function get($name = null)
    {
        $response = $this->getRequest('/api/v1/pods', [
            'limit' => 10
        ]);

        return response()->json(json_decode($response->getBody()), $response->getStatusCode());
    }
}
