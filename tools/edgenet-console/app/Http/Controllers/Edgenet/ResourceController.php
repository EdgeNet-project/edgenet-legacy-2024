<?php

namespace App\Http\Controllers\Edgenet;

use App\Http\Controllers\Controller;
use Illuminate\Http\Request;
use GuzzleHttp\Client;
use Auth;


/*
 * Passthrough class (Proxy)
 */
class ResourceController extends Controller
{

    protected $api, $client, $headers;

    public function __construct(Request $request, Client $client)
    {
        $this->api = config('edgenet.api.server')  . preg_replace('#/+#','/', config('edgenet.api.prefix'));
        $this->client = $client;

        $this->headers = [
            'Content-Type' => $request->headers->get('Content-Type'),
            'Accept' => 'application/json',
        ];

        if (Auth::check()) {
            $this->headers += [
                'Authorization' => 'Bearer ' . Auth::user()->api_token,
            ];
        }
    }

    public function get($resource)
    {
        try {
            $response = $this->client->request('GET', $this->api . '/' . $resource, [
                'headers' => $this->headers,
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

    public function patch(Request $request, $resource)
    {

        try {
            $response = $this->client->request('PATCH', config('edgenet.api.server') . '/' . $request->path(), [
                'headers' => $this->headers,
                'verify' => false,
                //'debug' => true
                'query' => $request->query(),
                'exceptions' => false,
                'json' => $request->all()
            ]);
        } catch (\Exception $e) {
            return response()->json([
                'error' => $e->getMessage()
            ], 500);
        }

        return response()->json(json_decode($response->getBody()), $response->getStatusCode());
    }
}
