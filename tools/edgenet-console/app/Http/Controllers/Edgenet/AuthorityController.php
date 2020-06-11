<?php

namespace App\Http\Controllers\Edgenet;

use App\Facades\Plc;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;

class AuthorityController extends EdgenetController
{
    private $plc;

    public function get(Request $request, $name = null)
    {

        $api = config('edgenet.api_prefix_crd') . '/authorities';

        if ($name) {
            $api .= '/' . $name;
        }

        $response = $this->getRequest($api, []);

        return response()->json(json_decode($response->getBody()), $response->getStatusCode());

    }

    public function users(Request $request, $username = null)
    {
        $name = $request->user()->edgenet['authority'];

        $api = config('edgenet.api_prefix_crd') . '/namespaces/authority-' . $name . '/users';

        if ($username) {
            $api .= '/' . $username;
        }

        $response = $this->getRequest($api, []);

        return response()->json(json_decode($response->getBody()), $response->getStatusCode());
    }

    public function post(Request $request)
    {

        $api = config('edgenet.api_prefix_crd') . '/authorities';

        //dd($request->all());
        return $this->postRequest($api, $request->all());

    }

    public function migrate(Request $request)
    {
        $request->validate([
            'password' => 'string|required'
        ]);

        $user = $request->user();
        $password = $request->input('password');

        try {
            Auth::guard('plc')->attempt([
                'email' => $user->email, 'password' => $password
            ]);
        } catch (\Exception $e) {
            return response()->json([
                'message' => 'invalid email or password',
                'debug' => $e->getMessage()
            ], 401);
        }

        // Use the first site
        if (empty($user->sites)) {
            return response()->json(['message' => 'user is not part of any site'], 400);
        }
        $site = (object)$user->sites[0];

        // check if edgenet authority exists
        $ret = $this->getRequest(config('edgenet.api_prefix_crd') . '/authorities/' . $site->name_id, []);


        if ($user->pi) {
            // PI
            if ($ret->getStatusCode() == 404) {
                // migrate authority

                $authority = [
                    'apiVersion' => 'apps.edgenet.io/v1alpha',
                    'kind' => 'Authority',
                    'metadata' => [
                        'name' => $site->name_id,
                    ],
                    'spec' => [
                        'fullname' => $site->name,
                        'shortname' => $site->shortname,
                        'url' => $site->url,
                        'address' => '-',
                        'contact' => [
                            'username' => $this->generateUserName($user),
                            'firstname' => $user->first_name,
                            'lastname' => $user->last_name,
                            'email' => $user->email,
                            'phone' => $user->phone ? $user->phone : '-'
                        ]
                    ]
                ];

                $response = $this->postRequest(config('edgenet.api_prefix_crd') . '/authorities', $authority);

            } else {
                // just migrate the user
                $response = $this->migrateUser($site, $user, $password);
            }
        } else {
            // all other users
            if ($ret->getStatusCode() != 404) {
                $response = $this->migrateUser($site, $user, $password);
            } else {
                // forbidden
                return response(403);
            }
        }

        // update PLE add tags to user
        if ($response->getStatusCode() == 201) {
            try {
                $plcret = Plc::admin()->edgenet(
                    $user->id, $this->generateUserName($user), $site->name_id
                );
            } catch (\Exception $e) {
                return response()->json([
                    'message' => 'an error occurred',
                    'debug' => $e->getMessage()
                ], 500);
            }
        }

        return response()->json(json_decode($response->getBody()), $response->getStatusCode());
    }

    public function migrateUser($site, $user, $password)
    {
        $user_name = $this->generateUserName($user);
        $user_namespace = 'authority-' . $site->name_id;

        $roles = ['User'];
        if ($user->pi) {
            $roles[] = 'Admin';
        }

        $userSpec = [
            'apiVersion' => 'apps.edgenet.io/v1alpha',
            'kind' => 'User',
            'metadata' => [
                'name' => $user_name,
                'namespace' => $user_namespace
            ],
            'spec' => [
                'firstname' => $user->first_name,
                'lastname' => $user->last_name,
                'email' => $user->email,
                'phone' => $user->phone ? $user->phone : '-',
                'bio' => $user->bio ? $user->bio : '-',
                'url' => $user->url ? $user->url : '-',
                'roles' => $roles,
                'password' => $password
            ],

        ];
        //dd($userSpec,config('edgenet.api_prefix_crd') . '/users');
        return $this->postRequest(config('edgenet.api_prefix_crd') . '/namespaces/' . $user_namespace . '/users', $userSpec);

    }

    private function generateUserName($user)
    {
        return preg_replace('/[^a-z0-9]/', '', strtolower(trim($user->first_name . $user->last_name)));
    }

}
