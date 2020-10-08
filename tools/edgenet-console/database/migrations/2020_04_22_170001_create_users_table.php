<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

class CreateUsersTable extends Migration
{
    /**
     * Run the migrations.
     *
     * @return void
     */
    public function up()
    {
        Schema::create('users', function (Blueprint $table) {
            $table->id();
            $table->timestamps();
            $table->rememberToken();

            $table->boolean('active')->default(false);

            $table->string('title')->nullable();
            $table->string('firstname');
            $table->string('lastname');

            $table->string('name')->unique();
            $table->string('authority');

            $table->string('email')->unique();
            $table->timestamp('email_verified_at')->nullable();

            $table->string('password');

            $table->string('api_token', 60)->nullable()->unique();

            $table->boolean('admin')->default(false);

            $table->boolean('nodemanager')->default(false);

        });
    }

    /**
     * Reverse the migrations.
     *
     * @return void
     */
    public function down()
    {
        Schema::dropIfExists('users');
    }
}
