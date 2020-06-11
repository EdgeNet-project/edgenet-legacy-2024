<?php

namespace App\Http\Controllers\Edgenet;

use Illuminate\Http\Request;

class UserRequestController extends EdgenetController
{
    public function get(Request $request, $name = null)
    {

        $api = config('edgenet.api_prefix_crd') . 'userregistrationrequests';

        $response = $this->getRequest($api, []);

        return response()->json(json_decode($response->getBody()), $response->getStatusCode());
    }

}
