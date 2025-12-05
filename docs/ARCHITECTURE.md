
## Overview
```mermaid
flowchart LR
    A[Client submits job via API] --> B[API Server]
    
    B --> C[PostgreSQL]
    
    B --> D[Redis Queue]

    E[Worker] --> D
    E --> C
    E --> F[Process Job]

    F --> C

    G[Client / Dashboard] --> C


```
## Repository structure
```
goqueue/
├── cmd/
│   ├── api/           # API Server - separate binary
│   ├── worker/        # Worker Service - separate binary
│   ├── scheduler/     # Scheduler Service - separate binary
├── internal/          # Shared internal code
│   ├── broker/
│   ├── storage/
│   ├── job/
│   └── models/
├── pkg/               # Public client library
│   └── client/
├── deployments/
│   ├── docker-compose.yml  # Run all 4 services
│   └── k8s/               # Separate deployments for each
└── Makefile
```
