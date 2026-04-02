# Folder Storage Backend

This backend stores buckets and objects directly on a local filesystem tree.
It is designed for local/self-hosted usage and prioritizes straightforward behavior and path safety.

## Scope

- Implements `core.StorageBackend` and bucket/object operations used by the S3 API layer.
- Uses filesystem directories/files as the source of truth.
- Uses a distributed lock service (`core.Locker`, currently Redis-based) for write-side coordination.

## Layout goals

This storage layout intentionally optimizes for two operational goals:

1. **Self-contained backups**: bucket data and metadata live together in the filesystem tree, so operators can back up raw backend files without requiring a separate metadata database backup.
2. **Easy conversion to a normal file tree**: object payloads are stored as regular files (`blob`) under key-shaped directories, with metadata in adjacent YAML files, so conversion/export to a plain filesystem hierarchy is straightforward.

## High-level architecture

Main components in this package:

- `Backend` in `backend.go`: backend init, bucket lifecycle (`ListBuckets`, `CreateBucket`, `DeleteBucket`, `HeadBucket`).
- `Bucket` in `bucket.go`: object CRUD, copy, tagging, multipart lifecycle, list operations.
- `Object` in `object.go`: lazy file reader for object blob + metadata view.
- `Config` in `config.go`: canonical path building and containment checks (`EnsureContained`).
- Symlink-safe filesystem helpers in `symlink.go`: no-follow open/create/mkdir/rename operations.
- Walkers in `walker.go`: prefix + marker traversal for object and multipart listings.

## Request flow (write path)

Current behavior for `PutObject`:

1. Resolve final object path (`buckets/<bucket>/objects/<key>`), validate containment.
2. Acquire lock on that object path via `Locker.Lock`.
3. Reject symlinks in existing path components.
4. Stream body to a temp upload directory (`uploads/regular/<uuid>/blob`) while computing SHA-256.
5. Validate checksum (`metadata.SHA256`) and write `metadata.yaml`.
6. Ensure parent directories exist.
7. Atomically move temp upload directory to final object path (`renameNoFollow`).

`CopyObject` and `CompleteMultipartUpload` follow the same stage-then-rename pattern.

## On-disk data model

Root is `FOLDER_STORAGE_BACKEND_PATH` (default `./d3_data`):

- `d3.yaml`: backend config version marker.
- `buckets/<bucket>/bucket.yaml`: bucket metadata (`creationDate`).
- `buckets/<bucket>/objects/<key>/blob`: object payload.
- `buckets/<bucket>/objects/<key>/metadata.yaml`: object metadata (size, checksums, tags, etc.).
- `buckets/<bucket>/uploads/regular/<uuid>/...`: temporary single-part upload staging.
- `buckets/<bucket>/uploads/multipart/<key>/<uploadID>/...`: multipart staging area.
- `tmp/bin/<uuid>`: tombstoned/deleted objects are renamed here before cleanup.

Object keys are mapped as nested directories. Path separators are normalized with `filepath` logic; multipart key extraction normalizes to forward slashes (`filepath.ToSlash`).

## Concurrency and locking model

- Write operations on a specific object path typically take a lock keyed by that path:
  - `PutObject`, `CopyObject`, `UploadPart(part path)`, `PutObjectTagging`, `DeleteObjectTagging`.
- Backend initialization takes a global init lock (`folder-storage-backend-init`).
- Multipart create/list/complete/abort currently do not use a single upload-wide lock; safety is mostly from path checks and operation ordering.
- Read operations (`GetObject`, `HeadObject`, listing) do not take locks.

Practical implication: concurrent writes to the same object are serialized where lock coverage exists, but lock semantics depend on external Redis availability and behavior.

## Consistency and durability

### Current guarantees

- **Object publish is rename-based**: writes happen in a temp directory and become visible at once when renamed into place.
- **Readers should not observe partial object blobs** at final object path in normal operation, because final placement is a directory rename.
- **Delete is rename-based**: object directory is moved to `tmp/bin` first, then parent dirs may be best-effort cleaned.
- **Checksum validation on upload**: mismatch fails the operation.

### Non-guarantees (current behavior)

- No explicit `fsync` of object blob, metadata file, or parent directories before/after rename.
  - Crash/power-loss durability is therefore filesystem-dependent.
- Metadata writes (`yaml.MarshalToFile`) use `os.WriteFile` directly (no atomic temp-file swap for metadata-only updates such as tagging).
- No transactional guarantee across multiple files beyond what rename provides.
- No cross-operation snapshot isolation for listings/reads during concurrent writes/deletes.
- Multipart complete does not hold a global lock for the upload; concurrent part mutations can still race at higher level.

## Checksum and integrity behavior

- `PutObject` computes SHA-256 while streaming and compares with `input.Metadata.SHA256`.
  - If client uses streaming signature marker, backend replaces marker with computed checksum.
  - On mismatch, returns `core.ErrObjectChecksumMismatch`.
- `metadata.yaml` stores:
  - `SHA256` (hex)
  - `SHA256Base64`
  - `Size`
  - `LastModified`
  - tags/custom metadata/content type
- Multipart:
  - Each uploaded part stores checksum in `part-<n>.yaml`.
  - `CompleteMultipartUpload` validates provided part ETags, concatenates parts, recomputes final SHA-256, writes final metadata, then renames upload dir to object path.

## Error semantics (developer/operator relevant)

Common backend errors from `internal/core/errors.go`:

- `ErrBucketAlreadyExists`, `ErrBucketNotFound`, `ErrBucketNotEmpty`
- `ErrObjectNotFound`, `ErrObjectAlreadyExists`
- `ErrObjectChecksumMismatch`
- `ErrPreconditionFailed` (`If-None-Match` behavior)
- `ErrInvalidUploadID`
- `ErrPathTraversal` (containment check failure)
- `ErrSymlinkNotAllowed` (symlink in relevant path components)
- `ErrObjectMetadataNotReadable`

API layer maps these to HTTP errors in `internal/apis/s3/middlewares/error_renderer.go` (for example: not found -> 404, conflicts -> 409, checksum/path/symlink issues -> 400).

## Filesystem requirements and assumptions

The backend assumes the underlying filesystem supports:

- **Atomic rename within same filesystem** for directories/files used by stage-then-publish flow.
  - Keep `FOLDER_STORAGE_BACKEND_PATH` and its temp/upload paths on the same mount.
- **Stable hard links for copy** (`CopyObject` uses `os.Link` from source blob to staged destination blob).
  - Cross-filesystem hard links are not supported.
- **Unix no-follow semantics** (`O_NOFOLLOW`, `openat`, `renameat`) because `symlink.go` is `//go:build unix`.
- **Case-sensitive path behavior is strongly recommended**.
  - Case-insensitive filesystems may collapse distinct S3 keys unexpectedly.
- **Permission to create/read/write/remove directories and files** under backend root.
- **Reasonable timestamp behavior** for metadata fields and fallback creation date (`mtime` for legacy buckets without `bucket.yaml`).
- **Path separator normalization handled by backend** (`filepath` + slash normalization where needed), but mixed-platform sharing can still be tricky.

Symlink policy:

- Symlinks are explicitly rejected in object/bucket/upload paths used by critical operations.
- This is both a security control (path escape prevention) and a consistency requirement.

## Operational caveats

- Network/distributed filesystems (NFS/SMB/FUSE/object gateways) may weaken rename atomicity, locking expectations, timestamp behavior, and visibility timing.
- Redis lock service is part of correctness for concurrent writers; lock outages can degrade write serialization.
- Background cleanup of `tmp/bin` is not implemented in this package; operators should monitor disk usage.
- Large-directory performance depends on filesystem characteristics and walk costs.

## Future improvements

- Add explicit file and directory `fsync` on write/rename critical paths for stronger crash durability.
- Use atomic write pattern for metadata-only updates (tags/metadata changes).
- Add upload-scoped locking for multipart complete/abort/part upload coordination.
- Add optional bin garbage collector and operational tooling.
