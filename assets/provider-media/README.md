# Media layout for Minio

`docker compose up -d` creates the public `provider-media` bucket and uploads every file from this directory.

The SVG/JPG previews and short looping MP4 clips are included. `provider_routers/draft.jpg` and `videos/draft-loop.mp4` are the default photo and video for new cards and drafts. The video keys are:

- `core-loop.mp4`
- `ring-loop.mp4`
- `residential-loop.mp4`
- `draft-loop.mp4`

The backend stores only the separate `imageKey` and `videoKey`; it generates the public URL from `MINIO_PUBLIC_URL`.
