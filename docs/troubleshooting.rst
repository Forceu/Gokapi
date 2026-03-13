.. _troubleshooting:

===============
Troubleshooting
===============

This page covers the most common problems reported by users. If your issue is not listed here, check the :ref:`changelog` for breaking changes in recent versions and open a `GitHub issue <https://github.com/Forceu/Gokapi/issues>`_ if needed.

----

.. contents:: On this page
   :local:
   :depth: 1

----

End-to-end encrypted files cannot be downloaded on mobile
----------------------------------------------------------

**Symptom:** Downloads of end-to-end encrypted files fail or are very slow on mobile browsers (especially prominent on Firefox)

**Cause:** End-to-end decryption is performed entirely in the browser. Mobile devices have limited memory and processing power, which can cause failures for large files. Firefox does not offer streaming downloads, which can then results in truncated files.

**Fix:** This is a known limitation. For files that need to be reliably downloaded on mobile devices, consider using Level 1 or Level 2 encryption instead of end-to-end (Level 3). See :ref:`setup_encryption`.

Uploads fail or time out behind a reverse proxy
-------------------------------------------------

**Symptom:** Large file uploads fail partway through, or the browser shows a gateway error (502, 504).

**Cause:** The reverse proxy has a body size limit or a timeout that is shorter than the upload takes.

**Fix:**

* Increase the body size limit — Nginx: ``client_max_body_size 200M;`` (match or exceed your largest expected upload).
* Increase timeouts to at least 300 seconds — Nginx: ``proxy_read_timeout 300; proxy_send_timeout 300; proxy_connect_timeout 300;``.
* See the full example: :ref:`nginx_config`.


Gokapi always shows the wrong client IP / logs show 127.0.0.1
---------------------------------------------------------------

**Symptom:** All downloads are logged from ``127.0.0.1`` instead of real client IPs.

**Cause:** Gokapi does not trust the ``X-Forwarded-For`` header from your proxy unless the proxy's IP is in the trusted proxies list.

**Fix:** Set ``GOKAPI_TRUSTED_PROXIES`` to the IP (or CIDR range) of your proxy, e.g.:

.. code-block:: bash

   GOKAPI_TRUSTED_PROXIES=10.0.0.1

Multiple IPs and subnets are comma-separated: ``10.0.0.1,192.168.1.0/24``. See :ref:`availenvvar` for details.

If Gokapi is behind Cloudflare, set ``GOKAPI_USE_CLOUDFLARE=true`` instead.


Uploads fail when using Cloudflare (free plan)
-----------------------------------------------

**Symptom:** File uploads fail for files larger than ~100 MB when routed through Cloudflare.

**Cause:** Cloudflare's free plan has a 100 MB upload limit per request. Gokapi uploads files in chunks, so the chunk size must be kept at or below this limit.

**Fix:** Set ``GOKAPI_CHUNK_SIZE_MB=90`` (leaving a safety margin below 100 MB). See :ref:`availenvvar`.



All data is lost after restarting Redis
-----------------------------------------

**Symptom:** After restarting your Redis server, all Gokapi files and metadata are gone.

**Cause:** Redis does not persist data to disk by default. Without persistence, everything stored in memory is lost on restart.

**Fix:** Enable Redis persistence. The simplest approach is to add ``save 1 1`` to your ``redis.conf``. This tells Redis to save to disk after at least 1 key has changed within 1 second. For production use, consult the `Redis persistence documentation <https://redis.io/docs/manual/persistence/>`_ for a more appropriate strategy.

.. danger::
   Never use Redis without persistence enabled for Gokapi. Data loss is unrecoverable.


Setup page shows a CORS error
-------------------------------

**Symptom:** During setup, a CORS-related error appears when testing the configuration.

**Cause:** The public-facing URL entered in setup does not match the URL you are actually accessing Gokapi from, or a reverse proxy is modifying the ``Origin`` header.

**Fix:**

* Make sure the *Public Facing URL* field matches the URL in your browser exactly (including ``http://`` vs ``https://`` and port number).
* If you need to bypass the CORS check for testing, set ``GOKAPI_DISABLE_CORS_CHECK=true`` temporarily. Do not leave this enabled in production.



File replacement is not available
-----------------------------------

**Symptom:** The option to replace a file's content is missing from the UI or returns an error via the API.

**Cause:** File replacement has been disabled via the ``GOKAPI_DISABLE_REPLACE`` environment variable.

**Fix:** Remove or set ``GOKAPI_DISABLE_REPLACE=false``. See :ref:`availenvvar`.


Cannot upload files — "not enough free space" error
-----------------------------------------------------

**Symptom:** Uploads are rejected with a message about insufficient disk space, even when disk space appears available.

**Cause:** Gokapi checks that at least ``GOKAPI_MIN_FREE_SPACE`` MB (default: 400 MB) is available before accepting an upload.

**Fix:** Free up disk space, or if you intentionally want to allow uploads with less headroom, lower the threshold:

.. code-block:: bash

   GOKAPI_MIN_FREE_SPACE=100

See :ref:`availenvvar`.


Docker container loses data after update
-----------------------------------------

**Symptom:** After pulling a new image and restarting the container, files and settings are gone.

**Cause:** The container was started without volume mounts, so data was stored inside the container's writable layer, which is discarded when the container is removed.

**Fix:** Always mount ``/app/data`` and ``/app/config`` as named volumes or bind mounts:

.. code-block:: bash

   docker run -v gokapi-data:/app/data -v gokapi-config:/app/config ...

See :ref:`quickstart_docker` for the full run command.

