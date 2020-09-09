<?php

Route::group(['middleware' => 'auth:api'], function() {
    Route::get('{resource}', 'ResourceController@get');
});

