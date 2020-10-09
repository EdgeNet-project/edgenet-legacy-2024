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
use Illuminate\Support\Str;
use Auth;

use App\User;

class RegisterController extends Controller
{
    use RegistersUsers;

    protected $client;

    public function __construct(Client $client)
    {
        $this->client = $client;
    }

    /**
     * Registers a user locally and on kubernetes
     *
     * @param Request $request
     * @return \Illuminate\Http\JsonResponse|mixed
     */
    public function register(Request $request)
    {
//        if (!$request->wantsJson()) {
//            return response()->json(['message' => 'wrong header, json required'], 422);
//        }

        $request->validate([
            'firstname' => ['required', 'string', 'max:255'],
            'lastname' => ['required', 'string', 'max:255'],
            'email' => ['required', 'string', 'email', 'max:255', 'unique:users'],
            'password' => ['required', 'string', 'min:8', 'confirmed'],
        ]);

        // verification: user should not already exist in kubernetes
        if ($this->verifyUser($request->input('email'))) {
            return response()->json(['message' => 'User already registered'], 422);
        }

        $authority = $request->input('authority');
        if (!$request->has('authority')) {
            return response()->json(['message' => 'Authority not selected'], 400);
        }

        // generates a unique name // . '-' . Str::random(5)
        $username = $this->generateName($request->input('firstname') . '-' . $request->input('lastname')) . $this->generateRandomCode();

        if ($request->has('authority.fullname') && $request->has('authority.shortname')) {

            $authority = $this->generateName($request->input('authority.shortname'));

            // verify
            if (!$this->verifyAuthority($authority)) {
                return response()->json(['message' => 'Authority with the same shortname already exists'], 422);
            }

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
     * Verifies authority exists in kubernetes
     */
    protected function verifyAuthority($authority)
    {
        return true;
    }

    /*
     * Verifies if user exists in kubernetes
     */
    protected function verifyUser($email)
    {
        return false;
    }

    protected function createKubernetesAuthority($authority, $username, $request)
    {
        $request->validate([
            'authority.fullname' => ['required', 'string', 'max:255'],
            'authority.shortname' => ['required', 'string', 'max:255'],
        ]);

        $authoritySpec = [
            'apiVersion' => 'apps.edgenet.io/v1alpha',
            'kind' => 'AuthorityRequest',
            'metadata' => [
                'name' => $authority,
            ],
            'spec' => [
                'fullname' => $request->input('authority.fullname'),
                'shortname' => $request->input('authority.shortname'),
                'url' => $request->input('authority.url', '-'),

                'address' => [
                    'street' => $request->input('authority.street'),
                    'zip'  => $request->input('authority.zip'),
                    'city'  =>  $request->input('authority.city'),
                    'region'  => $request->input('authority.region', '-'),
                    'country'  => $request->input('authority.country'),
                ],
                'contact' => [
                    'username' => $username,
                    'firstname' => $request->input('firstname'),
                    'lastname' => $request->input('lastname'),
                    'email' => $request->input('email'),
                    'phone' => $request->input('phone', '-'),

                ]
            ],

        ];

        $url = config('edgenet.api.server') . '/apis/apps.edgenet.io/v1alpha/authorityrequests';

        Log::channel('kubernetes')->info('Authority registration : ' . $url);
        Log::channel('kubernetes')->info('Authority registration : ' . print_r($authoritySpec, true));

        try {

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

        } catch (\Exception $e) {

            Log::channel('kubernetes')->error('Authority registration : ' . $e->getMessage());
            return false;
        }


        Log::channel('kubernetes')->info('Authority registration : Status code ' . $res->getStatusCode());
        Log::channel('kubernetes')->info('Authority registration : ' . $res->getBody());

        if ($res->getStatusCode() > 208) {
            return false;
        }


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

        $url = config('edgenet.api.server') . '/apis/apps.edgenet.io/v1alpha/namespaces/'.$namespace.'/userregistrationrequests';

        //return $this->postRequest($url, $userSpec);

        Log::channel('kubernetes')->info('User registration : ' . $url);
        Log::channel('kubernetes')->info('User registration : ' . print_r($userSpec, true));

        try {

            $res = $this->client->request('POST', $url, [
                'headers' => [
//                'Authorization' => 'Bearer ' . $this->token,
                    'Accept' => 'application/json',
                ],
                'verify' => false,
                'debug' => true,
                'json' => $userSpec,
                'exceptions' => false
            ]);

        } catch (\Exception $e) {

            Log::channel('kubernetes')->error('User registration : ' . $e->getMessage());
            return false;
        }

        Log::channel('kubernetes')->error('User registration : Status code ' . $res->getStatusCode());
        Log::channel('kubernetes')->error('User registration : ' . $res->getBody());

        if ($res->getStatusCode() > 208) {
            return false;
        }

        return true;
    }

    private function generateName($string)
    {
        return preg_replace('/[^a-z0-9]/', '', strtolower(trim($string)));
    }

    private function generateRandomCode($length = 5) {
        $characters = 'abcdefghijklmnopqrstuvwxyz';
        $charactersLength = strlen($characters);
        $randomString = '';
        for ($i = 0; $i < $length; $i++) {
            $randomString .= $characters[rand(0, $charactersLength - 1)];
        }
        return $randomString;
    }
}