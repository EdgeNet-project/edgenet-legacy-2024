<?php

namespace App\Http\Controllers\Edgenet;

use Illuminate\Http\Request;
use App\Http\Controllers\Controller;
use GuzzleHttp\Exception\GuzzleException;
use GuzzleHttp\Client;

//use Illuminate\Pipeline\Pipeline;
//use App\Http\QueryMiddleware\SortBy;
//use App\Http\QueryMiddleware\Filter;

//use App\Nagios\StatusApi;

use App\Http\Resources\NodeCollection;
use App\Http\Resources\NodeResource;
//use App\Http\Resources\NodeStatusResource;
//use App\Http\Resources\NodeLocationResource;
//use App\Http\Resources\SliceCollection;
use App\Facades\CountryCode;
use Maclof\Kubernetes\Client as K8client;

class NodeController extends EdgenetController
{

    public function get($node_hostname = null)
    {

        $response = $this->getRequest('/api/v1/nodes', []);

        return response()->json(json_decode($response->getBody()), $response->getStatusCode());


        $nodes = collect(json_decode($response->getBody()));
        dd($nodes);


//
//        $client = new Client();
//        $response = $client->request('GET', 'https://edgenet.planet-lab.eu:8080/get_status', [
//            'query' => [ 'api_key' => env('EDGENET_API_KEY') ]
//        ]);

//        echo $response->getStatusCode(); # 200
//        echo $response->getHeaderLine('content-type'); # 'application/json; charset=utf8'
//        echo $response->getBody(); # '{"id": 1420053, "name": "guzzle", ...}'

        $nodes = collect(json_decode($response->getBody()))->map(function($node) {
//            dd($node);
            return [
                'id' => $node->node,
                'site_id' => null,
                'site' => null,

                'hostname' => $node->node,
                'hardware' => $node->hardware,
                'interface' => null,

                'slices' => $node->namespaces,

                'ip' => $node->ip,

                'state' => $node->ready == 'True' ? 'boot' : 'fail',
                'status' => $node->ready == 'True' ? 'up' : 'down',
                'monitor' => null,
                'location' => [
                    'alpha2' => $node->{'country-iso'},
                    'alpha3' => CountryCode::alpha2($node->{'country-iso'})['alpha3'],
                    'country' => $node->{'country-iso'},
                    'city' => $node->city,
                    'state' => $node->{'state-iso'},
                    'lat' => $node->lat,
                    'lon' => $node->lon,
                ],

                'created_at' => null,
                'updated_at' => null,

                'testbed' => 'edgenet'
            ];
        });

        $total = $nodes->count();

        return response()->json([
            'data' => $nodes,
            'meta' => [
                'total' => $total,
                'count' => $total
            ]
        ]);
    }
}
