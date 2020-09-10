<?php

Route::get('{resource}', 'ResourceController@get');
Route::patch('{any}', 'ResourceController@patch')->where('any', '.*');