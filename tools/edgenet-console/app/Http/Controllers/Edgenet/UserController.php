<?php

namespace App\Http\Controllers\Edgenet;

use GuzzleHttp\Client;
use Illuminate\Http\Request;
use App\Http\Controllers\Controller;
use Auth;
use Illuminate\Support\Facades\Validator;


class UserController extends EdgenetController
{

    public function get(Request $request, $name = null)
    {
        $api = config('edgenet.api_prefix_crd') . '/users';

        if ($name) {
            $api .= '/' . $name;
        }
        $response = $this->getRequest($api, []);

        return response()->json(json_decode($response->getBody()), $response->getStatusCode());
    }

    public function slices()
    {

        $slices = SliceController::getSlices()->filter(function($slice) {
            return $slice['owner_id'] !== null && $slice['owner_id'] == Auth::user()->id;
        })->values();
        $total = count($slices);

        return response()->json([
            'data' => $slices,
            'meta' => [
                'total' => $total,
                'count' => $total
            ]
        ]);

    }


}
