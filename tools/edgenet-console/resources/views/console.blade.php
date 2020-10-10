<!doctype html>
<html lang="{{ app()->getLocale() }}">
<head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1">
{{--    <meta name="csrf-token" content="{{ csrf_token() }}">--}}
    <meta name="k8s-api-server" content="{{ config('edgenet.api.server', '') }}">
    <title>{{ env('APP_NAME', 'EdgeNet Console') }}</title>
    <link href="https://fonts.googleapis.com/css?family=Source+Sans+Pro" rel="stylesheet">
    <link rel="stylesheet" type="text/css" href="{{ mix('css/flag-icon.min.css') }}" />
{{--    <link rel="stylesheet" type="text/css" href="{{ mix('css/app.css') }}" />--}}
    <script type="text/javascript" src="{{ mix('js/app.js') }}" defer></script>
</head>
<body>
<div id="console"></div>
</body>
</html>
