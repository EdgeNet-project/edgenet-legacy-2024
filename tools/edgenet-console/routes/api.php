<?php

use Illuminate\Http\Request;
use Illuminate\Support\Facades\Route;

/*
|--------------------------------------------------------------------------
| API Routes
|--------------------------------------------------------------------------
|
| Here is where you can register API routes for your application. These
| routes are loaded by the RouteServiceProvider within a group which
| is assigned the "api" middleware group. Enjoy building your API!
|
*/

Route::middleware('auth:api')->get('/user', function (Request $request) {
    return $request->user();
});

Route::group(['middleware' => 'auth:api'], function () {
    Route::get('/user', function (Request $request) {
        return $request->user();
    });
    Route::get('/cluster', function (Request $request) {
        $contextCreate = stream_context_create([
            'ssl' => [
                'capture_peer_cert' => true,
                'allow_self_signed' => false,
                'verify_peer' => false
                ]
        ]);
        $res = stream_socket_client(str_replace('https://','ssl://', config('kubernetes.api.server')),
            $errno, $errstr, 30, STREAM_CLIENT_CONNECT, $contextCreate);
        $response = stream_context_get_params($res);

        $certInfo = openssl_x509_parse($response["options"]["ssl"]["peer_certificate"]);
        if (!openssl_x509_export($response["options"]["ssl"]["peer_certificate"], $certString)) {
            // TODO: manage errors
        }

        return [
            'server' => config('kubernetes.api.server'),
            'ca' => base64_encode($certString),
            'info' => $certInfo

        ];
    });
});
//
//// auth
//Route::group([], function () {
//    Route::post('login', 'AuthenticationController@login');
//    Route::post('password/reset', 'AuthenticationController@resetLink');
//    Route::post('password/reset/{token}', 'AuthenticationController@resetLink');
//});
//
//Route::group(['middleware' => 'auth:api'], function () {
//    Route::get('user', 'AuthenticationController@user');
//    Route::post('logout', 'AuthenticationController@logout');
//});
