# TODO

## Feature: Guest Uploading
- [x] New admin page to manage guest uploading
  - [x] View active tokens
    - [x] Get links for guest uploads
    - [x] Also allow deleting
- [x] New guest upload page /guestupload
  - [x] Upload like admin
    - [x] Only accept if file isn't too big

## Remaining tasks for first release
- [x] Do a good check for security holes
- [x] Show the link to the uploaded file instead of immediately going to download page
- [ ] ~~Enable E2E encrypted uploading~~
  - [x] Remove E2E code from the guest upload page and JS
  - [x] Display notices that guest tokens cannot use E2E encryption
- [ ] Actually delete the token after it's been used
