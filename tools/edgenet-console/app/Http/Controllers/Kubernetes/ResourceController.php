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
    protected $api, $token, $client;

    public function __construct(Client $client)
    {
        $this->api = config('kubernetes.api.server')  . preg_replace('#/+#','/', config('kubernetes.api.prefix'));
//        $this->token = config('edgenet.token');
        $this->client = $client;

    }

    public function get($resource)
    {
        try {
            $response = $this->client->request('GET', $this->api . '/' . $resource, [
                'headers' => [
                    'Authorization' => 'Bearer ' . Auth::user()->api_token,
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