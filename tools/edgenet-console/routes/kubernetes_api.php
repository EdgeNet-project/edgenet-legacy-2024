<?php

Route::group(['middleware' => 'auth:api'], function() {
    Route::get('{any}', 'ResourceController@get')->where('any', '.*');
    Route::patch('{resource}', 'ResourceController@patch');
});

