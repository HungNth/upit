# Upit

Upit is an on-demand file-upload product for interactive and automated use. Each invocation performs requested work and then exits.

## Language

**Global Configuration**:
The per-user document that selects the default Uploader, optional default Shortener, and default clipboard behavior for CLI and Manual Upload. File Manager Upload has a fixed clipboard contract.
_Avoid_: Global config, Main config, Settings

**Configuration Set**:
The per-user collection of the Global Configuration and the documents defining available Uploaders and optional Shorteners.
_Avoid_: Config, Config directory, Configuration files

**Uploader**:
A named definition of a remote upload destination, including how to send a file and identify the uploaded file URL in the response.
_Avoid_: Uploader profile, Custom uploader

**Request Body Mode**:
The Uploader rule that defines how a selected file and configured values are represented in the HTTP request body.
_Avoid_: Upload protocol, Body type

**Response Extractor**:
The Uploader rule that identifies one URL or provider error string from an HTTP response.
_Avoid_: Response parser, Output parser

**Shortener**:
A named definition of a remote URL-shortening destination, including how to submit an Original URL and identify the shortened URL in the response.
_Avoid_: URL shortener profile, Link provider

**Original URL**:
The uploaded file URL returned by an Uploader before optional post-processing.
_Avoid_: Source URL, Long URL

**Final URL**:
The URL emitted as the result of a successful upload after optional post-processing. It is identical to the Original URL when no post-processing changes it.
_Avoid_: Result URL, Output URL

**CLI**:
The public command-line interface of Upit for terminal and automated use.

**Product Version**:
The authoritative SemVer identity of the overall Upit release, shared across CLI, Desktop, private platform helpers, installers, and system metadata.
_Avoid_: Version, Config version, Launcher format

**Active Payload**:
The version-aligned set of Upit executables selected to serve new invocations on an installed system.
_Avoid_: Current folder, Launcher files

**Manual Upload**:
An interactive upload initiated by a user for exactly one local file selected through file browsing or drag-and-drop. It may override the Uploader, Shortener, clipboard behavior, and timeout for that upload, and exposes progress, cancellation, warnings, and the resulting Original URL and Final URL.
_Avoid_: Manual upload UI, GUI upload, Desktop upload

**File Manager Upload**:
An upload initiated from Windows File Explorer or macOS Finder for exactly one selected regular file. It uses the default Uploader and optional default Shortener from the Global Configuration, always attempts to copy the Final URL to the clipboard, and completes without opening Manual Upload.
_Avoid_: Explorer upload, Finder upload, Shell Upload, Direct Upload

**File Manager Integration**:
The installed operating-system capability that exposes File Manager Upload from a supported file manager and is managed through Upit Desktop as part of the Upit product.
_Avoid_: File manager app, Separate uploader app
