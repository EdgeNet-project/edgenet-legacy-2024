<?php

namespace App\Http\Controllers\Kubernetes;


use App\Http\Controllers\Controller;
use Illuminate\Http\Request;
use Auth;
use GuzzleHttp\Client;

/**
 * Class ApiController
 * @package App\Http\Controllers\Kubernetes
 *
 * We use this proxy controller to communicate with Kubernetes API
 * and avoid CORS related issues.
 *
 */
class ResourceController extends Controller
{
    protected $api, $client, $headers;

    public function __construct(Request $request, Client $client)
    {
        $this->api = config('kubernetes.api.server')  . preg_replace('#/+#','/', config('kubernetes.api.prefix'));
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
            $response = $this->client->request('PATCH', $this->api . '/' . $resource, [
                'headers' => $this->headers,
                'verify' => false,
                //'debug' => true
                'query' => [],
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