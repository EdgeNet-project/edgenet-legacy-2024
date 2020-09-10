<?php

Route::group(['middleware' => 'auth:api'], function() {
    Route::get('{resource}', 'ResourceController@get');
    Route::patch('{resource}', 'ResourceController@patch');
});

