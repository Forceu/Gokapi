.. _setup:

=====
Setup
=====

There are two different ways to setup Gokapi: Either a bare metal approach or docker.

Also there are two different versions: *Stable* indicates that you are using the latest release which should work without any bugs. *Unstable* is the latest developer version, which might include more features, but could also contain bugs.


**************
Installation
**************

Bare Metal
^^^^^^^^^^^^

Stable version
"""""""""""""""""
`Download the project <https://github.com/Forceu/gokapi/releases/latest>`_ and copy the executable into a new folder with write permissions.

Unstable version
"""""""""""""""""

Only recommended if you have expierence with the command line. Go 1.16+ needs to be installed.

Create a new folder and in this folder execute 
::

 git clone https://github.com/Forceu/Gokapi.git .
 go generate ./...
 go build Gokapi/cmd/gokapi

This will compile the source code and create an executable from the latest code.


Docker
^^^^^^^

To download, run the following command, and replace YOURTAG with either *latest* (stable) or *latest-dev* (unstable).
::

  docker pull f0rc3/barcodebuddy-docker:YOURTAG

Most of the time, you will need the *latest* tag. 

If you don't want to download the prebuilt image, you can find the Dockerfile on the `Github project page <https://github.com/Forceu/gokapi>`_. 



**************
First Start
**************

After the first start you will be redirected to a setup webpage. To change the port for the setup please set the GOKAPI_PORT env variable, see :ref:`envvar`


Starting Gokapi
^^^^^^^^^^^^^^^^

Bare Metal
""""""""""

To start Gokapi, execute the binary with your command line or by double clicking.


Docker
""""""""""

To start the container, run the following command: ::

 docker run -v gokapi-data:/app/data -v gokapi-config:/app/config -p 127.0.0.1:53842:53842 f0rc3/gokapi:latest

With the argument ``-p 127.0.0.1:53842:53842`` the service will only be accessible from the machine it is running on. In most usecases you will use a reverse proxy for SSL - if you want to make the service available to other computers in the network without a reverse proxy, replace the argument with ``-p 53842:53842``. Please note, unless you select SSL during the setup, the traffic will not be encrypted that way and data like passwords and transferred files can easily be read by 3rd parties!


Initial Setup
^^^^^^^^^^^^^^^

During the first start, a new configuration file will be created and you will be asked for several inputs.



Webserver
""""""""""""""

The following configuration can be set:

-  **Bind to localhost** Only allow the server to be accessed from the machine it is running on. Select this if you are running Gokapi behind a reverse proxy or for testing purposes
-  **Use SSL** Generates a self-signed SSL certificate (which can be replaced with a valid one). Select this if you are not running Gokapi behind a reverse proxy
-  **Webserver Port** Set the port that Gokapi can be accessed on
-  **Public Facing URL** Enter the URL where users from an external network can use to reach Gokapi. The URL will be used for generating download links
-  **Redirection URL**  By default Gokapi redirects to this URL instead of showing a generic page if no download link was passed


Authentication
""""""""""""""

This menu guides you through the authentication setup, where you select how an admin user logs in (only user that can upload files)


Username / Password 
*********************

The default authentication method. A single admin user will be generated that authenticates with a password


OAuth2 OpenID Connect
************************

Use this to authenticate with an OIDC server, eg. Google, Github or an internal server. *Note:* If a user is revoked on the OIDC server, it might take several days to affect the Gokapi session. 

+---------------+---------------------------------------------------------------------------------+---------------------------------------------+
| Option        | Expected Entry                                                                  | Example                                     |
+===============+=================================================================================+=============================================+
| Provider URL  | The URL to connect to the OIDC server                                           | https://accounts.google.com                 |
+---------------+---------------------------------------------------------------------------------+---------------------------------------------+
| Client ID     | Client ID provided by the OIDC server                                           | [random String]                             |
+---------------+---------------------------------------------------------------------------------+---------------------------------------------+
| Client Secret | Client secret provided by the OIDC server                                       | [random String]                             |
+---------------+---------------------------------------------------------------------------------+---------------------------------------------+
| Allowed users | List of users that is allowed to log in as an admin.                            | gokapiuser@gmail.com;companyadmin@gmail.com |
|               | Separate users with a semicolon or leave blank to allow any authenticated user  |                                             |
+---------------+---------------------------------------------------------------------------------+---------------------------------------------+

When creating an OIDC client on the server, you will need to provide a **redirection URL**. Enter http(s)://[gokapi URL]/oauth-callback

You can find a guide on how to create an OIDC client with Github at `Setting up GitHub OAuth 2.0 <https://docs.readme.com/docs/setting-up-github-oauth>`_ and a guide for Google at `Setting up OAuth 2.0 <https://support.google.com/cloud/answer/6158849>`_.


Header Authentication
************************

Only use this if you are running Gokapi behind a reverse proxy that is capable of authenticating users, e.g. by using Authelia or Authentik.

Enter the key of the header that returns the username. For Authelia this would be ``Remote-User`` and for Authentik `` X-authentik-username``.
Separate users with a semicolon or leave blank to allow any authenticated user, e.g. ``gokapiuser@gmail.com;companyadmin@gmail.com``


Access Restriction
************************

Only use this if you are running Gokapi behind a reverse proxy that is capable of authenticating users, e.g. by using Authelia or Authentik.

This option disables Gokapis internal authentication completely, except for API calls. The following URLs need to be restricted by the reverse proxy:

- ``/admin``
- ``/apiDelete``
- ``/apiKeys``
- ``/apiNew``
- ``/delete``
- ``/upload``

**Warning:** This option has potential to be dangerous, only proceed if you know what you are doing!



Storage
""""""""""""""

Here you can choose where uploaded files shall be stored

Local Storage
*********************

Stores files locally in the subdirectory ``data`` by default.


.. _cloudstorage:

Cloudstorage
""""""""""""""

Stores files remotely on an S3 compatible server, e.g. Amazon AWS S3 or Backblaze B2. Please note that currently no native encryption is available for Gokapi, therefore all files will be stored in plain text on the cloud server.


It is highly recommended to create a new bucket for Gokapi and set it to "private", so that no file can be downloaded externally. For each download request Gokapi will create a public URL that is only valid for a couple of seconds, so that the file can be downloaded from the external server directly instead of routing it through the local server.

You then need to create an app key with read-/write-access to this bucket.

The following data needs to be provided:


+-----------+-----------------------------------------------+-----------------------+-----------------------------------+
| Key       | Description                                   | Required              | Example                           |
+===========+===============================================+=======================+===================================+
| Bucket    | Name of the bucket in use                     | yes                   | gokapi                            |
+-----------+-----------------------------------------------+-----------------------+-----------------------------------+
| Region    | Name of the region                            | yes                   | eu-central-1                      |
+-----------+-----------------------------------------------+-----------------------+-----------------------------------+
| KeyId     | Name of the API key                           | yes                   | keyname123456789                  |
+-----------+-----------------------------------------------+-----------------------+-----------------------------------+
| KeySecret | Value of the API key secret                   | yes                   | verysecret123                     |
+-----------+-----------------------------------------------+-----------------------+-----------------------------------+
| Endpoint  | Endpoint to use. Leave blank if using AWS S3. | only for Backblaze B2 | s3.eu-central-001.backblazeb2.com |
+-----------+-----------------------------------------------+-----------------------+-----------------------------------+


************************
Changing Configuration
************************

To change any settings set in the initial setup (e.g. your password or storage location), run Gokapi with the parameter ``--reconfigure`` and follow the instructions. A random username and password will be generated and displayed in the programm output to access the configuration webpage, as all entered information can be read in plain text (except the user password).

