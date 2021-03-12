# Gokapi


### Available for:

- Bare Metal
- [Docker](https://hub.docker.com/repository/docker/f0rc3/gokapi)

## About

Gokapi is a lightweight server to share files, which expire after a set amount of downloads or days. It is similar to the discontinued [Firefox Send](https://github.com/mozilla/send), with the difference that only the admin is allowed to upload files. 

This enables companies or individuals to share their files very easily and having them removed afterwards, therefore saving disk space and having control over who downloads the file from the server.

The project is very new, but can already be used in production. Customization is very easy with HTML/CSS knowledge. Identical files will be deduplicated.

## Installation

### Bare Metal

Simply download the latest release for your platform and execute the binary (recommended). If you want to compile the source yourself, clone this repository and execute `go build`.

### Docker

Run the following command to create the container, volumes and execute the initial setup: `docker run -it -v gokapi-data:/app/data -v gokapi-config:/app/config -p 127.0.0.1:53842:53842 f0rc3/gokapi:latest`. Please note the `-it` docker argument, which is required for the first start!

With the argument `-p 127.0.0.1:53842:53842` the service will only be accessible from the machine it is running on. Normally you will use a reverse proxy to enable SSL - if you want to make the service availabe to other computers in the network without a reverse proxy, replace the argument with `-p 127.0.0.1:53842:53842`. Please note that traffic will **not** be encypted that way and data like passwords and transferred files can easily be read by 3rd parties!

## Usage

### First start

On the first start you will be prompted for the initial configuration.

* Username: Enter the name for the admin user name (who can upload files)
* Password: This will be used to enter the admin page
* Server URL: The external URL for the Gokapi server. Hosting it with a reverse proxy and SSL is strongly recommended! For testing purposes you can enter `http://127.0.0.1:53842/`
* Index URL: The URL where the index page redirects to. Leave blank to have it redirect to the Gokapi GitHub page
* Bind port to localhost: If you choose yes, you can only access Gokapi on the machine or by using a reverse proxy. Strongly recommended! Not displayed when deployed with Docker.

Then you can navigate to `http://127.0.0.1:53842/admin` in your browser and login with the credentials.

### Uploading

To upload, drag and drop a file, folder or multiple files to the Upload Zone. If you want to change the default expiry conditions, this has to be done before uploading. For each file an entry in the table will appear with a download link. You can also delete files on this screen.


### Customizing

By default, all files are included in the executable. If you want to change the layout (e.g. add your company logo or change the app name etc.), follow these steps:
* Clone this repository
* Copy either the folder `static`, `templates` or both to the directory where the executable is located
* Make changes to the folders. `static` contains images, CSS files and JavaScript. `templates` contains the HTML code.
* Restart the server. If the folders exist, the server will use the local files instead of the embedded files
* Optional: To embed the files permanently, the executable needs to be recompiled with `go build`.


## Screenshots
Admin Menu![image](https://user-images.githubusercontent.com/1593467/110936500-45a4da80-8331-11eb-8a2d-986af5ab411a.png)
Download Link![image](https://user-images.githubusercontent.com/1593467/110936659-869cef00-8331-11eb-83d8-7c2837f55620.png)



## Contributors
<a href="https://github.com/forceu/barcodebuddy/graphs/contributors">
  <img src="https://contributors-img.web.app/image?repo=forceu/gokapi" />
</a>

## License

This project is licensed under the GNU GPL3 - see the [LICENSE.md](LICENSE.md) file for details


## Donations

As with all Free software, the power is less in the finances and more in the collective efforts. I really appreciate every pull request and bug report offered up by our users! If however, you're not one for coding/design/documentation, and would like to contribute financially, you can do so with the link below. Every help is very much appreciated!

[![paypal](https://img.shields.io/badge/Donate-PayPal-green.svg)](https://www.paypal.com/cgi-bin/webscr?cmd=_donations&business=donate@bulling.mobi&lc=US&item_name=BarcodeBuddy&no_note=0&cn=&currency_code=EUR&bn=PP-DonationsBF:btn_donateCC_LG.gif:NonHosted) [![LiberaPay](https://img.shields.io/badge/Donate-LiberaPay-green.svg)](https://liberapay.com/MBulling/donate)


