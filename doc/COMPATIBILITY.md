# d3 and Amazon S3 API compatibility

This document is for **integrators and operators** who need to know how closely [d3](https://github.com/zhulik/d3)’s HTTP surface matches the [Amazon S3 REST API](https://docs.aws.amazon.com/AmazonS3/latest/API/Welcome.html). It focuses on commonly used operations, notable request features, and how **authentication** and **authorization** work in d3.

Implementation evidence is drawn from `internal/apis/s3/api_objects.go`, `internal/apis/s3/api_buckets.go`, `internal/apis/s3/middlewares/authenticator.go`, `internal/apis/s3/middlewares/authorizer.go`, `internal/apis/s3/auth/authorizer.go`, `pkg/s3actions`, and the folder storage backend (`internal/backends/storage/folder/backend.go`). **d3 is not a full Amazon S3 emulator**: many bucket- and account-level features present in AWS are intentionally absent.

---

## Scope summary


| Area               | Amazon S3 (reference)                                                                         | d3                                                                                                  |
| ------------------ | --------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| API style          | REST, XML bodies for many operations; SigV4 signing                                           | Same general patterns for implemented routes; XML where S3 uses XML                                 |
| API version string | Service uses `2006-03-01`                                                                     | Compatible request shapes for supported operations; not a guarantee of identical error XML or codes |
| Unsupported in d3  | ACLs, versioning, replication, notifications, most bucket subresources, KMS/SSE options, etc. | No routes/handlers for those features (see tables below)                                            |


---

## Bucket operations


| Operation (AWS name)  | HTTP shape (typical)     | d3 support    | Notes                                                                                                |
| --------------------- | ------------------------ | ------------- | ---------------------------------------------------------------------------------------------------- |
| **ListBuckets**       | `GET /`                  | **Supported** | XML listing; buckets include name, creation date, region, ARN from `core.Bucket` (`api_buckets.go`). |
| **CreateBucket**      | `PUT /{bucket}`          | **Supported** | Response sets `Location` and `x-amz-bucket-arn`. Backend creates a directory (`folder/backend.go`).  |
| **DeleteBucket**      | `DELETE /{bucket}`       | **Supported** | Empty bucket required (`ErrBucketNotEmpty` → HTTP 400 via error middleware).                         |
| **HeadBucket**        | `HEAD /{bucket}`         | **Supported** | Sets `x-amz-bucket-arn`, `x-amz-bucket-region`.                                                      |
| **GetBucketLocation** | `GET /{bucket}?location` | **Supported** | XML `LocationConstraint` from bucket region.                                                         |


---

## Object operations


| Operation (AWS name)    | HTTP shape (typical)                        | d3 support    | Notes                                                                                                                                                                                                                                                                                                                 |
| ----------------------- | ------------------------------------------- | ------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **ListObjectsV2**       | `GET /{bucket}?list-type=2&…`               | **Partial**   | Supports `prefix`, `delimiter`, `max-keys` (1…1000, default 1000), `continuation-token`. **No** `start-after`, `encoding-type`, `fetch-owner`, etc.                                                                                                                                                                   |
| **ListObjects** (v1)    | `GET /{bucket}?prefix=…&marker=…`           | **Partial**   | Implemented via v2 backend + marker/continuation mapping (`listObjectsV1Response` in `api_objects.go`).                                                                                                                                                                                                               |
| **GetObject**           | `GET /{bucket}/{key}`                       | **Partial**   | `Range` with `206` + `Content-Range`; conditional headers (see below); streams body.                                                                                                                                                                                                                                  |
| **HeadObject**          | `HEAD /{bucket}/{key}`                      | **Partial**   | Returns metadata headers; **does not** evaluate `If-*` conditionals (unlike `GetObject`).                                                                                                                                                                                                                             |
| **PutObject**           | `PUT /{bucket}/{key}`                       | **Partial**   | Body, `Content-Type`, `x-amz-meta-*`, `X-Amz-Tagging`, `X-Amz-Content-Sha256` (including streaming chunked `STREAMING-AWS4-HMAC-SHA256-PAYLOAD`). Conditional writes: `If-None-Match: *` and related behavior via `putObjectConditional`.                                                                             |
| **DeleteObject**        | `DELETE /{bucket}/{key}`                    | **Supported** | S3-compatible `204` semantics for missing keys (delete path uses `DeleteObjects` with a single key).                                                                                                                                                                                                                  |
| **DeleteObjects**       | `POST /{bucket}?delete`                     | **Partial**   | XML body; up to **1000** keys; `Quiet` respected; per-key errors use `NoSuchKey` / `InternalError` in XML.                                                                                                                                                                                                            |
| **GetObjectTagging**    | `GET /{bucket}/{key}?tagging`               | **Supported** | XML `TagSet`.                                                                                                                                                                                                                                                                                                         |
| **PutObjectTagging**    | `PUT /{bucket}/{key}?tagging`               | **Supported** | XML body (size-limited); tag count/length limits aligned with S3 (10 tags, key/value length checks).                                                                                                                                                                                                                  |
| **DeleteObjectTagging** | `DELETE /{bucket}/{key}?tagging`            | **Supported** |                                                                                                                                                                                                                                                                                                                       |
| **CopyObject** (effect) | `PUT /{bucket}/{key}` + `x-amz-copy-source` | **Partial**   | Implemented inside `PutObject`; checks **GetObject** on source for authorization. Supports metadata/tagging directives and `If-None-Match: *` for create-only copy. **IAM action** for policy checks on the destination is still `**s3:PutObject`** (there is no separate `s3:CopyObject` action in `pkg/s3actions`). |


---

## Multipart upload


| Operation (AWS name)        | HTTP shape (typical)           | d3 support    | Notes                                                                                                             |
| --------------------------- | ------------------------------ | ------------- | ----------------------------------------------------------------------------------------------------------------- |
| **CreateMultipartUpload**   | `POST /{bucket}/{key}?uploads` | **Supported** | Headers for content type, tagging, `x-amz-meta-*`.                                                                |
| **UploadPart**              | `PUT …?partNumber=&uploadId=`  | **Supported** | Returns `ETag` header.                                                                                            |
| **CompleteMultipartUpload** | `POST …?uploadId=`             | **Supported** | XML parts list; validates part numbers and ETags.                                                                 |
| **AbortMultipartUpload**    | `DELETE …?uploadId=`           | **Supported** |                                                                                                                   |
| **ListParts**               | `GET …?uploadId=`              | **Supported** | `max-parts`, `part-number-marker` (limits per `core.MaxParts`). Owner/initiator populated when a user is present. |
| **ListMultipartUploads**    | `GET /{bucket}?uploads`        | **Partial**   | `prefix`, `delimiter`, `max-uploads`, markers; aligns with backend pagination (`core.MaxUploads`).                |


---

## Headers, checksums, and metadata


| Feature                                                                 | Amazon S3 (typical)                                                     | d3 behavior                                                                                                                                                          |
| ----------------------------------------------------------------------- | ----------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **ETag**                                                                | Often MD5 for simple PUT; multipart varies                              | Object **ETag** is the **SHA256 hex** string; `**x-amz-checksum-sha256`** (base64) is also set on responses (`setObjectHeaders`). Clients must not assume MD5 ETags. |
| **Conditional GET**                                                     | `If-Match`, `If-None-Match`, `If-Modified-Since`, `If-Unmodified-Since` | **GetObject**: evaluated in `conditionalheaders.Parse` / `Check` (`pkg/conditionalheaders`).                                                                         |
| **Conditional PUT**                                                     | Varies by operation                                                     | **PutObject**: supports `If-None-Match: *` for create-only; other combinations use `HeadObject` + `Check` (412 / 404 as applicable).                                 |
| **User metadata**                                                       | `x-amz-meta-*`                                                          | Stored and returned (keys lowercased in `parseMeta`).                                                                                                                |
| **Object tags**                                                         | Header or tagging APIs                                                  | `X-Amz-Tagging` on PUT/create multipart; XML for `PutObjectTagging`.                                                                                                 |
| **Server-side encryption, ACLs, Object Lock, website, CORS, lifecycle** | Extensive API surface                                                   | **Not implemented** (no handlers in `internal/apis/s3`).                                                                                                             |


---

## Authentication


| Topic                     | Amazon S3                                                                                                                                   | d3                                                                                                                                                                                                                   |
| ------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Request signing**       | [AWS Signature Version 4](https://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-authenticating-requests.html) for authenticated REST calls | `**sigv4.Validate`** on incoming requests (`internal/apis/s3/middlewares/authenticator.go`). Invalid signature / credential issues map to **403 Forbidden** (`middlewares/error_renderer.go` + `pkg/sigv4` errors).  |
| **Access keys**           | IAM user keys, STS, etc.                                                                                                                    | Users stored via **management API**; each user has `AccessKeyID` / `SecretAccessKey` (`internal/apis/management/api_users.go`).                                                                                      |
| **Unsigned requests**     | Allowed for public/anonymous access where policy permits                                                                                    | Unsigned requests **do not** fail signature validation, but `**Authorizer` denies** when `user == nil` (anonymous access is effectively **not** implemented yet; see TODO in `internal/apis/s3/auth/authorizer.go`). |
| **Management API bodies** | N/A (not S3)                                                                                                                                | JSON requests require `**X-Amz-Content-Sha256`** matching the body hash (`validateBodyChecksumAndParseJSON`, `api_users.go`, `api_bindings.go`); policies use the same header (`api_policies.go`).                   |


---

## Authorization


| Topic                            | Amazon S3                           | d3                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| -------------------------------- | ----------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Policy model**                 | IAM policies, bucket policies, ACLs | **IAM-style JSON policies** parsed by `pkg/iampol`, stored via management API (`api_policies.go`), attached to users via **bindings** (`api_bindings.go`).                                                                                                                                                                                                                                                                            |
| **Actions**                      | Fine-grained `s3:*` actions         | Subset in `pkg/s3actions` (e.g. `s3:GetObject`, `s3:PutObject`, `s3:ListBuckets`, multipart and tagging actions).                                                                                                                                                                                                                                                                                                                     |
| **Resources**                    | ARNs, `*`                           | Statements use `arn:aws:s3:::**{pattern}`** where the suffix matches **bucket name** or `**bucket/key`** (`internal/apis/s3/auth/authorizer.go`); wildcards via `pkg/wld`.                                                                                                                                                                                                                                                            |
| **Resource for PUT-style calls** | Object-level policies apply per key | `**PutObject`**, `**CreateMultipartUpload**`, `**UploadPart**`, and `**CompleteMultipartUpload**` run **without** `ObjectFinder`, so the authorizer sees `**resource = bucket` only** (no `bucket/key` suffix). Prefix/object-level ARN patterns do **not** apply to those actions in the middleware. `**GetObject`**, `**HeadObject**`, `**DeleteObject**`, and tagging routes use object resolution and can match `**bucket/key**`. |
| **Deny vs Allow**                | Explicit deny wins                  | Same: **Deny** evaluated first, then **Allow** (`authorizer.go`).                                                                                                                                                                                                                                                                                                                                                                     |
| **Admin user**                   | N/A                                 | User named `**admin`** bypasses policy checks (`authorizer.go`).                                                                                                                                                                                                                                                                                                                                                                      |
| **Copy authorization**           | Read source, write dest             | Destination action `**s3:PutObject`**; **additional** `GetObject` check on **source** key in `CopyObject` (`api_objects.go`).                                                                                                                                                                                                                                                                                                         |
| **HTTP status when denied**      | Often `403 AccessDenied`            | `**401 Unauthorized`** for policy denial (`core.ErrUnauthorized` → `api_objects.go` / middleware).                                                                                                                                                                                                                                                                                                                                    |
| **Management API**               | IAM / AWS APIs                      | **Only `admin`** may call management routes (`internal/apis/management/middlewares/authorizer.go`).                                                                                                                                                                                                                                                                                                                                   |


---

## Management API (not Amazon S3)

These endpoints are **d3-specific**; they do **not** mirror an AWS S3 REST operation. They exist to configure users, policies, and bindings used by SigV4 and the S3 authorizer.


| Area         | Endpoints (summary)                                                                                                                                     | Purpose                                                           |
| ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| **Users**    | `GET/POST /users`, `PUT/DELETE /users/:userName`                                                                                                        | Create/list/update/delete users and rotate keys (`api_users.go`). |
| **Policies** | `GET /policies`, `GET/PUT/DELETE /policies/:policyID`, `POST /policies`                                                                                 | CRUD IAM-compatible policy documents (`api_policies.go`).         |
| **Bindings** | `GET /bindings`, `GET /bindings/user/:userName`, `GET /bindings/policy/:policyID`, `POST /bindings`, `DELETE /bindings/user/:userName/policy/:policyID` | Attach policies to users (`api_bindings.go`).                     |


---

## Storage backend note

The **folder** backend maps buckets and objects to directories and files on disk (`internal/backends/storage/folder/backend.go`). Compatibility statements above describe the **HTTP API**; durability, concurrency, and filesystem edge cases are backend-dependent.

---

## References (AWS)

- [Amazon S3 API Reference — Welcome](https://docs.aws.amazon.com/AmazonS3/latest/API/Welcome.html)
- [Making requests using the REST API](https://docs.aws.amazon.com/AmazonS3/latest/API/RESTAPI.html)
- [Authenticating Requests (AWS Signature Version 4)](https://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-authenticating-requests.html)
- [S3 API Reference index](https://docs.aws.amazon.com/AmazonS3/latest/API/Type_API_Reference.html)

## References (d3 source)

- `internal/apis/s3/api_objects.go` — object and multipart routes and behavior
- `internal/apis/s3/api_buckets.go` — bucket routes and responses
- `internal/apis/s3/middlewares/authenticator.go`, `authorizer.go`, `error_renderer.go` — authn/z and HTTP error mapping
- `internal/apis/s3/auth/authorizer.go` — policy evaluation
- `pkg/s3actions` — supported action constants for policies
- `internal/apis/management/api_users.go`, `api_policies.go`, `api_bindings.go` — management plane
- `internal/backends/storage/folder/backend.go` — folder storage semantics

