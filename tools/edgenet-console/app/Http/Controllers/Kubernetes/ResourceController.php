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
    protected $client;

    public function __construct(Client $client)
    {
        $this->client = $client;
    }

    private function headers() {
        $headers = [
            'Content-Type' => request()->headers->get('Content-Type','application/json'),
            'Accept' => 'application/json',
        ];
        if (Auth::user()) {
            $headers += [
                'Authorization' => 'Bearer ' . Auth::user()->api_token,
            ];
        }

        return $headers;
    }

    public function get(Request $request)
    {
        try {
            $response = $this->client->request('GET', config('kubernetes.api.server') . '/' . $request->path(), [
                'headers' => $this->headers(),
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

    public function patch(Request $request)
    {
        try {
            $response = $this->client->request('PATCH', config('kubernetes.api.server') . '/' . $request->path(), [
                'headers' => $this->headers(),
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