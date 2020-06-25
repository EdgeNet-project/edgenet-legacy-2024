<?php

namespace App\Http\Controllers\K8s;

use App\Http\Controllers\Controller;
use App\Providers\RouteServiceProvider;
use Illuminate\Auth\Events\Registered;
use Illuminate\Http\Request;
use GuzzleHttp\Client;
use GuzzleHttp\Exception\RequestException;

use Illuminate\Foundation\Auth\AuthenticatesUsers;
use Illuminate\Support\Facades\Hash;
use Illuminate\Support\Facades\Log;
use Auth;

use App\User;
use Illuminate\Support\Facades\Validator;

class SignupController extends Controller
{
    protected $client;

    public function __construct(Client $client)
    {

        //$this->api = config('edgenet.api');
        //$this->token = config('edgenet.token');
        $this->client = $client;


    }

    public function signup(Request $request)
    {
        if (!$request->wantsJson()) {
            return response()->json(['message' => 'wrong header accept, json required'], 422);
        }

        Validator::make($request->all(), [
            'firstname' => ['required', 'string', 'max:255'],
            'lastname' => ['required', 'string', 'max:255'],
            'email' => ['required', 'string', 'email', 'max:255', 'unique:users'],
            'password' => ['required', 'string', 'min:8', 'confirmed'],

            'authority' => ['required', 'string'],

        ]);

        if (!$this->verifyAuthority($request->input('authority'))) {
            return response()->json(['message' => 'Authority does not exist'], 422);
        }

        if ($this->verifyUser($request->input('email'))) {
            return response()->json(['message' => 'User already registered'], 422);
        }

        return $this->createUser($request->all());

        //event(new Registered());

        $this->guard()->login($user);

        if ($response = $this->registered($request, $user)) {
            return $response;
        }

        return response()->json([], 201);
    }

    /*
     * Verifies authority exists in k8s
     */
    protected function verifyAuthority($authority)
    {
        return true;
    }

    /*
     * Verifies if user exists in k8s
     */
    protected function verifyUser($email)
    {
        return false;
    }

    protected function createUserss(array $data)
    {
        //dd($data);
        if (!isset($data['authority'])) {
            // we need authority
        }

        $data_authority = $data['authority'];

        if (isset($data_authority['id'])) {
            $authority = Authority::find($data_authority['id']);
        } else {
            $authority = Authority::create([
                'name' => $data_authority['name'],
                'shortname' => $data_authority['shortname'],
                'address' => $data_authority['address'],
                'zipcode' => $data_authority['zipcode'],
                'city' => $data_authority['city'],
                'country' => $data_authority['country'],
                'url' => $data_authority['url'],
            ]);
        }

        return User::create([
            'firstname' => $data['firstname'],
            'lastname' => $data['lastname'],
            'email' => $data['email'],
            'phone' => $data['phone'],
            'bio' => $data['bio'],
            'password' => Hash::make($data['password']),
            'authority_id' => $authority->id
        ]);
    }


    private function createUser(array $data)
    {
        $namespace = 'authority-' . $data['authority'];

        $roles = ['User'];

        $userSpec = [
            'apiVersion' => 'apps.edgenet.io/v1alpha',
            'kind' => 'UserRegistrationRequest',
            'metadata' => [
                'name' => $this->generateUsername($data['firstname'] . $data['lastname']),
                'namespace' => $namespace
            ],
            'spec' => [
                'firstname' => $data['firstname'],
                'lastname' => $data['lastname'],
                'email' => $data['email'],
                'phone' => isset($data['phone']) ? $data['phone'] : '-',
                'bio' => isset($data['bio']) ? $data['bio'] : '-',
                'url' => isset($data['url']) ? $data['url'] : '-',
                'roles' => $roles,
                'password' => ''
            ],

        ];
        //dd($userSpec,config('edgenet.api_prefix_crd') . '/users');

        $url = 'https://eapi-test.planet-lab.eu/apis/apps.edgenet.io/v1alpha/namespaces/'.$namespace.'/userregistrationrequests';

        //return $this->postRequest($url, $userSpec);

        try {
            return $this->client->request('POST', $url, [
                'headers' => [
//                'Authorization' => 'Bearer ' . $this->token,
                    'Accept' => 'application/json',
                ],
                'verify' => false,
                //'debug' => true
                'json' => $userSpec,
                'exceptions' => true
            ]);
        } catch (RequestException $e) {
            return response()->json([
                'message' => 'An error has occurred',
                'debug' => $e->getMessage()
            ], $e->getCode());
        }

    }

    private function generateUsername($string)
    {
        return preg_replace('/[^a-z0-9]/', '', strtolower(trim($string)));
    }
}