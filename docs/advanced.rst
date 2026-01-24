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

For Windows environments, you need to run ``set`` first, e.g.:
::

  set GOKAPI_PORT=12345
  set GOKAPI_CHUNK_SIZE_MB=60
  [...]
  Gokapi.exe




Available environment variables
==================================


+--------------------------------+-------------------------------------------------------------------------------------+-----------------+-----------------------------+
| Name                           | Action                                                                              | Persistent [*]_ | Default                     |
+================================+=====================================================================================+=================+=============================+
| GOKAPI_CHUNK_SIZE_MB           | Sets the size of chunks that are uploaded in MB                                     | Yes             | 45                          |
+--------------------------------+-------------------------------------------------------------------------------------+-----------------+-----------------------------+
| GOKAPI_CONFIG_DIR              | Sets the directory for the config file                                              | No              | config                      |
+--------------------------------+-------------------------------------------------------------------------------------+-----------------+-----------------------------+
| GOKAPI_CONFIG_FILE             | Sets the name of the config file                                                    | No              | config.json                 |
+--------------------------------+-------------------------------------------------------------------------------------+-----------------+-----------------------------+
| GOKAPI_DATA_DIR                | Sets the directory for the data                                                     | Yes             | data                        |
+--------------------------------+-------------------------------------------------------------------------------------+-----------------+-----------------------------+
| GOKAPI_DISABLE_CORS_CHECK      | Disables the CORS check on startup and during setup, if set to true                 | No              | false                       |
+--------------------------------+-------------------------------------------------------------------------------------+-----------------+-----------------------------+
| GOKAPI_ENABLE_HOTLINK_VIDEOS   | Allow hotlinking of videos. Note: Due to buffering, playing a video might count as  | No              | false                       |
|                                |                                                                                     |                 |                             |
|                                | multiple downloads. It is only recommended to use video hotlinking for uploads with |                 |                             |
|                                |                                                                                     |                 |                             |
|                                | unlimited downloads enabled                                                         |                 |                             |
+--------------------------------+-------------------------------------------------------------------------------------+-----------------+-----------------------------+
| GOKAPI_GUEST_UPLOAD_BY_DEFAULT | Allows all users by default to create file requests, if set to true                 | No              | false                       |
+--------------------------------+-------------------------------------------------------------------------------------+-----------------+-----------------------------+
| GOKAPI_LENGTH_HOTLINK_ID       | Sets the length of the hotlink IDs. Value must be 8 or greater                      | No              | 40                          |
+--------------------------------+-------------------------------------------------------------------------------------+-----------------+-----------------------------+
| GOKAPI_LENGTH_ID               | Sets the length of the download IDs. Value must be 5 or greater                     | No              | 15                          |
+--------------------------------+-------------------------------------------------------------------------------------+-----------------+-----------------------------+
| GOKAPI_LOG_STDOUT              | Also outputs all log file entries to the console output, if set to true             | No              | false                       |
+--------------------------------+-------------------------------------------------------------------------------------+-----------------+-----------------------------+
| GOKAPI_MAX_FILESIZE            | Sets the maximum allowed file size in MB                                            | Yes             | 102400                      |
|                                |                                                                                     |                 |                             |
|                                | Default 102400 = 100GB                                                              |                 |                             |
+--------------------------------+-------------------------------------------------------------------------------------+-----------------+-----------------------------+
| GOKAPI_MAX_FILES_GUESTUPLOAD   | Sets the maximum number of files that can be uploaded per file requests created by  | No              | 100                         |
|                                |                                                                                     |                 |                             |
|                                | non-admin users                                                                     |                 |                             |
|                                |                                                                                     |                 |                             |
|                                | Set to 0 to allow unlimited file count for all users                                |                 |                             |
+--------------------------------+-------------------------------------------------------------------------------------+-----------------+-----------------------------+
| GOKAPI_MAX_MEMORY_UPLOAD       | Sets the amount of RAM in MB that can be allocated for an upload chunk or file      | Yes             | 50                          |
|                                |                                                                                     |                 |                             |
|                                | Any chunk or file with a size greater than that will be written to a temporary file |                 |                             |
+--------------------------------+-------------------------------------------------------------------------------------+-----------------+-----------------------------+
| GOKAPI_MAX_PARALLEL_UPLOADS    | Set the number of chunks that are uploaded in parallel for a single file            | Yes             | 3                           |
+--------------------------------+-------------------------------------------------------------------------------------+-----------------+-----------------------------+
| GOKAPI_MAX_SIZE_GUESTUPLOAD    | Sets the maximum file size for file requests created by                             | No              | 10240                       |
|                                |                                                                                     |                 |                             |
|                                | non-admin users                                                                     |                 |                             |
|                                |                                                                                     |                 |                             |
|                                | Set to 0 to allow files with a size of up to a value set with GOKAPI_MAX_FILESIZE   |                 |                             |
|                                |                                                                                     |                 |                             |
|                                | for all users                                                                       |                 |                             |
|                                |                                                                                     |                 |                             |
|                                | Default 10240 = 10GB                                                                |                 |                             |
+--------------------------------+-------------------------------------------------------------------------------------+-----------------+-----------------------------+
| GOKAPI_MIN_FREE_SPACE          | Sets the minium free space on the disk in MB for accepting an upload                | No              | 400                         |
+--------------------------------+-------------------------------------------------------------------------------------+-----------------+-----------------------------+
| GOKAPI_MIN_LENGTH_PASSWORD     | Sets the minium password length. Value must be 6 or greater                         | No              | 8                           |
+--------------------------------+-------------------------------------------------------------------------------------+-----------------+-----------------------------+
| GOKAPI_PORT                    | Sets the webserver port                                                             | Yes             | 53842                       |
+--------------------------------+-------------------------------------------------------------------------------------+-----------------+-----------------------------+
| TMPDIR                         | Sets the path which contains temporary files                                        | No              | Non-Docker: Default OS path |
|                                |                                                                                     |                 |                             |
|                                |                                                                                     |                 | Docker: [DATA_DIR]          |
+--------------------------------+-------------------------------------------------------------------------------------+-----------------+-----------------------------+
| DOCKER_NONROOT                 | DEPRECATED.                                                                         | No              | false                       |
|                                |                                                                                     |                 |                             |
|                                | Docker only: Runs the binary in the container as a non-root user, if set to "true"  |                 |                             |
+--------------------------------+-------------------------------------------------------------------------------------+-----------------+-----------------------------+

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



.. _databases:


********************************
Databases
********************************

By default, Gokapi uses an SQLite database for data storage, which should suffice for most use cases. Additionally, Redis is available as an experimental option.



Migrating to a different database
=================================

To switch to a different database, Gokapi provides a migration tool. By running:

::

 gokapi migrate-database --source [old Database URL] --destination [new Database URL]
 
all existing data, except for user sessions, will be transferred to the new database. After the migration, you will need to rerun the setup and specify the new database location. For details on the correct database URL format, refer to the section :ref:`databaseUrl`.

For Docker users, the command is:
::

 docker run --rm -v gokapi-data:/app/data f0rc3/gokapi:latest /app/run.sh migrate-database --source [old Database URL] --destination [new Database URL]


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

 gokapi migrate-database --source sqlite:///app/data/gokapi.sqlite --destination redis://127.0.0.1:6379

Migrating SQLite (``/app/data/gokapi.sqlite``) to SQLite (``./data/gokapi.sqlite``):

::

 gokapi migrate-database --source sqlite:///app/data/gokapi.sqlite --destination sqlite://./data/gokapi.sqlite
 
Migrating Redis (``127.0.0.1:6379, User: test, Password: 1234, Prefix: gokapi_, using SSL``) to SQLite (``./data/gokapi.sqlite``):


::

 gokapi migrate-database --source "redis://test:1234@127.0.0.1:6379?prefix=gokapi_&ssl=true" --destination sqlite://./data/gokapi.sqlite



.. _clitool:


********************************
CLI Tool
********************************

Gokapi also has a CLI tool that allows uploads from the command line. Binaries are avaible on the `Github release page <https://github.com/Forceu/Gokapi/releases>`_ for Linux, Windows and MacOS. To compile it yourself, download the repository and run ``make build-cli`` in the top directory.

Alternatively you can use the tool with Docker, although it will be slightly less user-friendly.

.. note::

  Gokapi v2.1.0 or newer is required to use the CLI tool.

Login
=================================

First you need to login with the command ``gokapi-cli login``. You will then be asked for your server URL and a valid API key with upload permission. If end-to-end encryption is enabled, you will also need to enter your encyption key. By default the login data is saved to ``gokapi-cli.json``, but you can define a different location with the ``-c`` parameter.


To logout, either delete the configuration file or run ``gokapi-cli logout``.

.. warning::

   The configuration file contains the login data as plain text.


Docker
---------------------------------

If you are using Docker, your config will be saved to ``/app/config/config.json`` by default, but the location can be changed. To login, execute the following command:
::

  docker run -it --rm -v gokapi-cli-config:/app/config docker.io/f0rc3/gokapi-cli:latest login

The volume ``gokapi-cli-config:/app/config`` is not required if you re-use the container, but it is still highly recommended. If a volume is not mounted, you will need to log in again after every new container creation.
  


.. _clitool-upload-file:

Uploading a file
=================================


To upload a file, simply run ``gokapi-cli upload -f /path/to/file``. By default the files are encrypted (if enabled) and stored without any expiration. These additional parameters are available:

+------------------------------------+---------------------------------------------------+
| Parameter                          | Effect                                            |
+====================================+===================================================+
|  \-\-json, -j                      | Only outputs in JSON format, unless upload failed |
+------------------------------------+---------------------------------------------------+
|  \-\-disable-e2e, -x               | Disables end-to-end encryption for this upload    |
+------------------------------------+---------------------------------------------------+
|  \-\-expiry-days, -e [number]      | Sets the expiry date of the file in days          |
+------------------------------------+---------------------------------------------------+
|  \-\-expiry-downloads, -d [number] | Sets the allowed downloads                        |
+------------------------------------+---------------------------------------------------+
|  \-\-password, -p [string]         | Sets a password                                   |
+------------------------------------+---------------------------------------------------+
|  \-\-name, -n [string]             | Sets a different filename for uploaded file       |
+------------------------------------+---------------------------------------------------+
|  \-\-configuration, -c [path]      | Use the configuration file specified              |
+------------------------------------+---------------------------------------------------+

**Example:** Uploading the file ``/tmp/example``. It will expire in 10 days, has unlimited downloads and requires the password ``abcd``:
::

 gokapi-cli upload -f /tmp/example --expiry-days 10 --password abcd
  
  
.. warning::

   If you are using end-to-end encryption, do not upload other encrypted files simultaneously to avoid race conditions. 
   
   
   
Docker
---------------------------------

As a Docker container cannot access your host files without a volume, you will need to mount the folder that contains your file to upload and then specify the internal file path with ``-f``. If no ``-f`` parameter is supplied and only a single file exists in the container folder ``/upload/``, this file will be uploaded.

**Example:** Uploading the file ``/tmp/example``. It will expire after 5 downloads, has no time expiry and has no password.
::

 docker run --rm -v gokapi-cli-config:/app/config -v /tmp/:/upload/ docker.io/f0rc3/gokapi-cli:latest upload -f /upload/example --expiry-downloads 5 

**Example:** Uploading the file ``/tmp/single/example``. There is no other file in the folder ``/tmp/single/``.
::

 docker run --rm -v gokapi-cli-config:/app/config -v /tmp/single/:/upload/ docker.io/f0rc3/gokapi-cli:latest upload

**Example:** Uploading the file ``/tmp/multiple/example``. There are other files in the folder ``/tmp/multiple/``.
::

 docker run --rm -v gokapi-cli-config:/app/config -v /tmp/multiple/example:/upload/example docker.io/f0rc3/gokapi-cli:latest upload
   



Uploading a directory
=================================


By running ``gokapi-cli upload-dir -D /path/to/directory/``, gokapi-cli compresses the given folder as a zip file and then uploads it. By default the foldername is used for the name of the zip file. Also the file is encrypted (if enabled) and stored without any expiration.

In addition to all the options seen in chapter :ref:`clitool-upload-file`, the following optional options are also available:

+------------------------------------+---------------------------------------------------+
| Parameter                          | Effect                                            |
+====================================+===================================================+
|  \-\-tmpfolder, -t                 | Sets the path for temporary files.                |
+------------------------------------+---------------------------------------------------+


**Example:** Uploading the folder ``/tmp/example/``. It will expire in 10 days, has unlimited downloads and requires the password ``abcd``:
::

 gokapi-cli upload-dir -D /tmp/example --expiry-days 10 --password abcd
  
  
.. warning::

   If you are using end-to-end encryption, do not upload other encrypted files simultaneously to avoid race conditions. 
   
   
   
Docker
---------------------------------

As a Docker container cannot access your host files without a volume, you will need to mount the folder that contains your file to upload and then specify the internal path with ``-D``. If no ``-D`` parameter is supplied, the folder ``/upload/`` will be uploaded (if it contains any files).

**Example:** Uploading the folder ``/tmp/example/``. It will expire after 5 downloads, has no time expiry and has no password.
::

 docker run --rm -v gokapi-cli-config:/app/config -v /tmp/example/:/upload/example docker.io/f0rc3/gokapi-cli:latest upload-dir -D /upload/example/ --expiry-downloads 5 

**Example:** Uploading the folder ``/tmp/another/example`` and setting the filename to ``example.zip``
::

 docker run --rm -v gokapi-cli-config:/app/config -v /tmp/another/example:/upload/ docker.io/f0rc3/gokapi-cli:latest upload-dir -n "example.zip"


   
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

By default, Gokapi uploads files in 45MB chunks stored in RAM. Up to 3 chunks are sent in parallel to enhance upload speed, requiring up to 150MB of RAM per file during upload in the standard configuration.

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
* Increasing the amount of parallel uploads by setting ``MaxParallelUploads`` to a higher value is possible, but not recommended if using HTTP1.1 (see warning below). 


Refer to :ref:`chunk_config` for instructions on changing these values.

.. note::
   Ensure your reverse proxy and CDN (if applicable) support the chosen chunk size. Cloudflare users on the free tier are limited to 100MB file chunks.
   
.. warning::
   Most browsers do not support more than 6 open connections with HTTP1.1 (which is the default connection). There is always one connection per tab used in the background for receiving status updates, therefore increasing the ``MaxParallelUploads`` value is not recommended in that case. If you require more connections, you can consider switching to HTTP2.


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
| Parallel uploads per file              | GOKAPI_MAX_PARALLEL_UPLOADS | MaxParallelUploads       | 3       |
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

If you want to change the layout (e.g. add your company logo or add/disable certain features), follow these steps:

1. Create a new folder named ``custom`` where your executable is. When using Docker, mount a new folder to ``/app/custom/``. Any file in this directory will be publicly available in the sub-URL ``/custom/``.
2. To have custom CSS included, create a file in the folder named ``custom.css``. The CSS will be applied to all pages.
3. To have custom JavaScript included, create the file ``public.js`` for all public pages and/or ``admin.js`` for all admin-related pages. Please note that the ``admin.js`` will be readable to all users.
4. In order to prevent caching issues, you can version your files by creating the file ``version.txt`` with a version number.
5. To have a custom Favicon, place a 512x512 PNG image named ``favicon.png`` in the folder ``custom``.
6. Restart the server. If the folders exist, the server will now add the local files.

Optional: If you require further changes or want to embedded the changes permanently, you can clone the source code and then modify the templates in ``internal/webserver/web/templates``. Afterwards run ``make`` to build a new binary with these changes.
