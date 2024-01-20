.. _examples:


===========================
Examples
===========================



*********************************
Nginx  Configuration
*********************************


.. code-block:: nginx

	server {
		listen 80;
		listen [::]:80;
		listen 443 ssl;
		listen [::]:443 ssl;
		ssl_certificate /your/certificate/fullchain.pem;
		ssl_certificate_key /your/certificate/privkey.pem;

		client_max_body_size 500M;
		client_body_buffer_size 128k;

		server_name your.server.url;

		location / {
			# If using Cloudflare
			proxy_set_header X-Forwarded-Host $http_cf_connecting_ip;
			
			proxy_set_header Host $http_host;
			proxy_set_header X-Real-IP $remote_addr;
			proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
			proxy_set_header X-Forwarded-Proto http;
			proxy_pass http://127.0.0.1:53842;
		}



		# Always redirect to https
		if ( $scheme = http ) {
			return 301 https://$server_name$request_uri;
		}
	}




*********************************
OpenID Connect  Configuration
*********************************


.. _oidcconfig_authelia:

Authelia
^^^^^^^^^^^^

Server Configuration
""""""""""""""""""""""

.. note::
   This guide has been written for version 4.37.5

See the `Authelia documentation <https://www.authelia.com/configuration/identity-providers/open-id-connect/>`_ on how to setup an OIDC server. An example file would be as followed:


.. code-block:: YAML

	identity_providers:
	  oidc:
	    hmac_secret: noz1Aow6Soo9lieyus2E_EXAMPLE_KEY
	    issuer_private_key: |
	      -----BEGIN PRIVATE KEY-----
	      ohf2shae1bahph7ahSh1
	      EXAMPLE_KEY
	      EP3EihoPhei9iingai0v==
	      -----END PRIVATE KEY-----
	    access_token_lifespan: 1h
	    authorize_code_lifespan: 1m
	    id_token_lifespan: 1h
	    refresh_token_lifespan: 90m
	    enable_client_debug_messages: false
	    enforce_pkce: public_clients_only
	    cors:
	      endpoints:
		- authorization
		- token
		- revocation
		- introspection
	      allowed_origins:
		- "https://*.your.domain"
	      allowed_origins_from_client_redirect_uris: false
	    clients:
	      - id: gokapi-dev
		description: Gokapi Example
		secret: 'AhXeV7_EXAMPLE_KEY'
		sector_identifier: ''
		public: false
		authorization_policy: one_factor
		consent_mode: pre-configured
		pre_configured_consent_duration: 1w
		audience: []
		scopes:
		  - openid
		  - email
		  - profile
		  - groups
		redirect_uris:
		  - https://gokapi.website.com/oauth-callback
		userinfo_signing_algorithm: none


* Set ``authorization_policy`` to ``two_factor`` to use OTP or a hardware key.
* If ``consent_mode`` is ``pre-configured``, the user has the option to remember consent. That way you can use a lower ``Recheck identity`` interval in Gokapi. Logout through the Gokapi interface will not be possible anymore, unless the user logs out their Authelia account. If the option is set to  ``explicit``, the user always has to grant the permission aftter the ``Recheck identity`` interval has passed
* ``scopes`` may exclude ``email`` and ``groups`` if these are not required for authentication, e.g. if all users registered with Authelia may access Gokapi.
* Make sure ``redirect_uris`` is set to the correct value


Gokapi Configuration
""""""""""""""""""""""

+--------------------------+-----------------------------------------------------------+-----------------------------------------+
| Gokapi Configuration     | Input                                                     | Example                                 |
+==========================+===========================================================+=========================================+
| Provider URL             | URL to Authelia Server                                    | \https://auth.autheliaserver.com        |
+--------------------------+-----------------------------------------------------------+-----------------------------------------+
| Client ID                | Client ID provided in config                              | gokapi-dev                              |
+--------------------------+-----------------------------------------------------------+-----------------------------------------+
| Client Secret            | Client secret provided in config                          | AhXeV7_EXAMPLE_KEY                      |
+--------------------------+-----------------------------------------------------------+-----------------------------------------+
| Recheck identity         | If mode is ``pre-configured``, use a low interval.        | 12 hours                                |
+--------------------------+-----------------------------------------------------------+-----------------------------------------+
| Restrict to user         | Check this if only certain users shall be allowed to      | checked                                 |
|                          |                                                           |                                         |
|                          | access Gokapi admin menu                                  |                                         |
+--------------------------+-----------------------------------------------------------+-----------------------------------------+
| Scope identifier (user)  | Use a scope that is unique to the user, e.g. the username | email                                   |
|                          |                                                           |                                         |
|                          | or the email                                              |                                         |
+--------------------------+-----------------------------------------------------------+-----------------------------------------+
| Authorised users         | Enter all users, separated by semicolon                   | \*\@company.com;admin\@othercompany.com |
|                          |                                                           |                                         |
|                          | ``*`` can be used as a wildcard                           |                                         |
+--------------------------+-----------------------------------------------------------+-----------------------------------------+
| Restrict to group        | Check this if only users from certain groups shall be     | checked                                 |
|                          |                                                           |                                         |
|                          | allowed to access Gokapi admin menu                       |                                         |
+--------------------------+-----------------------------------------------------------+-----------------------------------------+
| Scope identifier (group) | Use a scope that lists the user's groups                  | groups                                  |
+--------------------------+-----------------------------------------------------------+-----------------------------------------+
| Authorised groups        | Enter all groups, separated by semicolon                  | dev;admins;gokapi-*                     |
|                          |                                                           |                                         |
|                          | ``*`` can be used as a wildcard                           |                                         |
+--------------------------+-----------------------------------------------------------+-----------------------------------------+


.. _oidcconfig_keycloak:

Keycloak
^^^^^^^^^^^^

.. note::
   This guide has been written for version 23.0.4


Server Configuration
""""""""""""""""""""""


Creating the client
**********************

#. In your realm (default: master) click on ``[Manage] Clients`` and then ``Create Client``

    * Client Type: OpenID Connect
    * Client ID: a unique ID, ``gokapi-dev`` is used in this example
    
#. Click ``Next``

    * Set ``Client authentication`` to on
    * Only select ``Standard flow`` in ``Authentication flow``
    
#. Click ``Next``, add your redirect URL, e.g. ``https://gokapi.website.com/oauth-callback`` and click ``Save``

#. Click ``Credentials`` and note the ``Client Secret``


Addding a scope for exposing groups (optional)
*****************************************************

#. In the realm click on ``[Manage] Client Scopes`` and then ``Create Scope``

    * Name: groups
    * Type: Optional
    * Protocol: OpenID Connect
    * Click ``Save``
    
#. Click ``Mappers``

    * Click ``Add predefined mapper``
    * Search for ``groups`` and tick
    * Click ``Add``
    
#. In the realm click on ``[Manage] Clients`` and then ``gokapi-dev``

    * Click ``Client Scopes``
    * Click ``Add Client Scope``
    * Select ``groups`` and click ``Add / Optional``


Gokapi Configuration
""""""""""""""""""""""

+--------------------------+-----------------------------------------------------------+--------------------------------------------+
| Gokapi Configuration     | Input                                                     | Example                                    |
+==========================+===========================================================+============================================+
| Provider URL             | URL to Keycloak realm                                     | \http://keycloak.server.com/realms/master/ |
+--------------------------+-----------------------------------------------------------+--------------------------------------------+
| Client ID                | Client ID provided                                        | gokapi-dev                                 |
+--------------------------+-----------------------------------------------------------+--------------------------------------------+
| Client Secret            | Client secret provided                                    | AhXeV7_EXAMPLE_KEY                         |
+--------------------------+-----------------------------------------------------------+--------------------------------------------+
| Recheck identity         | If mode is ``pre-configured``, use a low interval.        | 12 hours                                   |
+--------------------------+-----------------------------------------------------------+--------------------------------------------+
| Restrict to user         | Check this if only certain users shall be allowed to      | checked                                    |
|                          |                                                           |                                            |
|                          | access Gokapi admin menu                                  |                                            |
+--------------------------+-----------------------------------------------------------+--------------------------------------------+
| Scope identifier (user)  | Use a scope that is unique to the user, e.g. the username | email                                      |
|                          |                                                           |                                            |
|                          | or the email                                              |                                            |
+--------------------------+-----------------------------------------------------------+--------------------------------------------+
| Authorised users         | Enter all users, separated by semicolon                   | \*\@company.com;admin\@othercompany.com    |
|                          |                                                           |                                            |
|                          | ``*`` can be used as a wildcard                           |                                            |
+--------------------------+-----------------------------------------------------------+--------------------------------------------+
| Restrict to group        | Check this if only users from certain groups shall be     | checked                                    |
|                          |                                                           |                                            |
|                          | allowed to access Gokapi admin menu                       |                                            |
+--------------------------+-----------------------------------------------------------+--------------------------------------------+
| Scope identifier (group) | Use a scope that lists the user's groups                  | groups                                     |
+--------------------------+-----------------------------------------------------------+--------------------------------------------+
| Authorised groups        | Enter all groups, separated by semicolon                  | dev;admins;gokapi-*                        |
|                          |                                                           |                                            |
|                          | ``*`` can be used as a wildcard                           |                                            |
+--------------------------+-----------------------------------------------------------+--------------------------------------------+


.. note::
   Logout through the Gokapi interface will not be possible anymore, unless the user logs out their Keycload account.
   


.. _oidcconfig_google:

Google
^^^^^^^^^^^^

Server Configuration
""""""""""""""""""""""

.. note::
   This guide has been last updated in January 2024 and is based on `this documentation <https://support.google.com/cloud/answer/6158849>`_
   
#. Go to the `Google Cloud Platform Console <https://console.cloud.google.com/>`_.
#. From the projects list, select a project or create a new one.
#. If the APIs & services page isn't already open, open the console left side menu and select APIs & services.
#. On the left, click Credentials.
#. Click New Credentials, then select OAuth client ID.
#. Select Application Type ``Webapplication``
#. Add the correct Gokapi redirect URL and click Create


Gokapi Configuration
""""""""""""""""""""""

+-------------------------+--------------------------------------------------+----------------------------------+
| Gokapi Configuration    | Input                                            | Example                          |
+=========================+==================================================+==================================+
| Provider URL            | \https://accounts.google.com                     | \https://accounts.google.com     |
+-------------------------+--------------------------------------------------+----------------------------------+
| Client ID               | Client ID provided                               | XXX.apps.googleusercontent.com   |
+-------------------------+--------------------------------------------------+----------------------------------+
| Client Secret           | Client secret provided                           | AhXeV7_EXAMPLE_KEY               |
+-------------------------+--------------------------------------------------+----------------------------------+
| Recheck identity        | Use a low interval.                              | 12 hours                         |
+-------------------------+--------------------------------------------------+----------------------------------+
| Restrict to user        | Check this, otherwise any Google user can access | checked                          |
|                         |                                                  |                                  |
|                         |                                                  |                                  |
|                         | your Gokapi admin menu                           |                                  |
+-------------------------+--------------------------------------------------+----------------------------------+
| Scope identifier (user) | email                                            | email                            |
+-------------------------+--------------------------------------------------+----------------------------------+
| Authorised users        | Enter all users, separated by semicolon          | user\@gmail.com;admin\@gmail.com |
+-------------------------+--------------------------------------------------+----------------------------------+
| Restrict to group       | Unsupported                                      | unchecked                        |
+-------------------------+--------------------------------------------------+----------------------------------+

