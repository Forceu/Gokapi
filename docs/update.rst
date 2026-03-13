.. _update:

======================
Updating Gokapi
======================

.. note::
   Before updating, always check the :ref:`changelog` for breaking changes in the version you are upgrading to.
   Some releases require manual steps before or after the update.

The database schema is migrated automatically on the first start after an upgrade.
It is not possible to downgrade to an older version after a schema migration has run.

.. warning::
   **Back up your data before updating**, especially for major version upgrades.
   Copy the ``data`` and ``config`` directories (or their Docker volumes) to a safe location.


***************
Docker
***************

Pull the new image:

.. code-block:: bash

   docker pull f0rc3/gokapi:latest

Then stop the running container and start it again with the same command you used originally.
Named volumes (``-v gokapi-data:/app/data``) preserve all your data automatically.

If you use Docker Compose:

.. code-block:: bash

   docker compose pull
   docker compose up -d


*******************
Native deployment
*******************

Stable version
==============

Download the latest release and extract it into the same directory as your existing installation, overwriting the old binary. Then restart Gokapi.

Unstable version
=================

.. code-block:: bash

   git pull
   make

Then restart Gokapi.
