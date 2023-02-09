.. _advanced:

================
Advanced usage
================

.. _envvar:

********************************
Environment variables
********************************

Environment variables can be passed to Gokapi - that way you can set it up without any interaction and pass cloud storage credentials without saving them to the filesystem.


.. _passingenv:

Passing environment variables to Gokapi
===============================================


Docker
------

Pass the variable with the ``-e`` argument. Example for setting the username to *admin* and the password to *123456*:
::

 docker run -it -e GOKAPI_USERNAME=admin -e GOKAPI_PASSWORD=123456 f0rc3/gokapi:latest


Bare Metal
----------

Linux / Unix
"""""""""""""

For Linux / Unix environments, execute the binary in this format:
::

  GOKAPI_USERNAME=admin GOKAPI_PASSWORD=123456 [...] ./Gokapi

Windows
""""""""

For Windows environments, you need to run ``setx`` first, e.g.:
::

  setx GOKAPI_USERNAME admin
  setx GOKAPI_PASSWORD 123456
  [...]
  Gokapi.exe




Available environment variables
==================================


+--------------------------+------------------------------------------------------------------------------+-------------+-----------------------------+
| Name                     | Action                                                                       | Persistent* | Default                     |
+==========================+==============================================================================+=============+=============================+
| GOKAPI_CONFIG_DIR        | Sets the directory for the config file                                       | No          | config                      |
+--------------------------+------------------------------------------------------------------------------+-------------+-----------------------------+
| GOKAPI_CONFIG_FILE       | Sets the name of the config file                                             | No          | config.json                 |
+--------------------------+------------------------------------------------------------------------------+-------------+-----------------------------+
| GOKAPI_DATA_DIR          | Sets the directory for the data                                              | Yes         | data                        |
+--------------------------+------------------------------------------------------------------------------+-------------+-----------------------------+
| GOKAPI_LENGTH_ID         | Sets the length of the download IDs. Value needs to be 5 or more             | Yes         | 15                          |
+--------------------------+------------------------------------------------------------------------------+-------------+-----------------------------+
| GOKAPI_MAX_FILESIZE      | Sets the maximum allowed file size in MB                                     | Yes         | 102400 (100GB)              |
+--------------------------+------------------------------------------------------------------------------+-------------+-----------------------------+
| GOKAPI_MAX_MEMORY_UPLOAD | Sets the amount of RAM in MB that can be allocated for an upload.            | Yes         | 20                          |
|                          | Any upload with a size greater than that will be written to a temporary file |             |                             |
+--------------------------+------------------------------------------------------------------------------+-------------+-----------------------------+
| GOKAPI_PORT              | Sets the webserver port                                                      | Yes         | 53842                       |
+--------------------------+------------------------------------------------------------------------------+-------------+-----------------------------+
| TMPDIR                   | Sets the path which contains temporary files                                 | No          | Non-Docker: Default OS path |
|                          |                                                                              |             | Docker:     [DATA_DIR]      |
+--------------------------+------------------------------------------------------------------------------+-------------+-----------------------------+


\* Variables that are persistent must be submitted during the first start when Gokapi creates a new config file. They can be omitted afterwards. Non-persistent variables need to be set on every start.



All values that are described in :ref:`cloudstorage` can be passed as environment variables as well. No values are persistent, therefore need to be set on every start.

+-----------------------+-------------------------+
| Name                  | Action                  |
+=======================+=========================+
| GOKAPI_AWS_BUCKET     | Sets the bucket name    |
+-----------------------+-------------------------+
| GOKAPI_AWS_REGION     | Sets the region name    |
+-----------------------+-------------------------+
| GOKAPI_AWS_KEY        | Sets the API key        |
+-----------------------+-------------------------+
| GOKAPI_AWS_KEY_SECRET | Sets the API key secret |
+-----------------------+-------------------------+
| GOKAPI_AWS_ENDPOINT   | Sets the endpoint       |
+-----------------------+-------------------------+


.. _api:


********************************
API
********************************

Gokapi offers an API that can be reached at ``http(s)://your.gokapi.url/api/``. You can find the current documentation with an overview of all API functions and examples at ``http(s)://your.gokapi.url/apidocumentation/``.


Interacting with the API
============================


All API calls will need an API key as authentication or a valid admin session cookie. An API key can be generated in the web UI in the menu "API". The API key needs to be passed as a header.

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



********************************
Customising
********************************

By default, all files are included in the executable. If you want to change the layout (e.g. add your company logo or change the app name etc.), follow these steps:

1. Download the source code for the Gokapi version you are using. It is either attached to the specific release  `on Github <https://github.com/Forceu/Gokapi/releases>`_ or you can clone the repository and checkout the tag for the specific version.
2. Copy either the folder ``static``, ``templates`` or both from the ``internal/webserver/web`` folder to the directory where the executable is located (if you are using Docker, mount the folders into the the ``/app/`` directory, e.g. ``/app/templates``).
3. Make changes to the folders. ``static`` contains images, CSS files and JavaScript. ``templates`` contains the HTML code.
4. Restart the server. If the folders exist, the server will use the local files instead of the embedded files.
5. Optional: To embed the files permanently, copy the modified files back to the original folders and recompile with ``go build Gokapi/cmd/gokapi``.

