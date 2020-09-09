<?php

namespace App\Http\Controllers\Edgenet;

use App\Http\Controllers\Controller;
use GuzzleHttp\Client;

/*
 * Passthrough class (Proxy)
 */
class ResourceController extends Controller
{

    protected $api, $token, $client;

    public function __construct(Client $client)
    {
        $this->api = preg_replace('#/+#','/',config('edgenet.api.server') . config('edgenet.api.prefix'));
//        $this->token = config('edgenet.token');
        $this->client = $client;

    }

    public function get($resource)
    {
        try {
            $response = $this->client->request('GET', $this->api . '/' . $resource, [
                'headers' => [
//                'Authorization' => 'Bearer ' . $this->token,
                    'Accept' => 'application/json',
                ],
                /*
                 * Verify peer certificate.
                 */
                'verify' => false,
                //'debug' => true
                'query' => [],
                'exceptions' => false
            ]);
        } catch (\Exception $e) {
            return response()->json([
                'error' => $e->getMessage()
            ], 500);
        }

        return response()->json(json_decode($response->getBody()), $response->getStatusCode());
    }
}
