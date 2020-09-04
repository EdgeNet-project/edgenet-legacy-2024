<?php

namespace App\Http\Controllers\Kubernetes;

use App\Http\Controllers\Controller;
use Illuminate\Auth\Events\Registered;
use Illuminate\Foundation\Auth\RegistersUsers;
use Illuminate\Http\Request;
use GuzzleHttp\Client;

use Illuminate\Foundation\Auth\AuthenticatesUsers;
use Illuminate\Support\Facades\Hash;
use Illuminate\Support\Facades\Log;
use Auth;

use App\User;

class SignupController extends Controller
{
    use RegistersUsers;

    protected $client;

    public function __construct(Client $client)
    {
        $this->client = $client;
    }

    public function signup(Request $request)
    {
        if (!$request->wantsJson()) {
            return response()->json(['message' => 'wrong header accept, json required'], 422);
        }

        $request->validate([
            'firstname' => ['required', 'string', 'max:255'],
            'lastname' => ['required', 'string', 'max:255'],
            'email' => ['required', 'string', 'email', 'max:255', 'unique:users'],
            'password' => ['required', 'string', 'min:8', 'confirmed'],
        ]);


        if ($this->verifyUser($request->input('email'))) {
            return response()->json(['message' => 'User already registered'], 422);
        }

        $authority = $request->input('authority');
        $username = $this->generatName($request->input('firstname') . $request->input('lastname'));

        if (!$authority) {
            $request->validate([
                'fullname' => ['required', 'string', 'max:255'],
                'shortname' => ['required', 'string', 'max:255'],
            ]);

            $authority = $this->generatName($request->input('shortname'));

            if (!$this->createKubernetesAuthority($authority, $username, $request)) {
                return response()->json(['message' => 'Can\'t create authority (kubernetes)'], 422);
            }
        } else {

            if (!$this->verifyAuthority($authority)) {
                return response()->json(['message' => 'Authority does not exist'], 422);
            }

            if (!$this->createKubernetesUser($authority, $username, $request)) {
                return response()->json(['message' => 'Can\'t create user (kubernetes)'], 422);
            }
        }

        /**
         * Creates local user
         */
        event(new Registered(
            $user = User::create([
                'firstname' => $request->input('firstname'),
                'lastname' => $request->input('lastname'),
                'email' => $request->input('email'),

                'name' => $username,
                'authority' => $authority,

                'password' => Hash::make($request->input('password')),
            ])
        ));

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

    protected function createKubernetesAuthority($authority, $username, $request)
    {
        $authoritySpec = [
            'apiVersion' => 'apps.edgenet.io/v1alpha',
            'kind' => 'AuthorityRequest',
            'metadata' => [
                'name' => $authority,
            ],
            'spec' => [
                'fullname' => $request->input('fullname'),
                'shortname' => $request->input('shortname'),
                'url' => $request->input('url', '-'),

                'address' => [
                    'street' => $request->input('street'),
                    'zip'  => $request->input('zip'),
                    'city'  =>  $request->input('city'),
                    'region'  => $request->input('region', '-'),
                    'country'  => $request->input('country'),
                ],
                'contact' => [
                    'username' => $username,
                    'firstname' => $request->input('firstname'),
                    'lastname' => $request->input('lastname'),
                    'email' => $request->input('email'),
                    'phone' => $request->input('phone', '-'),

                ],
            ],

        ];

        $url = env('EDGENET_API_SERVER') . '/apis/apps.edgenet.io/v1alpha/authorityrequests';

        $res = $this->client->request('POST', $url, [
            'headers' => [
//                'Authorization' => 'Bearer ' . $this->token,
                'Accept' => 'application/json',
            ],
            'verify' => false,
            //'debug' => true
            'json' => $authoritySpec,
            'exceptions' => false
        ]);

        Log::channel('k8s')->info($url);
        Log::channel('k8s')->info(print_r($authoritySpec, true));
        Log::channel('k8s')->info($res->getBody());

        if ($res->getStatusCode()) {

        }
        Log::channel('k8s')->error($res->getStatusCode());

        return true;
    }


    private function createKubernetesUser($authority, $username, $request)
    {
        $namespace = 'authority-' . $authority;

        $roles = ['User'];

        $userSpec = [
            'apiVersion' => 'apps.edgenet.io/v1alpha',
            'kind' => 'UserRegistrationRequest',
            'metadata' => [
                'name' => $username,
                'namespace' => $namespace
            ],
            'spec' => [
                'firstname' => $request->input('firstname'),
                'lastname' => $request->input('lastname'),
                'email' => $request->input('email'),
//                'phone' => $request->input('phone', '-'),
//                'bio' => $request->input('bio','-'),
//                'url' => $request->input('url','-'),
//                'roles' => $roles,
            ],

        ];
        //dd($userSpec,config('edgenet.api_prefix_crd') . '/users');

        $url = env('EDGENET_API_SERVER') . '/apis/apps.edgenet.io/v1alpha/userregistrationrequests';

        //return $this->postRequest($url, $userSpec);

        $res = $this->client->request('POST', $url, [
                'headers' => [
//                'Authorization' => 'Bearer ' . $this->token,
                    'Accept' => 'application/json',
                ],
                'verify' => false,
                //'debug' => true
                'json' => $userSpec,
                'exceptions' => false
            ]);

        Log::channel('k8s')->info('url: ' . $url);

        if ($res->getStatusCode() > 208) {
            Log::channel('k8s')->error(print_r($userSpec, true));
            Log::channel('k8s')->error($res->getBody());
            return false;
        }

        return true;
    }

    private function generatName($string)
    {
        return preg_replace('/[^a-z0-9]/', '', strtolower(trim($string)));
    }
}