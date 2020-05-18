<?php

namespace App\Http\Controllers\K8s;

use App\Http\Controllers\Controller;
use App\Providers\RouteServiceProvider;
use Illuminate\Http\Request;

use Illuminate\Foundation\Auth\AuthenticatesUsers;
use Illuminate\Support\Facades\Log;
use Auth;
use App\User;

class AuthenticationController extends Controller
{
    public function authenticate(Request $request)
    {
        Log::info(var_export($request->all(), true));

        $failed = [
            'apiVersion' => 'authentication.k8s.io/v1beta1',
            'kind' => 'TokenReview',
            'status' => [ 'authenticated' => false ]
        ];


        if ($request->input('kind') != 'TokenReview') {
            return response()->json($failed, 400);
        }

        // apiVersion authentication.k8s.io/v1beta1

        if (!$request->has('spec.token')) {
            return response()->json($failed, 401);
        }

        if (!$user = User::where('api_token', $request->input('spec.token'))->first()) {
            return response()->json($failed, 401);
        }


        return response()->json(
            [
                'apiVersion' => 'authentication.k8s.io/v1beta1',
                'kind' => 'TokenReview',
                'status' => [
                    'authenticated' => true,
                    'user' => [
                        'username' => 'readonlyuser',
//                        'uid' => '42',
                        'groups' => [
                            'default:readonlyuser'
                        ],
//                        'extra' => [
//                            'extrafield1' => [
//                                'extravalue1',
//                                'extravalue2'
//                            ]
//                        ]
                    ]
                ]
            ]
        );
    }
}
