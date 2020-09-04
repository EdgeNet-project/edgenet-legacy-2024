<?php

namespace App\Http\Controllers\Kubernetes;


use App\Http\Controllers\Controller;
use Illuminate\Http\Request;

/**
 * Class ApiController
 * @package App\Http\Controllers\Kubernetes
 *
 * We use this proxy controller to communicate with Kubernetes API
 * and aoid CORS related issues.
 *
 */
class ApiController extends Controller
{
    public function get(Request $request)
    {

    }
}