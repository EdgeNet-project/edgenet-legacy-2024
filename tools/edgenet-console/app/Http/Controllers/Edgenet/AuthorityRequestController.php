<?php

namespace App\Http\Controllers\Edgenet;

use Illuminate\Http\Request;

class AuthorityRequestController extends EdgenetController
{
    public function get(Request $request, $name = null)
    {
        $api = config('edgenet.api_prefix_crd') . 'siteregistrationrequests';

        $response = $this->getRequest($api, []);

        return response()->json(json_decode($response->getBody()), $response->getStatusCode());
    }

}
