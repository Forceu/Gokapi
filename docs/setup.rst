.. _setup:

=====
Setup
=====

There are two different ways to set up Gokapi: either a native deployment approach or a containerised installation using Docker/Podman.

Also, there are two different versions: *Stable* indicates that you are using the latest release, which should work without any major bugs. *Unstable* is the latest developer version, which might include more features but could also contain bugs.


**************
Installation
**************


Docker
^^^^^^^

To download, run the following command:
::

  docker pull docker.io/f0rc3/gokapi:latest

If you want to install the unstable version, use ``latest-dev`` instead of ``latest``

If you don't want to download the prebuilt image, you can find the Dockerfile on the `Github project page <https://github.com/Forceu/gokapi>`_. 





Native Deployment
^^^^^^^^^^^^^^^^^^

Stable version
"""""""""""""""""
`Download the latest release <https://github.com/Forceu/gokapi/releases/latest>`_ and copy the executable into a new folder with write permissions. Select the executable according to your system. If you are using Windows, select ``gokapi-windows_amd64``, for Mac either ``gokapi-darwin_amd64`` or ``gokapi-darwin_arm64`` and for Linux the ``gokapi-linux_`` file matching your system.

Unstable version
"""""""""""""""""

Only recommended if you have experience with the command line. Go 1.20+ needs to be installed.

Create a new folder and in this folder execute 
::

 git clone https://github.com/Forceu/Gokapi.git .
 make

This will compile the source code and create an executable from the latest code.

**************
First Start
**************

After the first start you will be redirected to a setup webpage. If you require to change the port for the setup webserver, please set the ``GOKAPI_PORT`` env variable, see :ref:`envvar`


Starting Gokapi
^^^^^^^^^^^^^^^^


Docker
""""""""""

To start the container, run the following command: ::

 docker run -v gokapi-data:/app/data -v gokapi-config:/app/config -p 127.0.0.1:53842:53842 -e TZ=UTC f0rc3/gokapi:latest

With the argument ``-p 127.0.0.1:53842:53842`` the service will only be accessible from the machine it is running on. In most use-cases you will use a reverse proxy for SSL - if you want to make the service available to other computers in the network without a reverse proxy, replace the argument with ``-p 53842:53842``. Please note, unless you select SSL during the setup, the traffic will not be encrypted that way and data like passwords or transferred files can easily be read by third parties!

Set ``-e TZ=UTC`` to the timezone you are in, e.g. ``-e TZ=Europe/Berlin``.

If you do not want the binary to run as the root user in the container, you can set the environment variable ``DOCKER_NONROOT`` to true.

Please make sure that ``/app/data`` and ``/app/config`` are mounted as volumes (see example above), otherwise you will lose all your data after rebuilding or updating your container.


Native Deployment
""""""""""""""""""

To start Gokapi, execute the binary with your command line or by double clicking.



Initial Setup
^^^^^^^^^^^^^^^

During the first start, a new configuration file will be created and you will be asked for several inputs. With your webbrowser open ``http://localhost:53842/setup`` (or the appropriate URL) and follow the setup.



Database
""""""""""""""
By default, Gokapi stores its data in a database located in the ``data`` directory. You can specify a different database location in this menu. If no changes are needed, you can proceed as is. Redis is recommended for servers with a high frequency of downloads.

You can configure the following settings:

-  **Type of database** Choose either SQLite or Redis.
-  **Database location** Specify the path to the SQLite database.
-  **Database host** Provide the host and port number for the Redis database.
-  **Key prefix (optional)** This prefix will be added to all keys to prevent conflicts if the database is shared with other applications.
-  **Username (optional)** Enter the username for database connection.
-  **Password (optional)** Enter the password for database connection.
-  **Use SSL** Select this option to establish an SSL connection.

.. warning::
   The Redis password will be stored in plain text and can be viewed when re-running the setup.

Webserver
""""""""""""""

The following configuration can be set:

-  **Bind to localhost** Only allow the server to be accessed from the machine it is running on. Select this if you are running Gokapi behind a reverse proxy or for testing purposes
-  **Use SSL** Generates a self-signed SSL certificate (which can be replaced with a valid one). Select this if you are not running Gokapi behind a reverse proxy. Please note: Gokapi needs to be restarted in order to renew a certificate.
-  **Save IP** If set, the IP address of the client requesting a download will be saved to the log file. This might not be GDPR compliant.
-  **Include filename in download URL** If set, all Gokapi URLs for file downloads will include the filename as well. Example: ``https:/gokapi.server/d/1234/File.pdf`` instead of ``https:/gokapi.server/d?id=1234``
-  **Webserver Port** Set the port that Gokapi can be accessed on
-  **Public Facing URL** Enter the URL where users from an external network can use to reach Gokapi. The URL will be used for generating download links
-  **Redirection URL**  By default Gokapi redirects to this URL instead of showing a generic page if no download link was passed


.. note::
   If you choose to include the filename in the URL and later enable end-to-end encryption, the filename will appear in these URLs. However, since the filename is encrypted, it is only appended locally (client-side). This could pose a privacy concern in some situations. To address this, you can either disable the option or modify sensitive filenames in the URLs.
   
   Gokapi does not verify the filename in the URL, so you can change it to anything while the downloaded file will retain its original filename.



Authentication
""""""""""""""

This menu guides you through the authentication setup, where you select how an admin user logs in (only user that can upload files). It is possible to disable authentication completely, but strongly discouraged.


Username / Password 
*********************

The default authentication method. A single admin user will be generated that authenticates with a password


OAuth2 OpenID Connect
************************

Setup interface
========================

Use this to authenticate with an OIDC server, e.g. Google or an internal server like Authelia or Keycloak.

+--------------------+---------------------------------------------------------------------------------------------------+-----------------------------------------+
| Option             | Expected Entry                                                                                    | Example                                 |
+====================+===================================================================================================+=========================================+
| Provider URL       | The URL to connect to the OIDC server                                                             | https://accounts.google.com             |
+--------------------+---------------------------------------------------------------------------------------------------+-----------------------------------------+
| Client ID          | Client ID provided by the OIDC server                                                             | [random String]                         |
+--------------------+---------------------------------------------------------------------------------------------------+-----------------------------------------+
| Client Secret      | Client secret provided by the OIDC server                                                         | [random String]                         |
+--------------------+---------------------------------------------------------------------------------------------------+-----------------------------------------+
| Recheck identity   | How often to recheck identity.                                                                    | 12 hours                                |
|                    |                                                                                                   |                                         |
|                    | If the OIDC server is configured to remember the consent, the user should not receive any further |                                         |
|                    |                                                                                                   |                                         |
|                    | login prompts and it can be ensured, that the user still exist on the server.                     |                                         |
|                    |                                                                                                   |                                         |
|                    | Otherwise the user has actively grant access every time the identity is rechecked. In that case   |                                         |
|                    |                                                                                                   |                                         |
|                    | a higher interval would make sense.                                                               |                                         |
+--------------------+---------------------------------------------------------------------------------------------------+-----------------------------------------+
| Restrict to users  | Only allow authorised users to access Gokapi that are listed below                                | true                                    |
+--------------------+---------------------------------------------------------------------------------------------------+-----------------------------------------+
| Scope for users    | The OIDC scope that contains the user info                                                        | email                                   |
+--------------------+---------------------------------------------------------------------------------------------------+-----------------------------------------+
| Authorised users   | List of users that are authorised to log in as an admin, separated by semicolon.                  | \*\@company.com;admin\@othercompany.com |
|                    |                                                                                                   |                                         |
|                    | ``*`` can be used as a wildcard                                                                   |                                         |
+--------------------+---------------------------------------------------------------------------------------------------+-----------------------------------------+
| Restrict to groups | Only allow users that are part of authorised groups to access Gokapi                              | true                                    |
+--------------------+---------------------------------------------------------------------------------------------------+-----------------------------------------+
| Scope for groups   | The OIDC scope that contains the group info                                                       | groups                                  |
+--------------------+---------------------------------------------------------------------------------------------------+-----------------------------------------+
| Authorised groups  | List of groups that are authorised to log their users in as an admin, separated by semicolon.     | admin;dev;gokapi-\*                     |
|                    |                                                                                                   |                                         |
|                    | ``*`` can be used as a wildcard                                                                   |                                         |
+--------------------+---------------------------------------------------------------------------------------------------+-----------------------------------------+

.. note::
   If login is restricted to users and groups, both need to be present for a user to access. That means if a user has only one of the two factors, access to the admin menu will be denied.

.. note::
   A user will be authenticated until the time specified in ``Recheck identity`` has passed. To log out all users immediately, re-run the setup with `--reconfigure`` and complete it. Thereafter all active session will be deleted. 
   
   
.. note::
   If the OIDC provider is set up to remember consent, it might not be possible to log out through the Gokapi interface
   
   


OIDC client/server configuration
=======================================

When creating an OIDC client on the server, you will need to provide a **redirection URL**. Enter ``http[s]://[gokapi URL]/oauth-callback``

Tutorial for configuring OIDC servers and the correct client settings for Gokapi can be found in the :ref:`examples` page for the following servers:

* :ref:`oidcconfig_authelia`
* :ref:`oidcconfig_keycloak`
* :ref:`oidcconfig_google`
* :ref:`oidcconfig_entra`


Header Authentication
************************

Only use this if you are running Gokapi behind a reverse proxy that is capable of authenticating users, e.g. by using Authelia or Authentik. Keycloak does apparently not support this feature.

Enter the key of the header that returns the username. For Authelia this would be ``Remote-User`` and for Authentik ``X-authentik-username``.
Separate users with a semicolon or leave blank to allow any authenticated user, e.g. ``gokapiuser@gmail.com;companyadmin@gmail.com``


Disabled / Access Restriction
*************************************

Only use this if you are running Gokapi behind a reverse proxy that is capable of authenticating users, e.g. by using Authelia or Authentik.

This option disables Gokapis internal authentication completely, except for API calls. The following URLs need to be restricted by the reverse proxy:

- ``/admin``
- ``/apiKeys``
- ``/changePassword``
- ``/e2eInfo``
- ``/e2eSetup``
- ``/logs``
- ``/uploadChunk``
- ``/uploadStatus``
- ``/users``

.. warning::
   This option has potential to be *very* dangerous, only proceed if you know what you are doing!



Storage
""""""""""""""

Here you can choose where uploaded files shall be stored. Use the option to always store image files to the local storage, if you want to use encryption for cloudstorage, but require hotlink support. 

If using cloud storage, by default Gokapi creates a pre-signed download link for files to be downloaded (basically a URL that can only be used for a very short time). If your storage is not accessible from the internet or if you prefer to not expose any cloud storage URLs, you can choose to proxy the downloads. That way Gokapi downloads them and passes them to the user through the Gokapi service.

Local Storage
*********************

Stores files locally in the subdirectory ``data`` by default.


.. _cloudstorage:

Cloudstorage
*********************

.. note::
   Files will be stored in plain-text, if no encryption is selected later on in the setup

Stores files remotely on an S3 compatible server, e.g. Amazon AWS S3 or Backblaze B2.


It is highly recommended to create a new bucket for Gokapi and set it to "private", so that no file can be downloaded externally. For each download request Gokapi will create a public URL that is only valid for a couple of seconds, so that the file can be downloaded from the external server directly instead of routing it through the local server.

You then need to create an app key with read-/write-access to this bucket. If you are planning to use the encryption feature, make sure to set the bucket's CORS rules to allow access from the Gokapi URL.

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

Encryption
""""""""""""""

.. warning::
   Encryption has not been audited.

There are three different encryption levels, level 1 encrypts only local files and level 2 encrypts local and files stored on cloud storage (e.g. AWS S3). Decryption of files on remote storage is done client-side, for which a 2MB library needs to be downloaded on first visit. End-to-End encryption (level 3) encrypts the files client-side, therefore even if the Gokapi server has been compromised, no data should leak to the attacker. If the decryption is done client-side, the download on mobile devices may be significantly slower.

There are some drawbacks of using encryption:

+------------------------------+---------------+---------------------------------+---------------------------------+-------------------------+
|                              | No Encryption | Level 1 Local                   | Level 2 Full                    | Level 3 End-to-End      |
+==============================+===============+=================================+=================================+=========================+
| File Encryption              | None          | Only local files                | Local and cloud storage         | Local and cloud storage |
+------------------------------+---------------+---------------------------------+---------------------------------+-------------------------+
| Hotlink Support              | Yes           | Yes                             | Only local files                | No                      |
+------------------------------+---------------+---------------------------------+---------------------------------+-------------------------+
| Download Progress Indication | Yes           | Only cloud storage              | No                              | No                      |
+------------------------------+---------------+---------------------------------+---------------------------------+-------------------------+
| Download Speed               | Full          | Might be slower for local files | Slower for remote files,        | Slower for all files    |
|                              |               |                                 | might be slower for local files |                         |
+------------------------------+---------------+---------------------------------+---------------------------------+-------------------------+

You can choose to store the key in the configuration file, which is preferred if access by other parties to your configuration file is unlikely.

If you are concerned that the configuration file can be read, you can also choose to enter a master password on startup. This needs to be entered in the command line however and Gokapi will not be able to start without it.

.. note::
   If you re-run the setup and enable encryption, unencrypted files will stay unencrypted. If you change any configuration related to encryption, all already encrypted files will be deleted.

************************
Changing Configuration
************************

To change any settings set in the initial setup (e.g. your password or storage location), run Gokapi with the parameter ``--reconfigure`` and follow the instructions. A random username and password will be generated and displayed in the program output to access the configuration webpage, as all entered information can be read in plain text (except the user password).

If you are using Docker, shut down the running instance and create a new temporary container with the following command: ::

 docker run --rm -p 127.0.0.1:53842:53842 -v gokapi-data:/app/data -v gokapi-config:/app/config  f0rc3/gokapi:latest /app/run.sh --reconfigure
 
.. note::
   After completing the setup, all users will be logged out


.. note::
   If you are using Docker, make sure to stop the temporary container and to restart the original one after the setup is complete


**********************************
Reverse Proxy
**********************************

It is highly recommended to run Gokapi behind a reverse proxy. Make sure to select a high timeout (recommended: 300 seconds) and increase the allowed body size.

An example for Nginx can be found here: :ref:`nginx_config`


**********************************
Installing a systemd service
**********************************

.. warning::
   Only install Gokapi as a service *after* running it manually first and completing the setup steps under the `Initial Setup section <#initial-setup>`_.

.. note::
   This feature is currently only supported on UNIX-like systems that use systemd, for unsupported systems an error message will be shown. 

If you want to run Gokapi as a background service that starts on boot, you can use the following command:
::

  sudo ./gokapi --install-service

If you decide later to uninstall the service, you can use the following command:
::

  sudo ./gokapi --uninstall-service

By using either of these commands, all other command line flags will be ignored. Gokapi will try to determine the username of the user that invoked sudo, make sure that it is the correct user. It will not allow the service to be run as the root user.





