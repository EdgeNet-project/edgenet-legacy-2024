<?php

Route::post('/authenticate', 'AuthenticationController@authenticate');
Route::get('/dashboard', 'AuthenticationController@dashboard');


