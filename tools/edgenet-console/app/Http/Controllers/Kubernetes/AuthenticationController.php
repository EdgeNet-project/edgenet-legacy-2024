<?php

namespace App\Http\Controllers\Kubernetes;

use App\Http\Controllers\Controller;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Log;

use App\User;

/**
 * Class AuthenticationController
 * @package App\Http\Controllers\Kubernetes
 *
 * This controller will be used by the kubernetes API server to verify user tokens
 */
class AuthenticationController extends Controller
{
    public function authenticate(Request $request)
    {
        Log::channel('k8s')->info(var_export($request->all(), true));

        $failed = [
            'apiVersion' => 'authentication.k8s.io/v1beta1',
            'kind' => 'TokenReview',
            'status' => [ 'authenticated' => false ]
        ];


        if ($request->input('kind') != 'TokenReview') {
            return response()->json($failed, 400);
        }

        if (!$request->has('spec.token')) {
            return response()->json($failed, 401);
        }

        if (!$user = User::where('api_token', $request->input('spec.token'))->first()) {
            return response()->json($failed, 401);
        }

        Log::channel('k8s')->info('User ' . $user->name . ' authenticated');

        return response()->json(
            [
                'apiVersion' => 'authentication.k8s.io/v1beta1',
                'kind' => 'TokenReview',
                'status' => [
                    'authenticated' => true,
                    'user' => [
                        'username' => $user->email,
//                        'uid' => '42',
                        'groups' => [
//                            'default:readonlyuser'
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
