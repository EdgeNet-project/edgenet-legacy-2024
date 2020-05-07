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
