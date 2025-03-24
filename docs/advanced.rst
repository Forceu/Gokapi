.. _advanced:

================
Advanced usage
================

.. _envvar:

********************************
Environment variables
********************************

Several environment variables can be passed to Gokapi. They can be used to modify settings that are not present during setup or to pass cloud storage credentials without saving them to the filesystem.


.. _passingenv:

Passing environment variables to Gokapi
=========================================


Docker
------

Pass the variable with the ``-e`` argument. Example for setting the port in use to *12345* and the chunk size to *60MB*:
::

 docker run -it -e GOKAPI_PORT=12345 -e GOKAPI_CHUNK_SIZE_MB=60 f0rc3/gokapi:latest


Native Deployment
-------------------

Linux / Unix
"""""""""""""

For Linux / Unix environments, execute the binary in this format:
::

  GOKAPI_PORT=12345 GOKAPI_CHUNK_SIZE_MB=60 [...] ./Gokapi

Windows
""""""""

For Windows environments, you need to run ``setx`` first, e.g.:
::

  setx GOKAPI_PORT 12345
  setx GOKAPI_CHUNK_SIZE_MB 60 database.sqlite
  [...]
  Gokapi.exe




Available environment variables
==================================


+-------------------------------+-------------------------------------------------------------------------------------+-----------------+--------------------------------------+
| Name                          | Action                                                                              | Persistent [*]_ | Default                              |
+===============================+=====================================================================================+=================+======================================+
| GOKAPI_CHUNK_SIZE_MB          | Sets the size of chunks that are uploaded in MB                                     | Yes             | 45                                   |
+-------------------------------+-------------------------------------------------------------------------------------+-----------------+--------------------------------------+
| GOKAPI_CONFIG_DIR             | Sets the directory for the config file                                              | No              | config                               |
+-------------------------------+-------------------------------------------------------------------------------------+-----------------+--------------------------------------+
| GOKAPI_CONFIG_FILE            | Sets the name of the config file                                                    | No              | config.json                          |
+-------------------------------+-------------------------------------------------------------------------------------+-----------------+--------------------------------------+
| GOKAPI_DATA_DIR               | Sets the directory for the data                                                     | Yes             | data                                 |
+-------------------------------+-------------------------------------------------------------------------------------+-----------------+--------------------------------------+
| GOKAPI_DATABASE_URL           | Sets the type and location of the database. See :ref:`Databases`                    | Yes             | sqlite://[data folder]/gokapi.sqlite |
+-------------------------------+-------------------------------------------------------------------------------------+-----------------+--------------------------------------+
| GOKAPI_LENGTH_ID              | Sets the length of the download IDs. Value needs to be 5 or more                    | Yes             | 15                                   |
+-------------------------------+-------------------------------------------------------------------------------------+-----------------+--------------------------------------+
| GOKAPI_MAX_FILESIZE           | Sets the maximum allowed file size in MB                                            | Yes             | 102400 (100GB)                       |
+-------------------------------+-------------------------------------------------------------------------------------+-----------------+--------------------------------------+
| GOKAPI_MAX_MEMORY_UPLOAD      | Sets the amount of RAM in MB that can be allocated for an upload chunk or file      | Yes             | 50                                   |
|                               |                                                                                     |                 |                                      |
|                               | Any chunk or file with a size greater than that will be written to a temporary file |                 |                                      |
+-------------------------------+-------------------------------------------------------------------------------------+-----------------+--------------------------------------+
| GOKAPI_MAX_PARALLEL_UPLOADS   | Set the amount of chunks that are uploaded in parallel for a single file            | Yes             | 4                                    |
+-------------------------------+-------------------------------------------------------------------------------------+-----------------+--------------------------------------+
| GOKAPI_PORT                   | Sets the webserver port                                                             | Yes             | 53842                                |
+-------------------------------+-------------------------------------------------------------------------------------+-----------------+--------------------------------------+
| GOKAPI_DISABLE_CORS_CHECK     | Disables the CORS check on startup and during setup, if set to "true"               | No              | false                                |
+-------------------------------+-------------------------------------------------------------------------------------+-----------------+--------------------------------------+
| GOKAPI_LOG_STDOUT             | Also outputs all log file entries to the console output                             | No              | false                                |
+-------------------------------+-------------------------------------------------------------------------------------+-----------------+--------------------------------------+
| GOKAPI_ENABLE_HOTLINK_VIDEOS  | Allow hotlinking of videos. Note: Due to buffering, playing a video might count as  | No              | false                                |
|                               |                                                                                     |                 |                                      |
|                               | multiple downloads. It is only recommend to use video hotlinking for uploads with   |                 |                                      |
|                               |                                                                                     |                 |                                      |
|                               | unlimited downloads enabled                                                         |                 |                                      |
+-------------------------------+-------------------------------------------------------------------------------------+-----------------+--------------------------------------+
| DOCKER_NONROOT                | Docker only: Runs the binary in the container as a non-root user, if set to "true"  | No              | false                                |
+-------------------------------+-------------------------------------------------------------------------------------+-----------------+--------------------------------------+
| TMPDIR                        | Sets the path which contains temporary files                                        | No              | Non-Docker: Default OS path          |
|                               |                                                                                     |                 |                                      |
|                               |                                                                                     |                 | Docker: [DATA_DIR]                   |
+-------------------------------+-------------------------------------------------------------------------------------+-----------------+--------------------------------------+


.. [*] Variables that are persistent must be submitted during the first start when Gokapi creates a new config file. They can be omitted afterwards. Non-persistent variables need to be set on every start.



All values that are described in :ref:`cloudstorage` can be passed as environment variables as well. No values are persistent; therefore, they need to be set on every start.

+---------------------------+-----------------------------------------+-----------------------------+
| Name                      | Action                                  | Example                     |
+===========================+=========================================+=============================+
| GOKAPI_AWS_BUCKET         | Sets the bucket name                    | gokapi                      |
+---------------------------+-----------------------------------------+-----------------------------+
| GOKAPI_AWS_REGION         | Sets the region name                    | eu-central-000              |
+---------------------------+-----------------------------------------+-----------------------------+
| GOKAPI_AWS_KEY            | Sets the API key                        | 123456789                   |
+---------------------------+-----------------------------------------+-----------------------------+
| GOKAPI_AWS_KEY_SECRET     | Sets the API key secret                 | abcdefg123                  |
+---------------------------+-----------------------------------------+-----------------------------+
| GOKAPI_AWS_ENDPOINT       | Sets the endpoint                       | eu-central-000.provider.com |
+---------------------------+-----------------------------------------+-----------------------------+
| GOKAPI_AWS_PROXY_DOWNLOAD | If true, users will not be redirected   | true (default:false)        |
|                           |                                         |                             |
|                           | to a pre-signed S3 URL for downloading. |                             |
|                           |                                         |                             |
|                           | Instead, Gokapi will download the file  |                             |
|                           |                                         |                             |
|                           | and proxy it to the user                |                             |
+---------------------------+-----------------------------------------+-----------------------------+



.. _databases:


********************************
Databases
********************************

By default, Gokapi uses an SQLite database for data storage, which should suffice for most use cases. Additionally, Redis is available as an experimental option.



Migrating to a different database
=================================

To switch to a different database, Gokapi provides a migration tool. By running:

::

 gokapi --migrate [old Database URL] [new Database URL]
 
all existing data, except for user sessions, will be transferred to the new database. After the migration, you will need to rerun the setup and specify the new database location. For details on the correct database URL format, refer to the section :ref:`databaseUrl`.

For Docker users, the command is:
::

 docker run --rm -v gokapi-data:/app/data f0rc3/gokapi:latest /app/run.sh [old Database URL] [new Database URL]


.. _databaseUrl:

Database URL format
---------------------------------

Database URLs must start with either ``sqlite://`` or ``redis://``.


For SQLite, the path to the database follows the prefix. No additional options are allowed.

For Redis, the URL can include authentication credentials (username and password), an optional prefix for keys, and parameter to use SSL.


Redis URL Format
---------------------------------

A Redis URL has the following structure:
::

 redis://[username:password@]host[:port][?options]
 
* username: (optional) The username for authentication.
* password: (optional) The password for authentication.
* host: (required) The address of the Redis server.
* port: (optional) The port of the Redis server (default is 6379).
* options: (optional) Additional options such as SSL (``ssl=true``) and key prefix (``prefix=``).


Examples
---------------------------------

Migrating SQLite (``/app/data/gokapi.sqlite``) to Redis (``127.0.0.1:6379``):


::

 gokapi --migrate sqlite:///app/data/gokapi.sqlite redis://127.0.0.1:6379

Migrating SQLite (``/app/data/gokapi.sqlite``) to SQLite (``./data/gokapi.sqlite``):

::

 gokapi --migrate sqlite:///app/data/gokapi.sqlite sqlite://./data/gokapi.sqlite
 
Migrating Redis (``127.0.0.1:6379, User: test, Password: 1234, Prefix: gokapi_, using SSL``) to SQLite (``./data/gokapi.sqlite``):


::

 gokapi --migrate "redis://test:1234@127.0.0.1:6379?prefix=gokapi_&ssl=true" sqlite://./data/gokapi.sqlite

.. _api:


********************************
API
********************************

Gokapi offers an API that can be reached at ``http(s)://your.gokapi.url/api/``. You can find the current documentation with an overview of all API functions and examples at ``http(s)://your.gokapi.url/apidocumentation/``.


Interacting with the API
============================


All API calls will need an API key as authentication. An API key can be generated in the web UI in the menu "API". The API key needs to be passed as a header.

Example: Getting a list of all stored files with curl
::

 curl -X GET "https://your.gokapi.url/api/files/list" -H "accept: application/json" -H "apikey: secret"

Some calls expect parameters as form/post parameter, others as headers. Please refer to the current API documentation.

Example: Uploading a file
::

 curl -X POST "https://your.gokapi.url/api/files/add" -H "accept: application/json" -H "apikey: secret" -H "Content-Type: multipart/form-data" -F "allowedDownloads=1" -F "expiryDays=5" -F "password=" -F "file=@yourfile.dat"

Example: Deleting a file
::

 curl -X DELETE "https://your.gokapi.url/api/files/delete" -H "accept: */*" -H "id: PFnh2DlQRS2PVKM" -H "apikey: secret"



.. _chunksizes:

*****************************************************************************
Chunk Sizes / Considerations for servers with limited or high amount of RAM
*****************************************************************************

By default, Gokapi uploads files in 45MB chunks stored in RAM. Up to 4 chunks are sent in parallel to enhance upload speed, requiring up to 200MB of RAM per file during upload in the standard configuration.

Servers with limited RAM
================================

To conserve RAM, you can either 

* configure Gokapi to save the chunks on disk instead of RAM, by setting the ``MaxMemory`` setting to a value lower than your chunk size
* reduce the chunk size by setting the ``ChunkSize`` to a lower value
* decrease the amount of parallel uploads by setting ``MaxParallelUploads`` to a lower value

Refer to :ref:`chunk_config` for instructions on changing these values.

Servers with high amount of RAM
================================

If your server has a lot of available RAM, you can improve upload speed by increasing the chunk size, which reduces overhead during upload.

* Increase the chunk size by setting the ``ChunkSize`` to a larger value
* Make sure that the ``MaxMemory`` setting is a higher value than your chunk size
* Consider increasing the amount of parallel uploads by setting ``MaxParallelUploads`` to a higher value


Refer to :ref:`chunk_config` for instructions on changing these values.

.. note::
   Ensure your reverse proxy and CDN (if applicable) support the chosen chunk size. Cloudflare users on the free tier are limited to 100MB file chunks.


.. _chunk_config:


Changing the configuration
============================

If you have not completed the Gokapi setup yet, you can set all the values mentioned above using environment variables. See :ref:`passingenv` for instructions. If the setup is complete, Gokapi will ignore these environment variables, and you'll need to modify the configuration file (by default: ``config.json`` in the folder ``config``). See the table below on how to change the values:


+----------------------------------------+-----------------------------+--------------------------+---------+
| Configuration                          | Environment Variable        | Configuration File Entry | Default |
+========================================+=============================+==========================+=========+
| Chunk size for uploads                 | GOKAPI_CHUNK_SIZE_MB        | ChunkSize                | 45      |
+----------------------------------------+-----------------------------+--------------------------+---------+
| Maximum size for chunks or whole files | GOKAPI_MAX_MEMORY_UPLOAD    | MaxMemory                | 50      |
|                                        |                             |                          |         |
| to store in RAM during upload          |                             |                          |         |
+----------------------------------------+-----------------------------+--------------------------+---------+
| Parallel uploads per file              | GOKAPI_MAX_PARALLEL_UPLOADS | MaxParallelUploads       | 4       |
+----------------------------------------+-----------------------------+--------------------------+---------+




********************************
Automatic Deployment
********************************

It is possible to deploy Gokapi without having to run the setup. You will need to complete the setup on a temporary instance first. This is to create the configuration files, which can then be used for deployment.


Configuration Files
============================


The configuration consists of up to two files in the configuration directory (default: ``config``). All files can be read-only, however ``config.json`` might need write access in some situations.

cloudconfig.yml
------------------------

Stores the access data for cloud storage. This can be reused without modification, however all fields can also be set with environment variables. The file does not exist if no cloud storage is used and can always be read-only.


config.json
------------------------

Contains the server configuration. If you want to deploy Gokapi in multiple instances for redundancy  (e.g. all instances share the same data), then the configuration file can be reused without modification. Otherwise you need to modify it before deploying (see below). Can be read-only, but might need write access when upgrading Gokapi to a newer version. Needs write access when re-running setup or changing the admin password.


Modifying config.json to deploy without setup
====================================================

If you want to deploy Gokapi to multiple instances that contain different data, you have to modify the config.json. Open it and change the following fields:

+-----------+------------------------------------------------------------+----------------------+
| Field     | Operation                                                  | Example              |
+===========+============================================================+======================+
| SaltAdmin | Change to empty value                                      | "SaltAdmin": "",     |
+-----------+------------------------------------------------------------+----------------------+
| SaltFiles | Change to empty value                                      | "SaltFiles": "",     |
+-----------+------------------------------------------------------------+----------------------+
| Password  | Change to empty value                                      | "Password": "",      |
+-----------+------------------------------------------------------------+----------------------+
| Username  | Change to the username of your preference,                 | "Username": "admin", |
|           |                                                            |                      |
|           | if you are using internal username/password authentication |                      |
+-----------+------------------------------------------------------------+----------------------+

Setting an admin password
====================================================

If you are using internal username/password authentication, run the binary with the parameter ``--deployment-password [YOUR_PASSWORD]``. This sets the password and also generates a new salt for the password. This has to be done before Gokapi is run for the first time on the new instance. Alternatively you can do this on the orchestrating machine and then copy the configuration file to the new instance.

If you are using a Docker image, this has to be done by starting a container with the entrypoint ``/app/run.sh``, for example: ::

 docker run --rm -v gokapi-data:/app/data -v gokapi-config:/app/config  f0rc3/gokapi:latest /app/run.sh --deployment-password newPassword


********************************
Customising
********************************

By default, all files are included in the executable. If you want to change the layout (e.g. add your company logo or change the app name etc.), follow these steps:

1. Download the source code for the Gokapi version you are using. It is either attached to the specific release  `on Github <https://github.com/Forceu/Gokapi/releases>`_ or you can clone the repository and checkout the tag for the specific version.
2. Copy either the folder ``static``, ``templates`` or both from the ``internal/webserver/web`` folder to the directory where the executable is located (if you are using Docker, mount the folders into the the ``/app/`` directory, e.g. ``/app/templates``).
3. Make changes to the folders. ``static`` contains images, CSS files and JavaScript. ``templates`` contains the HTML code.
4. Restart the server. If the folders exist, the server will use the local files instead of the embedded files.
5. Optional: To embed the files permanently, copy the modified files back to the original folders and recompile with ``go generate ./...`` and then ``go build github.com/forceu/gokapi/cmd/gokapi``.
