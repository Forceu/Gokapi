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

General
--------


+----------------------+----------------------------------------------------------------------------------------------------------+-------------+-----------------------------------+-------------------------------+
| Name                 | Action                                                                                                   | Persistent* | Default                           | Required for unattended setup |
+----------------------+----------------------------------------------------------------------------------------------------------+-------------+-----------------------------------+-------------------------------+
| GOKAPI_CONFIG_DIR    | Sets the directory for the config file                                                                   | No          | config                            | No                            |
+----------------------+----------------------------------------------------------------------------------------------------------+-------------+-----------------------------------+-------------------------------+
| GOKAPI_CONFIG_FILE   | Sets the name of the config file                                                                         | No          | config.json                       | No                            |
+----------------------+----------------------------------------------------------------------------------------------------------+-------------+-----------------------------------+-------------------------------+
| GOKAPI_DATA_DIR      | Sets the directory for the data                                                                          | Yes         | data                              | No                            |
+----------------------+----------------------------------------------------------------------------------------------------------+-------------+-----------------------------------+-------------------------------+
| GOKAPI_USERNAME      | Sets the admin username                                                                                  | Yes         | unset                             | Yes                           |
+----------------------+----------------------------------------------------------------------------------------------------------+-------------+-----------------------------------+-------------------------------+
| GOKAPI_PASSWORD      | Sets the admin password                                                                                  | Yes         | unset                             | Yes                           |
+----------------------+----------------------------------------------------------------------------------------------------------+-------------+-----------------------------------+-------------------------------+
| GOKAPI_PORT          | Sets the server port                                                                                     | Yes         | 53842                             | Yes                           |
+----------------------+----------------------------------------------------------------------------------------------------------+-------------+-----------------------------------+-------------------------------+
| GOKAPI_EXTERNAL_URL  | Sets the external URL where Gokapi can be reached                                                        | Yes         | unset                             | Yes                           |
+----------------------+----------------------------------------------------------------------------------------------------------+-------------+-----------------------------------+-------------------------------+
| GOKAPI_REDIRECT_URL  | Sets the external URL where Gokapi will redirect to the index page is accesses                           | Yes         | unset                             | Yes                           |
+----------------------+----------------------------------------------------------------------------------------------------------+-------------+-----------------------------------+-------------------------------+
| GOKAPI_LOCALHOST     | Bind server to localhost. Expects true/false/yes/no, always false for Docker images                      | Yes         | false for Docker, otherwise unset | Yes                           |
+----------------------+----------------------------------------------------------------------------------------------------------+-------------+-----------------------------------+-------------------------------+
| GOKAPI_SALT_FILES    | Sets the salt for the file password hashes                                                               | Yes         | random salt                       | Yes                           |
+----------------------+----------------------------------------------------------------------------------------------------------+-------------+-----------------------------------+-------------------------------+
| GOKAPI_USE_SSL       | Serve all content through HTTPS and generate certificates. Expects true/false/yes/no                     | Yes         | unset                             | Yes                           |
+----------------------+----------------------------------------------------------------------------------------------------------+-------------+-----------------------------------+-------------------------------+
| GOKAPI_SALT_ADMIN    | Sets the salt for the admin password hash                                                                | Yes         | random salt                       | No                            |
+----------------------+----------------------------------------------------------------------------------------------------------+-------------+-----------------------------------+-------------------------------+
| GOKAPI_SALT_FILES    | Sets the salt for the file password hashes                                                               | Yes         | random salt                       | No                            |
+----------------------+----------------------------------------------------------------------------------------------------------+-------------+-----------------------------------+-------------------------------+
| GOKAPI_LENGTH_ID     | Sets the length of the download IDs. Value needs to be 5 or more                                         | Yes         | 15                                | No                            |
+----------------------+----------------------------------------------------------------------------------------------------------+-------------+-----------------------------------+-------------------------------+
| GOKAPI_MAX_FILESIZE  | Sets the maximum allowed file size in MB                                                                 | Yes         | 102400 (100GB)                    | No                            |
+----------------------+----------------------------------------------------------------------------------------------------------+-------------+-----------------------------------+-------------------------------+
| GOKAPI_DISABLE_LOGIN | Disables login for admin menu. DO NOT USE unless you have a 3rd party authentication for the ``/admin`` URL! | Yes         | false                             | No                            |
+----------------------+----------------------------------------------------------------------------------------------------------+-------------+-----------------------------------+-------------------------------+

\*Variables that are persistent must be submitted during the first start when Gokapi creates a new config file. They can be omitted afterwards. Non-persistent variables need to be set on every start.

Cloudstorage
-------------

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


External Authentication
------------------------

In order to use external authentication (eg. services like Authelia or Authentik), set the environment variable ``GOKAPI_DISABLE_LOGIN`` to ``true`` on the first start. *Warning:* This will diasable authentication for the admin menu, which can be dangerous if not set up correctly!

Refer to the documention of your reverse proxy on how to protect the ``/admin`` URL, as authentication is only required for this URL.

.. _api:

********************************
API
********************************

Gokapi offers an API that can be reached at ``http(s)://your.gokapi.url/api``. You can find the current documentation with an overview of all API functions and examples at ``http(s)://your.gokapi.url/apidocumentation/``.


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

1. Clone this repository
2. Copy either the folder ``static``, ``templates`` or both from the ``internal/webserver/web`` folder to the directory where the executable is located
3. Make changes to the folders. ``static`` contains images, CSS files and JavaScript. ``templates`` contains the HTML code.
4. Restart the server. If the folders exist, the server will use the local files instead of the embedded files
5. (Optional) To embed the files permanently, copy the modified files back to the original folders and recompiled with ``go build Gokapi/cmd/gokapi``.

