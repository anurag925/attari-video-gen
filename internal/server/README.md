# API Server

Echo-based REST API for the video generation pipeline.

## Running

```bash
go run cmd/api/main.go -port 8080 -host localhost
```

## Endpoints

### Health
- `GET /health` - Health check

### Artifacts
- `GET /api/v1/artifacts` - List all artifacts
- `GET /api/v1/artifacts/:name` - Get artifact details
- `GET /api/v1/artifacts/:name/download` - Download artifact file
- `DELETE /api/v1/artifacts/:name` - Delete artifact

### Pipelines
- `GET /api/v1/pipelines` - List all pipelines
- `GET /api/v1/pipelines/:id` - Get pipeline details
- `POST /api/v1/pipelines` - Start a new pipeline
- `POST /api/v1/pipelines/:id/cancel` - Cancel a running pipeline
- `DELETE /api/v1/pipelines/:id` - Delete a pipeline

## API Examples

### Start a pipeline
```bash
curl -X POST http://localhost:8080/api/v1/pipelines \
  -H "Content-Type: application/json" \
  -d '{
    "video_url": "https://youtube.com/watch?v=...",
    "url": "https://example.com/article",
    "duration": 60,
    "output_name": "my-video.mp4"
  }'
```

### List pipelines
```bash
curl http://localhost:8080/api/v1/pipelines
```

### Get pipeline status
```bash
curl http://localhost:8080/api/v1/pipelines/:id
```

### List artifacts
```bash
curl http://localhost:8080/api/v1/artifacts
```

## Next Steps

- [ ] Implement pipeline execution (connect to existing processor)
- [ ] Add WebSocket support for real-time progress updates
- [ ] Add persistence (database) for store
- [ ] Add authentication
- [ ] Add pagination for list endpoints
- [ ] Add filtering for artifacts by type/pipeline