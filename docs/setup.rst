.. _setup:

=====
Setup
=====

.. note::
   **Most users:** pull the Docker image, run the container, open the setup URL in your browser.
   Jump straight to :ref:`quickstart_docker` if that describes you.
   The sections below cover every option in full detail.


*****************************
Installation
*****************************

There are two deployment methods (Docker and native) and two release tracks (stable and unstable).
**Stable + Docker is the recommended combination for most users.**

Docker
^^^^^^^

Pull the latest stable image:

.. code-block:: bash

   docker pull docker.io/f0rc3/gokapi:latest

For the unstable (development) build, use ``latest-dev`` instead of ``latest``.

If you prefer to build the image yourself, the Dockerfile is available on the `GitHub project page <https://github.com/Forceu/gokapi>`_.


Native Deployment
^^^^^^^^^^^^^^^^^^

Stable version
""""""""""""""

`Download the latest release <https://github.com/Forceu/gokapi/releases/latest>`_ and copy the executable into a new folder with write permissions.

Choose the binary that matches your system:

* **Windows:** ``gokapi-windows_amd64``
* **Mac (Intel):** ``gokapi-darwin_amd64``
* **Mac (Apple Silicon):** ``gokapi-darwin_arm64``
* **Linux:** ``gokapi-linux_<arch>`` matching your architecture

.. note::
   ``gokapi-XXX`` is the main server application. ``gokapi-cli-XXX`` is the optional command-line upload/download tool — you do not need it to run the server.

Unstable version
""""""""""""""""

Only recommended if you are comfortable with the command line. Requires Go 1.25 or newer.

.. code-block:: bash

   git clone https://github.com/Forceu/Gokapi.git .
   make

This compiles the source and produces a ``gokapi`` executable from the latest code.


*****************************
Running Gokapi
*****************************

.. _quickstart_docker:

Docker
^^^^^^^

.. warning::
   Always mount ``/app/data`` and ``/app/config`` as volumes. Without them, **all data is lost** when the container is removed or updated.

The standard run command:

.. code-block:: bash

   docker run -v gokapi-data:/app/data -v gokapi-config:/app/config \
     -p 127.0.0.1:53842:53842 -e TZ=UTC f0rc3/gokapi:latest

* ``-p 127.0.0.1:53842:53842`` — binds to localhost only, which is correct when using a reverse proxy.
  Replace with ``-p 53842:53842`` only if you need direct access without a proxy — note that without SSL, traffic is unencrypted.
* ``-e TZ=UTC`` — set this to your timezone, e.g. ``-e TZ=Europe/Berlin``.

Running as a non-root user:

.. code-block:: bash

   docker run --user "1000:1000" \
     -v ./gokapi-data:/app/data -v ./gokapi-config:/app/config \
     -p 127.0.0.1:53842:53842 -e TZ=UTC f0rc3/gokapi:latest

Replace ``1000:1000`` with the desired ``uid:gid``. Note that this form uses bind mounts (``./gokapi-data``) instead of named volumes — make sure the host directories exist and the user has read/write access.

See :ref:`deprecation_nonroot` if you are migrating from the old ``DOCKER_NONROOT`` environment variable.

Docker Compose
""""""""""""""

Download ``docker-compose.yaml`` and ``.env.dist`` from the repository. Rename ``.env.dist`` to ``.env`` and edit as needed.

The ``gokapi-data`` and ``gokapi-config`` folders are created automatically in the current directory. Start with:

.. code-block:: bash

   docker compose up -d

By default the container restarts automatically on boot (``restart: always``). Change to ``restart: unless-stopped`` if you only want automatic restart after a crash.

Native Deployment
"""""""""""""""""

Execute the binary from the command line or by double-clicking it.


*****************************
Initial Setup
*****************************

On the first start, Gokapi creates a configuration file and opens a setup wizard. Open ``http://localhost:53842/setup`` in your browser (adjust the host and port as needed) and work through the steps below.

If you need to change the port before running setup, set ``GOKAPI_PORT`` — see :ref:`envvar`.

.. _setup_database:

Database
^^^^^^^^^

.. warning::
   If you choose Redis, **you must enable Redis persistence** before storing any data (e.g. add ``save 1 1`` to your ``redis.conf``). Without persistence, all data is lost on a Redis restart.

.. warning::
   The Redis password is stored in plain text in the configuration file and will be visible if you re-run setup.

By default Gokapi uses SQLite, which is fine for most deployments. Use Redis if:

* you expect high download/upload traffic, or
* your SQLite database lives on a slow disk (e.g. a network share or SD card).

Settings:

* **Type of database** — SQLite or Redis.
* **Database location** — path to the SQLite file.
* **Database host** — host and port for Redis (e.g. ``127.0.0.1:6379``).
* **Key prefix** *(optional)* — added to all Redis keys; useful when sharing a Redis instance with other applications.
* **Username / Password** *(optional)* — Redis authentication credentials.
* **Use SSL** — enables TLS for the Redis connection.

.. _setup_webserver:

Webserver
^^^^^^^^^

* **Bind to localhost** — only accept connections from the local machine. Enable this when running behind a reverse proxy.
* **Use SSL** — generates a self-signed certificate. Use this only if you are *not* behind a reverse proxy. Gokapi must be restarted to renew the certificate.
* **Save IP** — logs the downloader's IP address. This may not be GDPR-compliant depending on your jurisdiction.
* **Include filename in download URL** — appends the filename to download links, e.g. ``/d/1234/Report.pdf`` instead of ``/d?id=1234``.
* **Public Name** — shown in the page title; use your company or service name.
* **Webserver Port** — the port Gokapi listens on.
* **Public Facing URL** — the externally reachable URL used when generating download links.
* **Redirection URL** — where users are sent when they open the root URL without a valid download link.

.. note::
   If you enable filename-in-URL together with end-to-end encryption, the filename is appended client-side after decryption and is therefore visible in the URL. This may be a privacy concern for sensitive filenames. You can disable the option, or simply edit the filename in the URL — Gokapi ignores it and always serves the correct file.

.. _setup_authentication:

Authentication
^^^^^^^^^^^^^^

Choose how users log in. Disabling authentication entirely is strongly discouraged.

Username / Password
""""""""""""""""""""

The default method. All users authenticate with a username and password. You will be asked to create the initial admin credentials in the next step.

OAuth2 / OpenID Connect
""""""""""""""""""""""""""

.. note::
   Users must have an email address associated with their OIDC account.

Use this to delegate authentication to an OIDC server such as Google, Authelia, or Keycloak.

+---------------------+---------------------------------------------------------------------------------------------------+-----------------------------------------+
| Option              | Expected Entry                                                                                    | Example                                 |
+=====================+===================================================================================================+=========================================+
| Provider URL        | The URL of the OIDC server                                                                        | https://accounts.google.com             |
+---------------------+---------------------------------------------------------------------------------------------------+-----------------------------------------+
| Client ID           | Client ID from the OIDC server                                                                    | [random string]                         |
+---------------------+---------------------------------------------------------------------------------------------------+-----------------------------------------+
| Client Secret       | Client secret from the OIDC server                                                                | [random string]                         |
+---------------------+---------------------------------------------------------------------------------------------------+-----------------------------------------+
| Admin email         | Email address that identifies the super-admin account                                             | gokapi@company.com                      |
+---------------------+---------------------------------------------------------------------------------------------------+-----------------------------------------+
| Recheck identity    | How often to re-verify the user's identity with the OIDC server.                                  | 12 hours                                |
|                     |                                                                                                   |                                         |
|                     | If the server remembers consent, users will not see a login prompt again.                         |                                         |
|                     |                                                                                                   |                                         |
|                     | If it does not, each recheck triggers an active login prompt — use a longer interval.             |                                         |
+---------------------+---------------------------------------------------------------------------------------------------+-----------------------------------------+
| Restrict to groups  | Only allow members of specific groups                                                             | true                                    |
+---------------------+---------------------------------------------------------------------------------------------------+-----------------------------------------+
| Scope for groups    | The OIDC scope that carries group membership                                                      | groups                                  |
+---------------------+---------------------------------------------------------------------------------------------------+-----------------------------------------+
| Only existing users | Do not create a new Gokapi account automatically on first OIDC login                              | checked                                 |
+---------------------+---------------------------------------------------------------------------------------------------+-----------------------------------------+
| Authorised groups   | Semicolon-separated list of groups. ``*`` is a wildcard.                                          | admin;dev;gokapi-\*                     |
+---------------------+---------------------------------------------------------------------------------------------------+-----------------------------------------+

.. note::
   If a user is disabled in the OIDC provider, they can still log in to Gokapi until the *Recheck identity* interval expires.
   To revoke access immediately, delete the user's Gokapi account from the Users page.

.. note::
   If the OIDC provider remembers consent, logging out through the Gokapi interface may not fully log the user out.

When registering Gokapi as a client in your OIDC server, set the **redirect URL** to:
``http[s]://[your-gokapi-url]/oauth-callback``

Step-by-step configuration guides for specific providers are in the :ref:`examples` section:

* :ref:`oidcconfig_authelia`
* :ref:`oidcconfig_keycloak`
* :ref:`oidcconfig_google`
* :ref:`oidcconfig_entra`

Header Authentication
""""""""""""""""""""""

Use this only if your reverse proxy handles authentication (e.g. Authelia or Authentik) and forwards the authenticated username as a request header. Keycloak does not support this mode.

Enter the header key that contains the username (e.g. ``Remote-User`` for Authelia, ``X-authentik-username`` for Authentik) and the username of the admin account.

If *Only allow already existing users to log in* is enabled, new usernames coming from the proxy will be rejected until an account is created through the UI.

Disabled / Access Restriction
"""""""""""""""""""""""""""""""

.. warning::
   This option is **very dangerous**. Only use it if you fully understand the implications and your reverse proxy is correctly configured.

This disables all of Gokapi's internal authentication except for API calls. The following paths **must** be protected by your reverse proxy:

- ``/admin``
- ``/apiKeys``
- ``/auth/token``
- ``/changePassword``
- ``/downloadPresigned``
- ``/e2eSetup``
- ``/filerequests``
- ``/logs``
- ``/uploadChunk``
- ``/uploadStatus``
- ``/users``

.. _setup_storage:

Storage
^^^^^^^^^^

Choose where uploaded files are stored.

Local Storage
"""""""""""""""""

Files are stored in the ``data`` subdirectory by default.

.. _cloudstorage:

Cloud Storage (S3-compatible)
""""""""""""""""""""""""""""""""""

.. note::
   Without encryption enabled later in setup, files are stored in plain text on the cloud provider.

Stores files on an S3-compatible service such as Amazon S3 or Backblaze B2.

Create a dedicated private bucket for Gokapi. For each download, Gokapi generates a short-lived pre-signed URL so the user downloads directly from the storage provider without routing traffic through your server. If the storage is not publicly reachable, or if you prefer not to expose storage URLs, enable the proxy download option — Gokapi will retrieve the file and stream it to the user itself.

Required fields:

+-----------+-----------------------------------------------+-----------------------+-----------------------------------+
| Field     | Description                                   | Required              | Example                           |
+===========+===============================================+=======================+===================================+
| Bucket    | Name of the bucket                            | Yes                   | gokapi                            |
+-----------+-----------------------------------------------+-----------------------+-----------------------------------+
| Region    | Region name                                   | Yes                   | eu-central-1                      |
+-----------+-----------------------------------------------+-----------------------+-----------------------------------+
| KeyId     | API key ID                                    | Yes                   | keyname123456789                  |
+-----------+-----------------------------------------------+-----------------------+-----------------------------------+
| KeySecret | API key secret                                | Yes                   | verysecret123                     |
+-----------+-----------------------------------------------+-----------------------+-----------------------------------+
| Endpoint  | Custom endpoint. Leave blank for AWS S3.      | Only for Backblaze B2 | s3.eu-central-001.backblazeb2.com |
+-----------+-----------------------------------------------+-----------------------+-----------------------------------+

If you plan to use end-to-end encryption with cloud storage, configure your bucket's CORS rules to allow requests from your Gokapi URL.

.. _setup_encryption:

Encryption
^^^^^^^^^^^

.. warning::
   The encryption implementation has not been independently audited. Evaluate this risk before using it for sensitive data.

Three levels are available:

+------------------------------+---------------+---------------------------------+---------------------------------+-------------------------+
|                              | No Encryption | Level 1 — Local only            | Level 2 — Full                  | Level 3 — End-to-End    |
+==============================+===============+=================================+=================================+=========================+
| File Encryption              | None          | Local files only                | Local and cloud storage         | Local and cloud storage |
+------------------------------+---------------+---------------------------------+---------------------------------+-------------------------+
| Hotlink Support              | Yes           | Yes                             | Local files only                | No                      |
+------------------------------+---------------+---------------------------------+---------------------------------+-------------------------+
| Download Progress Indication | Yes           | Cloud storage only              | No                              | No                      |
+------------------------------+---------------+---------------------------------+---------------------------------+-------------------------+
| Download Speed               | Full          | May be slower for local files   | Slower for remote files         | Slower for all files    |
+------------------------------+---------------+---------------------------------+---------------------------------+-------------------------+

* **Level 1** encrypts files stored locally; cloud files are unencrypted.
* **Level 2** encrypts both local and cloud files. Cloud file decryption is done client-side; a 2 MB library is downloaded on first visit.
* **Level 3 (end-to-end)** encrypts files in the browser before upload. Even a fully compromised server cannot read the file contents. Downloads on mobile devices may be noticeably slower.

The encryption key can be stored in the configuration file (convenient, lower security) or entered as a master password at each startup (more secure, requires manual input).

.. note::
   If you re-run setup and enable encryption, existing unencrypted files remain unencrypted. Changing any encryption setting deletes all already-encrypted files.

.. warning::
   Firefox is currently not completly compatible with end-to-end encryption, which may result in truncated files when downloading end-to-end encrypted files with a Firefox browser


*****************************
Changing Configuration
*****************************

To change settings after the initial setup (e.g. password, storage backend, authentication method), run Gokapi with ``--reconfigure``. A temporary random username and password are printed to the console for accessing the configuration page.

Native:

.. code-block:: bash

   ./gokapi --reconfigure

Docker:

.. code-block:: bash

   docker run --rm -p 127.0.0.1:53842:53842 \
     -v gokapi-data:/app/data -v gokapi-config:/app/config \
     f0rc3/gokapi:latest /app/run.sh --reconfigure

.. note::
   Stop the ``--reconfigure`` container and restart your normal container after completing the setup. All users will be logged out.


*****************************
Reverse Proxy
*****************************

Running Gokapi behind a reverse proxy is strongly recommended for production deployments. The proxy handles SSL termination and exposes Gokapi on standard ports.

Required proxy settings:

* **Timeout** — at least 300 seconds (to accommodate large file uploads).
* **Maximum request body size** — set high enough for your largest expected upload.
* **Forwarded headers** — pass ``X-Real-IP`` and ``X-Forwarded-For`` so Gokapi sees the real client IP.

If your proxy's outgoing IP is not ``127.0.0.1``, add it to ``GOKAPI_TRUSTED_PROXIES`` — see :ref:`availenvvar`.

If Gokapi is behind Cloudflare, set ``GOKAPI_USE_CLOUDFLARE=true``.

.. note::
   Cloudflare's free plan limits upload chunks to 100 MB. If you use Cloudflare, keep ``GOKAPI_CHUNK_SIZE_MB`` at or below 100.

A complete Nginx configuration example is in the :ref:`examples` section: :ref:`nginx_config`.

For other reverse proxies (Caddy, Traefik, Apache), the same principles apply: increase timeouts, increase body size limits, and forward the real client IP. Consult your proxy's documentation for the specific directives.


*********************************
Installing as a systemd Service
*********************************

.. warning::
   Complete the initial setup by running Gokapi manually at least once before installing it as a service.

.. note::
   Only supported on Linux systems using systemd. An error is shown on unsupported systems.

.. code-block:: bash

   sudo ./gokapi --install-service

To uninstall:

.. code-block:: bash

   sudo ./gokapi --uninstall-service

Gokapi detects the user who invoked ``sudo`` and runs the service as that user. Running as root is not permitted.


.. _deprecation_nonroot:

********************************************************
Migration from DOCKER_NONROOT to docker --user
********************************************************

The ``DOCKER_NONROOT`` environment variable is deprecated. To migrate to ``docker --user``:

.. code-block:: bash

   # Copy data out of the container (directories must not exist yet)
   docker cp gokapi:/app/config ./gokapi-config
   docker cp gokapi:/app/data ./gokapi-data

   # Remove the old container
   docker rm -f gokapi

   # Start a new container as the current user
   docker run --user "$(id -u):$(id -g)" \
     -v ./gokapi-data:/app/data -v ./gokapi-config:/app/config \
     -p 127.0.0.1:53842:53842 -e TZ=UTC f0rc3/gokapi:latest

Replace ``gokapi`` with your actual container name if different. Replace ``$(id -u):$(id -g)`` with explicit IDs if you want a specific user. Ensure the bind-mount directories are owned by that user.
