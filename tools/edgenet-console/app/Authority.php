<?php

namespace App;

use Illuminate\Database\Eloquent\Model;

class Authority extends Model
{
    protected $fillable = [
        'name', 'shortname', 'address', 'zipcode', 'city', 'country', 'url'
    ];
}
