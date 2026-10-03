# Always copy the File Manager Upload Final URL

File Manager Upload always attempts to copy its Final URL to the clipboard, regardless of the general Global Configuration clipboard preference, because the file-manager surface has no result view and copying is its primary result delivery. A clipboard failure preserves the completed upload as success with a warning and offers Copy Final URL against the existing result rather than retrying the upload; no separate File Manager clipboard setting is introduced.
