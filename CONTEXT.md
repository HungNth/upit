# Upit

Upit is an on-demand file-upload product for interactive and automated use. Each invocation performs requested work and then exits.

## Language

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
