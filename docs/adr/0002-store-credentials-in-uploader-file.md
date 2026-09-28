# Store credentials in the uploader configuration file

Upit stores uploader credentials directly in `custom-uploader.json` instead of adding environment-variable substitution or operating-system keychains. This keeps configuration portable for headless and automated use; Upit must never expose these values and must reject the file on Unix when group or other permissions are present, with guidance to set mode `0600`.
