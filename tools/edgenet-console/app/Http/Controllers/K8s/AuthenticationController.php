<?php

namespace App\Http\Controllers\K8s;

use App\Http\Controllers\Controller;
use App\Providers\RouteServiceProvider;
use Illuminate\Http\Request;

use Illuminate\Foundation\Auth\AuthenticatesUsers;
use Illuminate\Support\Facades\Log;
use Auth;

class AuthenticationController extends Controller
{
    public function authenticate(Request $request)
    {
        Log::info(var_export($request, true));

        return response()->json(
            [
                'apiVersion' => 'authentication.k8s.io/v1beta1',
                'kind' => 'TokenReview',
                'status' => [
                    'authenticated' => true,
                    'user' => [
                        'username' => 'janedoe@example.com',
                        'uid' => '42',
                        'groups' => [
                            'developers',
                            'qa'
                        ],
                        'extra' => [
                            'extrafield1' => [
                                'extravalue1',
                                'extravalue2'
                            ]
                        ]
                    ]
                ]
            ]
        );
    }
}
