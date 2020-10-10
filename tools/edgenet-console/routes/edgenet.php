<?php
Route::get('authorities', 'ResourceController@get');
Route::patch('namespaces/registration/emailverifications/{token}', 'ResourceController@patch');
Route::patch('namespaces/authority-{namespace}/emailverifications/{token}', 'ResourceController@patch');
Route::group(['middleware' => 'auth:api'], function() {
    Route::get('{any}', 'ResourceController@get')->where('any', '.*');
    Route::patch('{any}', 'ResourceController@patch')->where('any', '.*');
});