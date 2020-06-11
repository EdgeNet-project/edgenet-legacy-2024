<?php

namespace App\Http\Controllers\Edgenet;

use Illuminate\Http\Request;
use App\Http\Controllers\Controller;
use GuzzleHttp\Exception\GuzzleException;
use GuzzleHttp\Client;
use Illuminate\Support\Facades\Validator;

use Auth;


class SliceController extends EdgenetController
{

    public function get(Request $request, $name = null)
    {
        $api = config('edgenet.api_prefix_crd') . '/slices';

        $response = $this->getRequest($api, []);

        return response()->json(json_decode($response->getBody()), $response->getStatusCode());
    }

    static function getSlices() {
        $client = new Client();
        $response = $client->request('GET', 'https://edgenet.planet-lab.eu:8080/namespaces', [
            'query' => [ 'api_key' => env('EDGENET_API_KEY') ]
        ]);

        $namespaces = explode(' ', trim($response->getBody(), '[]'));

        $slices = [];
        foreach($namespaces as $namespace) {
            if (preg_match('/^ple-([a-z0-9]+)-([a-z0-9]+)-([a-z0-9]+)/i', $namespace, $matches)) {
                $slices[] = [
                    'id' => $namespace,
                    'name' => $matches[2],
                    'site_id' => $matches[1],
                    'site' => null,
                    'owner_id' => $matches[3],
                    'testbed' => 'edgenet',
                    'description' => null,
                    'url' => null,
                    'nodes' => [],
                    'users' => [],
                    'created_at' => null,
                    'expires_at' => null,
                ];
            } else {
                $slices[] = [
                    'id' => $namespace,
                    'name' => $namespace,
                    'site_id' => null,
                    'site' => null,
                    'owner_id' => null,
                    'testbed' => 'edgenet',
                    'description' => null,
                    'url' => null,
                    'nodes' => [],
                    'users' => [],
                    'created_at' => null,
                    'expires_at' => null,
                ];
            }
        }

        return collect($slices);

    }

    private function createSlice($name) {
        $client = new Client();
        $response = $client->request('GET', 'https://edgenet.planet-lab.eu:8080/make-user', [
            'query' => [
                'api_key' => env('EDGENET_API_KEY'),
                'user' => $name,
                ]
        ]);

        return $response;

    }



    public function getOld(Request $request, $id = null)
    {

//        echo $response->getStatusCode(); # 200
//        echo $response->getHeaderLine('content-type'); # 'application/json; charset=utf8'
//        echo $response->getBody(); # '{"id": 1420053, "name": "guzzle", ...}'

        $slices = $this->getSlices();

        if ($id) {
            $slice = $slices->firstWhere('id', $id);

            if (!$slice) {
                return response()->json(['message' => 'not found'], 404);
            }

            return response()->json($slice);

        }

        $total = count($slices);

        return response()->json([
            'data' => $slices,
            'meta' => [
                'total' => $total,
                'count' => $total
            ]
        ]);
    }

    public function create(Request $request)
    {
        $validator = Validator::make($request->all(), [
            'site' => 'required|regex:/^[a-zA-Z0-9]+$/u|max:50',
            'name' => 'required|regex:/^[a-zA-Z0-9]+$/u|max:50',
            'url' => 'string|max:50',
            'description' => 'string',
        ]);

        if ($validator->fails()) {
            return response()->json([ 'message' => 'request error'], 400);
        }

        $slice_name = 'ple-' . $request->input('site') . '-' . $request->input('name') . '-' . Auth::user()->person_id;

        $slices = $this->getSlices();

        if ($slices->firstWhere('id', $slice_name)) {
            return response()->json(['message' => 'slice already exists'],403);
        }

        if ($slices->firstWhere('owner_id', Auth::user()->person_id)) {
            return response()->json(['message' => 'user already has a slice'],403);
        }

        $id = $this->createSlice(
            $slice_name
        );

        if (!$id) {
            return response()->json([ 'message' => 'error creating new slice'], 500);
        }

        return response()->json([
            'name' => $request->input('name')
        ]);
    }


    public function config($id)
    {

        if (!preg_match('/^ple-([a-z0-9]+)-([a-z0-9]+)-([a-z0-9]+)/i', $id, $matches)) {
            return response()->json([ 'message' => 'invalid namespace'], 500);
        }

        if (Auth::user()->person_id != $matches[3]) {
            return response()->json([ 'message' => 'forbidden'], 403);
        }

        if (!$this->getSlices()->firstWhere('id', $id)) {
            return response()->json(['message' => 'namespace does not exist'],403);
        }


        $client = new Client();
        $response = $client->request('GET', 'https://edgenet.planet-lab.eu:8080', [
            'query' => [
                'api_key' => env('EDGENET_API_KEY'),
                'user' => $id
            ]
        ]);

        if ($response->getStatusCode() !== 200) {
            return response()->json(['message' => 'internal error'],$response->getStatusCode());
        }

        return response()->streamDownload(function() use ($response) { echo $response->getBody(); }, $id . '.yaml',
            ['Content-Type' => 'text/yaml',]
        );
    }
}
