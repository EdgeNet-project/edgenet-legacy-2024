<?php

namespace App\Http\Controllers\Edgenet;

use App\Http\Controllers\Controller;
use Illuminate\Http\Request;
use GuzzleHttp\Client;
use GuzzleHttp\Exception;

class EdgenetController extends Controller
{
    protected $api, $token, $client;

    public function __construct(Client $client)
    {
        $this->api = config('edgenet.api');
        $this->token = config('edgenet.token');
        $this->client = $client;

    }

    protected function getRequest($path, $query)
    {

        return $this->client->request('GET', $this->api . $path, [
            'headers' => [
                'Authorization' => 'Bearer ' . $this->token,
                'Accept' => 'application/json',
            ],
            'verify' => false,
            //'debug' => true
            'query' => $query,
            'exceptions' => false
        ]);

//        return response()->json(json_decode($response->getBody()), $response->getStatusCode());

    }

//    protected function getRequest($path, $query) {
//        return $this->sendRequest('GET', $path, $query);
//    }

    protected function postRequest($path, $data) {
        $response = $this->client->request('POST', $this->api . $path, [
            'headers' => [
                'Authorization' => 'Bearer ' . $this->token,
                'Accept' => 'application/json',
            ],
            'verify' => false,
            //'debug' => true
            'json' => $data,
            'exceptions' => false
        ]);

        return $response;
    }

}
