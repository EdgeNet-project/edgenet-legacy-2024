<?php

use Illuminate\Support\Facades\Route;
use Illuminate\Support\Facades\Log;
use Illuminate\Http\Request;
/*
|--------------------------------------------------------------------------
| Web Routes
|--------------------------------------------------------------------------
|
| Here is where you can register web routes for your application. These
| routes are loaded by the RouteServiceProvider within a group which
| contains the "web" middleware group. Now create something great!
|
*/
//
//Route::get('/password/reset/{token?}', function () {
//    return view('console');
//})->where('token', '.*');
//
//Auth::routes(['register' => false]);

//Auth::routes();

Route::post('login', 'Auth\LoginController@login');
Route::post('logout', 'Auth\LoginController@logout')->name('logout');
Route::post('register', 'Kubernetes\RegisterController@register');
Route::post('password/email', 'Auth\ForgotPasswordController@sendResetLinkEmail')->name('password.email');
Route::post('password/reset', 'Auth\ResetPasswordController@reset')->name('password.update');
Route::post('password/confirm', 'Auth\ConfirmPasswordController@confirm');

//Route::post('/signup', 'Kubernetes\SignupController@signup');

Route::get('/{any?}', function () {
    return view('console');
})->where('any', '.*');
