# Media layout for Minio

`docker compose up -d` creates the public `provider-media` bucket and uploads every file from this directory.

The SVG previews and short looping MP4 clips are included. The video keys are:

- `core-loop.mp4`
- `ring-loop.mp4`
- `residential-loop.mp4`
- `draft-loop.mp4`

The backend stores only the separate `imageKey` and `videoKey`; it generates the public URL from `MINIO_PUBLIC_URL`.
