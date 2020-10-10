<?php

namespace App\Http\Controllers\Edgenet;

use App\Facades\Plc;
use Illuminate\Http\Request;

class AdminController extends EdgenetController
{
    private $plc;

    public function authorities(Request $request, $name = null)
    {

        $api = config('edgenet.api_prefix_crd') . '/authorities';

        if ($name) {
            $api .= '/' . $name;
        }

        $response = $this->getRequest($api, []);

        return response()->json(json_decode($response->getBody()), $response->getStatusCode());

    }

    public function users(Request $request, $name = null)
    {

        $api = config('edgenet.api_prefix_crd') . '/users';

        if ($name) {
            $api .= '/' . $name;
        }

        $response = $this->getRequest($api, []);

        return response()->json(json_decode($response->getBody()), $response->getStatusCode());
    }

}
